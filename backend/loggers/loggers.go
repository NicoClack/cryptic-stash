package loggers

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"os"
	"reflect"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/messengers"
	"github.com/NicoClack/cryptic-stash/backend/ratelimiting"
	"github.com/google/uuid"
	"github.com/lmittmann/tint"
)

const (
	// Special attributes
	PublicMessageKey = "publicMessage"
	UserIDKey        = "userID"

	MaxSaveBatchSize = 100
	ShutdownTimeout  = 5 * time.Second
)

type disableErrorLoggingKey = struct{} // Used to prevent infinite loops

//nolint:recvcheck
type Handler struct {
	App              *common.App
	Level            slog.Level
	SaveToDatabase   bool
	ShouldPrint      bool
	tintHandler      slog.Handler
	baseAttrs        map[string]any
	baseSpecialProps specialProperties
	baseGroups       []string
	topHandler       *topHandler
	mu               *sync.RWMutex
}
type topHandler struct {
	entryChan           chan *entry
	requestShutdownChan chan struct{}
	shutdownCtx         context.Context
	cancelShutdownCtx   context.CancelFunc
	listenOnce          sync.Once
	shutdownOnce        sync.Once
	// Note: the channels and sync.Onces are assumed to be constant once created
	// shutdownCtx and cancelShutdownCtx are only set during shutdown
	mu sync.RWMutex
}
type entry struct {
	time                         time.Time
	timeKnown                    bool
	level                        int
	message                      string
	attributes                   map[string]any
	sourceFile                   string
	sourceFunction               string
	sourceLine                   int
	publicMessage                string
	userID                       uuid.UUID
	disableErrorLogging          bool
	useAdminNotificationFallback bool
	disableAdminNotification     bool
}

func NewHandler(
	level slog.Level, saveToDatabase bool, shouldPrint bool,
	app *common.App,
) Handler {
	return Handler{
		App:            app,
		Level:          level,
		SaveToDatabase: saveToDatabase,
		ShouldPrint:    shouldPrint,
		tintHandler: tint.NewHandler(os.Stdout, &tint.Options{
			Level:      level,
			AddSource:  true,
			TimeFormat: time.TimeOnly,
		}),
		baseAttrs: map[string]any{},
		topHandler: &topHandler{
			entryChan:           make(chan *entry, 100),
			requestShutdownChan: make(chan struct{}),
			listenOnce:          sync.Once{},
			shutdownOnce:        sync.Once{},
			mu:                  sync.RWMutex{},
		},
		mu: &sync.RWMutex{},
	}
}

func (handler *Handler) Listen() {
	handler.topHandler.listenOnce.Do(func() {
		loggedBulkWarning := false
		loggedAdminNotificationError := false
		entries := []*entry{}
	listenLoop:
		for {
			shouldReEnableSelfLogging := false
			select {
			case entry := <-handler.topHandler.entryChan:
				entries = append(entries, entry)
			case <-handler.topHandler.requestShutdownChan:
				break listenLoop
			}

			timeoutChan := handler.App.Clock.After(handler.App.Env.LOG_STORE_INTERVAL)
		collectBatchLoop:
			for {
				select {
				case entry := <-handler.topHandler.entryChan:
					entries = append(entries, entry)
				case <-timeoutChan:
					shouldReEnableSelfLogging = true
					break collectBatchLoop
				case <-handler.topHandler.requestShutdownChan:
					break listenLoop
				}
				if len(entries) >= MaxSaveBatchSize {
					break collectBatchLoop
				}
			}

			if len(entries) > 0 {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				var bulkWriteErr error
				func() {
					defer cancel()
					bulkWriteErr = handler.bulkWrite(entries, ctx)
				}()
				if bulkWriteErr != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					var selfLoggedNow bool
					func() {
						defer cancel()
						selfLoggedNow = handler.individualWriteFallback(
							entries, bulkWriteErr, &loggedBulkWarning, ctx,
						)
					}()
					if selfLoggedNow {
						shouldReEnableSelfLogging = false
					}
				}
				if handler.maybeNotifyAdmin(entries, &loggedAdminNotificationError) {
					shouldReEnableSelfLogging = false
				}
				entries = []*entry{}
			}

			if shouldReEnableSelfLogging {
				loggedBulkWarning = false
				loggedAdminNotificationError = false
			}
		}
		close(handler.topHandler.requestShutdownChan)

		handler.topHandler.mu.RLock()
		shutdownCtx := handler.topHandler.shutdownCtx
		handler.topHandler.mu.RUnlock()
	shutdownLoop:
		for {
			selfLogged := false
		drainLoop:
			for {
				select {
				case entry := <-handler.topHandler.entryChan:
					entries = append(entries, entry)
				case <-shutdownCtx.Done():
					break shutdownLoop
				default:
					break drainLoop
				}
			}
			if len(entries) > 0 {
				bulkWriteErr := handler.bulkWrite(entries, shutdownCtx)
				if bulkWriteErr != nil {
					if handler.individualWriteFallback(
						entries, bulkWriteErr, &loggedBulkWarning, shutdownCtx,
					) {
						selfLogged = true
					}
				}
				if handler.maybeNotifyAdmin(entries, &loggedAdminNotificationError) {
					selfLogged = true
				}
				entries = []*entry{}
			}

			if !selfLogged {
				break
			}
		}

		handler.topHandler.mu.Lock()
		handler.topHandler.cancelShutdownCtx()
		handler.topHandler.mu.Unlock()
	})
}

func (handler *Handler) bulkWrite(entries []*entry, ctx context.Context) error {
	return dbcommon.WithWriteTx(
		ctx, handler.App.Database,
		func(tx *ent.Tx, ctx context.Context) error {
			return tx.LogEntry.MapCreateBulk(entries, func(logEntryCreate *ent.LogEntryCreate, i int) {
				entry := entries[i]
				logEntryCreate.SetLoggedAt(entry.time).SetLoggedAtKnown(entry.timeKnown).
					SetLevel(entry.level).
					SetMessage(entry.message).
					SetAttributes(entry.attributes).
					SetSourceFile(entry.sourceFile).
					SetSourceFunction(entry.sourceFunction).
					SetSourceLine(entry.sourceLine).
					SetPublicMessage(entry.publicMessage)
				if entry.userID != uuid.Nil {
					logEntryCreate.SetUserID(entry.userID) // Nullable
				}
			}).Exec(ctx)
		},
	)
}

func (handler *Handler) individualWriteFallback(
	entries []*entry,
	bulkWriteErr error,
	loggedBulkWarningPtr *bool,
	ctx context.Context,
) bool {
	selfLogged := false
	allSucceeded := true
	for _, entry := range entries {
		var timeout time.Duration
		switch {
		case entry.level >= int(slog.LevelError):
			timeout = time.Second
		case entry.level >= int(slog.LevelWarn):
			timeout = 500 * time.Millisecond
		default:
			timeout = 100 * time.Millisecond
		}
		individualCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		entryID, stdErr := dbcommon.WithReadWriteTx(
			individualCtx, handler.App.Database,
			func(tx *ent.Tx, ctx context.Context) (uuid.UUID, error) {
				ob, stdErr := tx.LogEntry.Create().
					SetLoggedAt(entry.time).SetLoggedAtKnown(entry.timeKnown).
					SetLevel(entry.level).
					SetMessage(entry.message).
					SetAttributes(entry.attributes).
					SetSourceFile(entry.sourceFile).
					SetSourceFunction(entry.sourceFunction).
					SetSourceLine(entry.sourceLine).
					SetPublicMessage(entry.publicMessage).
					// UserID is hydrated later in case it was the cause of the original error
					Save(ctx)
				if stdErr != nil {
					return uuid.Nil, stdErr
				}
				return ob.ID, stdErr
			},
		)
		if stdErr != nil {
			cancel()
			allSucceeded = false
			if !entry.disableErrorLogging {
				pc, _, _, _ := runtime.Caller(0)
				record := slog.NewRecord(
					handler.App.Clock.Now(),
					slog.LevelError,
					"failed to write log entry to database",
					pc,
				)
				record.AddAttrs(slog.Any("log", entry))
				record.AddAttrs(slog.Any("error", stdErr))
				//nolint:contextcheck // logging is a different context to the code that created the original log
				_ = handler.Handle(
					context.WithValue(context.Background(), disableErrorLoggingKey{}, true),
					record,
				)
				selfLogged = true
			}
			continue
		}
		if entry.userID == uuid.Nil {
			cancel()
			continue
		}
		stdErr = dbcommon.WithWriteTx(
			individualCtx, handler.App.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				return tx.LogEntry.UpdateOneID(entryID).SetUserID(entry.userID).Exec(ctx)
			},
		)
		cancel()
		if stdErr != nil {
			if common.IsErrorType[*ent.ConstraintError](stdErr) {
				pc, _, _, _ := runtime.Caller(0)
				record := slog.NewRecord(
					handler.App.Clock.Now(),
					slog.LevelWarn,
					"couldn't find user with ID provided in log statement",
					pc,
				)
				record.AddAttrs(slog.Any("log", entry))
				record.AddAttrs(slog.Any("error", stdErr))
				//nolint:contextcheck // logging is a different context to the code that created the original log
				_ = handler.Handle(
					context.WithValue(context.Background(), disableErrorLoggingKey{}, true),
					record,
				)
				selfLogged = true
			} else {
				pc, _, _, _ := runtime.Caller(0)
				record := slog.NewRecord(
					handler.App.Clock.Now(),
					slog.LevelError,
					"couldn't set UserID field on log statement",
					pc,
				)
				record.AddAttrs(slog.Any("log", entry))
				record.AddAttrs(slog.Any("error", stdErr))
				//nolint:contextcheck // logging is a different context to the code that created the original log
				_ = handler.Handle(
					context.WithValue(context.Background(), disableErrorLoggingKey{}, true),
					record,
				)
				selfLogged = true
			}
			allSucceeded = false
			continue
		}
	}
	if allSucceeded && !*loggedBulkWarningPtr {
		pc, _, _, _ := runtime.Caller(0)
		record := slog.NewRecord(
			handler.App.Clock.Now(),
			slog.LevelWarn,
			"bulk log write failed but the individual fallback writes all succeeded, "+
				"so the writes took longer than they should have",
			pc,
		)
		record.AddAttrs(
			slog.Any("error", bulkWriteErr),
			slog.Any("entryCount", len(entries)),
		)
		//nolint:contextcheck // logging is a different context to the code that created the original log
		_ = handler.Handle(
			context.Background(),
			record,
		)
		*loggedBulkWarningPtr = true
		selfLogged = true
	}
	return selfLogged
}

func (handler *Handler) maybeNotifyAdmin(entries []*entry, loggedAdminNotificationErrorPtr *bool) bool {
	if *loggedAdminNotificationErrorPtr {
		return false
	}
	selfLogged := false

	shouldNotifyAdmin := false
	useFallback := handler.App.Env.ADMIN_USERNAME == ""
	for _, entry := range entries {
		if entry.level >= int(slog.LevelError) && !entry.disableAdminNotification {
			shouldNotifyAdmin = true
		}
		if entry.useAdminNotificationFallback {
			useFallback = true
		}
	}
	if shouldNotifyAdmin {
		if useFallback {
			baseCtx := context.Background()
			handler.topHandler.mu.RLock()
			if handler.topHandler.shutdownCtx != nil {
				baseCtx = handler.topHandler.shutdownCtx
			}
			handler.topHandler.mu.RUnlock()
			ctx, cancel := context.WithTimeout(baseCtx, 2*time.Second)
			defer cancel()
			shouldCrash, stdErr := dbcommon.WithReadWriteTx(
				ctx, handler.App.Database,
				func(tx *ent.Tx, ctx context.Context) (bool, error) {
					lastCrashSignal := time.Time{}
					wrappedErr := handler.App.KeyValue.Get("LAST_CRASH_SIGNAL", &lastCrashSignal, ctx)
					if wrappedErr != nil {
						return false, wrappedErr
					}
					now := handler.App.Clock.Now()
					if handler.App.Env.MIN_CRASH_SIGNAL_GAP <= 0 ||
						now.Before(lastCrashSignal.Add(handler.App.Env.MIN_CRASH_SIGNAL_GAP)) {
						return false, nil
					}

					wrappedErr = handler.App.KeyValue.Set("LAST_CRASH_SIGNAL", now, ctx)
					if wrappedErr != nil {
						return false, wrappedErr
					}
					return true, nil
				},
			)
			if stdErr != nil {
				pc, _, _, _ := runtime.Caller(0)
				record := slog.NewRecord(
					handler.App.Clock.Now(),
					slog.LevelError,
					"failed to check LAST_CRASH_SIGNAL in key/value storage. in order to be cautious, the server won't crash",
					pc,
				)
				record.AddAttrs(slog.Any("error", stdErr))

				_ = handler.Handle(
					context.WithValue(context.Background(), common.DisableAdminNotificationKey{}, true),
					record,
				)
				return true
			}

			if shouldCrash {
				handler.App.Shutdown("crashing to notify admin because messengers failed")
			}
			// Set here rather than at the fallback error logs to ensure the logger loops back around to here
			*loggedAdminNotificationErrorPtr = true
			return selfLogged
		}

		session, wrappedErr := handler.App.RateLimiter.RequestSession(
			"admin-error-message", 1, "",
		)
		if wrappedErr != nil {
			if errors.Is(wrappedErr, ratelimiting.ErrGlobalRateLimitExceeded) {
				return selfLogged
			}
			pc, _, _, _ := runtime.Caller(0)
			record := slog.NewRecord(
				handler.App.Clock.Now(),
				slog.LevelError,
				"failed to check admin-error-message rate limit",
				pc,
			)
			record.AddAttrs(slog.Any("error", wrappedErr))

			_ = handler.Handle(
				context.WithValue(context.Background(), common.AdminNotificationFallbackKey{}, true),
				record,
			)
			return true
		}

		// TODO: reserve a bit of time for this in case the database writing times out during a shutdown
		baseCtx := context.Background()
		handler.topHandler.mu.RLock()
		if handler.topHandler.shutdownCtx != nil {
			baseCtx = handler.topHandler.shutdownCtx
		}
		handler.topHandler.mu.RUnlock()
		ctx, cancel := context.WithTimeout(baseCtx, 2*time.Second)
		defer cancel()
		var queuedCount int
		var errs map[string]common.WrappedError
		stdErr := dbcommon.WithWriteTx(
			ctx, handler.App.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				userOb, stdErr := tx.User.Query().Where(user.Username(handler.App.Env.ADMIN_USERNAME)).Only(ctx)
				if stdErr != nil {
					return stdErr
				}
				var wrappedErr common.WrappedError
				queuedCount, errs, wrappedErr = handler.App.Messengers.SendUsingAll(
					&common.Message{
						Type: common.MessageAdminError,
						User: userOb,
					},
					ctx,
				)
				return wrappedErr
			},
		)
		cancel()
		if stdErr != nil {
			session.Cancel()
			pc, _, _, _ := runtime.Caller(0)
			record := slog.NewRecord(
				handler.App.Clock.Now(),
				slog.LevelError,
				"failed to message admin about an error",
				pc,
			)
			record.AddAttrs(slog.Any("error", stdErr))

			_ = handler.Handle(
				context.WithValue(context.Background(), common.AdminNotificationFallbackKey{}, true),
				record,
			)
			return true
		}

		if len(errs) > 0 { // SendUsingAll will have logged
			*loggedAdminNotificationErrorPtr = true
			selfLogged = true
		}
		if queuedCount == 0 {
			session.Cancel()
			message := "admin user has no contacts so couldn't notify them about an error"
			for _, wrappedErr := range errs {
				// TODO: this error should be moved to common (or common/errors?) to avoid circular imports in the future
				if !errors.Is(wrappedErr, messengers.ErrMessengerDisabledForUser) {
					message = "unable to prepare messages to notify admin about an error, see the errors before"
				}
			}

			pc, _, _, _ := runtime.Caller(0)
			record := slog.NewRecord(
				handler.App.Clock.Now(),
				slog.LevelError,
				message,
				pc,
			)

			_ = handler.Handle(
				context.WithValue(context.Background(), common.AdminNotificationFallbackKey{}, true),
				record,
			)
			selfLogged = true
		}
	}
	return selfLogged
}

func (handler *Handler) Shutdown() {
	go handler.Listen() // TODO: is this a race condition?
	handler.topHandler.shutdownOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		handler.topHandler.mu.Lock()
		handler.topHandler.shutdownCtx = ctx
		handler.topHandler.cancelShutdownCtx = cancel
		handler.topHandler.mu.Unlock()

		select {
		case handler.topHandler.requestShutdownChan <- struct{}{}:
			<-ctx.Done()
		case <-ctx.Done():
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			pc, _, _, _ := runtime.Caller(0)
			record := slog.NewRecord(
				handler.App.Clock.Now(),
				slog.LevelError,
				"logger shutdown timed out",
				pc,
			)
			_ = handler.tintHandler.Handle(ctx, record)
		}
	})
}

func (handler Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= handler.Level
}

func (handler Handler) Handle(ctx context.Context, record slog.Record) error {
	disableErrLogging, _ := ctx.Value(disableErrorLoggingKey{}).(bool)
	useAdminNotificationFallback, _ := ctx.Value(common.AdminNotificationFallbackKey{}).(bool)
	disableAdminNotification, _ := ctx.Value(common.DisableAdminNotificationKey{}).(bool)
	entry := &entry{
		level:                        int(record.Level),
		message:                      record.Message,
		disableErrorLogging:          disableErrLogging,
		useAdminNotificationFallback: useAdminNotificationFallback,
		disableAdminNotification:     disableAdminNotification,
	}
	if !record.Time.IsZero() {
		entry.time = record.Time
		entry.timeKnown = true
	}
	source := record.Source()
	if source != nil {
		entry.sourceFile = source.File
		entry.sourceFunction = source.Function
		entry.sourceLine = source.Line
	}

	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	handler.mu.RLock()
	//nolint:contextcheck // loggers should ignore deadlines and cancellations from the context
	resolvedAttrs := handler.resolveNestedAttrs(attrs, !disableErrLogging, &entry.publicMessage, &entry.userID)
	handler.mu.RUnlock()
	entry.attributes = resolvedAttrs

	stdErr := handler.tintHandler.Handle(ctx, record)
	handler.topHandler.entryChan <- entry
	if stdErr != nil && !disableErrLogging {
		pc, _, _, _ := runtime.Caller(0)
		record := slog.NewRecord(
			handler.App.Clock.Now(),
			slog.LevelWarn,
			"logger Handler.textHandler.Handle returned an error",
			pc,
		)
		record.AddAttrs(slog.Any("error", stdErr))
		//nolint:contextcheck // logging is a different context to the code that created the original log
		_ = handler.Handle(
			context.WithValue(context.Background(), disableErrorLoggingKey{}, true),
			record,
		)
	}

	if record.Level >= slog.LevelError && handler.App.Env.PANIC_ON_ERROR {
		panic("an error was logged")
	}
	return nil
}

type specialProperties struct {
	publicMessage string
	userID        uuid.UUID
}

func (handler Handler) resolveNestedAttrs(
	attrs []slog.Attr, logErrors bool,
	publicMessagePtr *string, userIDPtr *uuid.UUID,
) map[string]any {
	resolved := maps.Clone(handler.baseAttrs)
	nestedResolved := resolved
	for _, key := range handler.baseGroups {
		newMap, ok := nestedResolved[key].(map[string]any)
		if ok {
			newMap = maps.Clone(newMap)
		} else {
			newMap = map[string]any{}
		}
		nestedResolved[key] = newMap
		nestedResolved = newMap
	}

	isTopLevel := len(handler.baseGroups) == 0
	for _, attr := range attrs {
		handler.appendAttr(attr, nestedResolved, isTopLevel, logErrors, publicMessagePtr, userIDPtr)
	}
	return resolved
}

// Note: handler.baseGroups is handled by appendNestedAttrs instead

func (handler Handler) appendAttr(
	attr slog.Attr, outputAttrs map[string]any,
	isTopLevel bool, logErrors bool,
	publicMessagePtr *string, userIDPtr *uuid.UUID,
) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}

	kind := attr.Value.Kind()
	if kind == slog.KindGroup {
		groupAttrs := attr.Value.Group()
		if len(groupAttrs) == 0 {
			return
		}
		// If the key is non-empty, write it out and indent the rest of the attrs.
		// Otherwise, inline the attrs.
		if attr.Key == "" { // Inline
			for _, attr := range groupAttrs {
				handler.appendAttr(attr, outputAttrs, true, logErrors, publicMessagePtr, userIDPtr)
			}
		} else {
			groupAttr := map[string]any{}
			for _, attr := range groupAttrs {
				handler.appendAttr(attr, groupAttr, false, logErrors, common.Pointer(""), common.Pointer(uuid.Nil))
			}
			outputAttrs[attr.Key] = groupAttr
		}
		return
	}
	if isTopLevel {
		if attr.Key == PublicMessageKey {
			*publicMessagePtr = attr.Value.String()
			return
		}
		if attr.Key == UserIDKey {
			uuidValue, ok := attr.Value.Any().(uuid.UUID)
			if ok {
				*userIDPtr = uuidValue
			} else if logErrors {
				pc, _, _, _ := runtime.Caller(0)
				record := slog.NewRecord(
					handler.App.Clock.Now(),
					slog.LevelWarn,
					"userID property in log statement was not a UUID so has been ignored",
					pc,
				)
				record.AddAttrs(slog.String("type", reflect.TypeOf(attr.Value.Any()).String()))

				_ = handler.Handle(
					context.WithValue(context.Background(), disableErrorLoggingKey{}, true),
					record,
				)
			}
			// Also store the value in the attributes so it's preserved if the user is deleted
			outputAttrs[attr.Key] = attr.Value.Any()
			return
		}
	}

	outputAttrs[attr.Key] = attr.Value.Any()
}

func (handler Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return handler
	}
	oldMu := handler.mu
	oldMu.RLock()
	defer oldMu.RUnlock()

	handler.tintHandler = handler.tintHandler.WithAttrs(attrs)
	resolvedAttrs := handler.resolveNestedAttrs(
		attrs, true,
		&handler.baseSpecialProps.publicMessage, &handler.baseSpecialProps.userID,
	)
	handler.baseAttrs = resolvedAttrs
	handler.mu = &sync.RWMutex{}
	// We don't need to clone any other properties since they get cloned before modification

	return handler
}

func (handler Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return handler
	}
	oldMu := handler.mu
	oldMu.RLock()
	defer oldMu.RUnlock()

	handler.tintHandler = handler.tintHandler.WithGroup(name)
	handler.baseGroups = slices.Concat(handler.baseGroups, []string{name})
	handler.mu = &sync.RWMutex{}
	// We don't need to clone any other properties since they get cloned before modification

	return handler
}
