package common

/*
The core principal is to abstract just enough that:
* The service can be mocked to some extent (although I don't think this is really necessary for the database)
* The service can be used in simplified ways for testing.
e.g a test can use a different job registry with a real implementation
*/

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

type Env struct {
	IS_DEV                        bool
	PORT                          int
	MOUNT_PATH                    string
	PROXY_ORIGINAL_IP_HEADER_NAME string
	ALLOWED_ORIGINS               []string
	// Things like deleting expired login sessions
	CLEAN_UP_INTERVAL time.Duration
	FULL_GC_INTERVAL  time.Duration

	JOB_POLL_INTERVAL    time.Duration
	MAX_TOTAL_JOB_WEIGHT int

	ADMIN_PASSWORD_HASH_SETTINGS *PasswordHashSettings
	ENABLE_ENV_SETUP             bool
	ADMIN_CODE_ROTATION_INTERVAL time.Duration
	ADMIN_PASSWORD_HASH          []byte
	ADMIN_PASSWORD_SALT          []byte
	ADMIN_TOTP_SECRET            string

	UNLOCK_TIME         time.Duration
	AUTH_CODE_VALID_FOR time.Duration
	// Once used, how much longer the auth code remains valid for
	USED_AUTH_CODE_VALID_FOR         time.Duration
	ACTIVE_SESSION_REMINDER_INTERVAL time.Duration
	MIN_SUCCESSFUL_MESSAGE_COUNT     int
	PASSWORD_HASH_SETTINGS           *PasswordHashSettings

	LOG_STORE_INTERVAL time.Duration
	// How long the server should wait for messengers to succeed before crashing the server to send the message
	// Note: this time will be exceeded as it's a simple check when the job succeeds and doesn't take into account
	// when the next retry is.
	// Note: currently all of the successfully prepared messages must succeed for a crash to be avoided
	ADMIN_MESSAGE_TIMEOUT time.Duration
	// If it's been less than this amount of time since the last admin message,
	// other errors won't send a message to avoid spamming the admin
	MIN_ADMIN_MESSAGE_GAP time.Duration
	MIN_CRASH_SIGNAL_GAP  time.Duration
	// Used for testing, not recommended when running the server
	PANIC_ON_ERROR bool

	ENABLE_DEVELOP_MESSENGER bool
	DISCORD_TOKEN            string
	SENDGRID_TOKEN           string // TODO: implement
}
type PasswordHashSettings struct {
	Time   uint32
	Memory uint32
	// Note: this affects the hash produced
	Threads uint8
}

type App struct {
	Env              *Env
	Clock            clockwork.Clock
	Logger           LoggerService
	RateLimiter      LimiterService
	ShutdownService  ShutdownService
	Database         DatabaseService
	KeyValue         KeyValueService
	TwoFactorActions TwoFactorActionService
	Messengers       MessengerService
	Server           ServerService
	Core             CoreService
	Setup            SetupService
	Jobs             JobService
	Scheduler        SchedulerService
}

// If reason is "", the server will exit with a 0 exit code
func (app *App) Shutdown(reason string) {
	go app.ShutdownService.Shutdown(reason)
}

type HasDefaultLogger interface {
	DefaultLogger() Logger
}

type MessengerService interface {
	Send(
		versionedType string, message *Message,
		ctx context.Context,
	) WrappedError
	ScheduleSend(
		versionedType string, message *Message,
		sendTime time.Time,
		ctx context.Context,
	) WrappedError

	// The error map is more like warnings about why specific messengers failed to prepare,
	// they are logged already so you might just want to ignore them
	//
	// But check the second WrappedError value first because you should fail the transaction if it's not nil
	//
	// Note: the number of successfully queued messages (the int return value) might not be 0 if some messages
	// were queued before a non-messenger specific error occurred
	SendUsingAll(message *Message, ctx context.Context) (int, map[string]WrappedError, WrappedError)
	ScheduleSendUsingAll(
		message *Message,
		sendTime time.Time,
		ctx context.Context,
	) (int, map[string]WrappedError, WrappedError)
	SendBulk(messages []*Message, ctx context.Context) WrappedError

	GetConfiguredMessengerTypes(user *ent.User) []string
	GetPublicDefinition(versionedType string) (*MessengerDefinition, bool)
	AllPublicDefinitions() []*MessengerDefinition
	EnableMessenger(
		userOb *ent.User,
		versionedType string,
		options json.RawMessage,
		ctx context.Context,
	) WrappedError
	DisableMessenger(
		userOb *ent.User,
		versionedType string,
		ctx context.Context,
	) WrappedError
}
type MessageType string

const (
	MessageTest                  MessageType = "test"
	MessageAdminError            MessageType = "adminError"
	MessageRegular               MessageType = "regular"
	MessageLogin                 MessageType = "login"
	MessageActiveSessionReminder MessageType = "activeSessionReminder"
	MessageDownload              MessageType = "download"
	MessageUserUpdate            MessageType = "userUpdate"
	MessageLock                  MessageType = "lock"
	MessageUnlock                MessageType = "unlock"
	MessageSelfLock              MessageType = "selfLock"
	MessageSelfUnlock            MessageType = "selfUnlock" // When the self-lock expires
	Message2FA                   MessageType = "2FA"
)

type Message struct {
	Type       MessageType
	User       *ent.User
	Code       string
	Time       time.Time
	SessionIDs []uuid.UUID
}

// The public version of *messengers.Definition
type MessengerDefinition struct {
	ID             string
	Version        int
	Name           string
	IsSupplemental bool
	OptionsSchema  json.RawMessage
}

type Logger interface {
	Debug(msg string, args ...any)
	DebugContext(ctx context.Context, msg string, args ...any)
	Enabled(ctx context.Context, level slog.Level) bool
	Error(msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
	Info(msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
	Warn(msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	With(args ...any) *slog.Logger
	WithGroup(name string) *slog.Logger
}

// When in a context passed to a logger.Error call, the server will deliberately crash to
// notify the admin as opposed to sending a message
type AdminNotificationFallbackKey struct{}

// When in a context passed to a logger.Error call, the server won't attempt to notify the admin,
// neither by crashing or sending a message
type DisableAdminNotificationKey struct{}

// Used to store a logger override in a context
type LoggerKey struct{}
type LoggerService interface {
	Logger
	Start()    // Should fatalf rather than returning an error
	Shutdown() // Should log warning rather than return an error
}

type ShutdownService interface {
	// Note: this blocks until shutdown is complete, crashes should usually call this in a separate Goroutine
	//
	// If reason is "", the server will exit with a 0 exit code
	Shutdown(reason string)
}

type DatabaseService interface {
	HasDefaultLogger
	Start()    // Should fatalf rather than returning an error
	Shutdown() // Should log warning rather than return an error
	Client() *ent.Client
	ReadTx(ctx context.Context) (*ent.Tx, error)
	WriteTx(ctx context.Context) (*ent.Tx, error)
}
type KeyValueService interface {
	Init()
	Get(name string, ptr any, ctx context.Context) WrappedError
	Set(name string, value any, ctx context.Context) WrappedError
}

type ServerService interface {
	http.Handler // Mainly used for testing
	Start()      // Should fatalf rather than returning an error
	Shutdown()   // Should log warning rather than return an error
}
type CoreService interface {
	CheckAdminCode(givenCode string) bool
	CheckAdminCredentials(password string, totpCode string) bool
	GetAdminCode(password string, totpCode string) (string, bool)
	RandomAuthCode() []byte

	SendActiveSessionReminders(ctx context.Context) WrappedError
	DeleteExpiredSessions(ctx context.Context) WrappedError
	InvalidateUserSessions(userID uuid.UUID, ctx context.Context) WrappedError
	IsUserSufficientlyNotified(sessionOb *ent.Session) bool
	IsUserLocked(userOb *ent.User) bool

	Encrypt(data []byte, encryptionKey []byte) ([]byte, []byte, WrappedError)
	Decrypt(encrypted []byte, encryptionKey []byte, nonce []byte) ([]byte, WrappedError)
	GenerateSalt() []byte
	HashPassword(password string, salt []byte, settings *PasswordHashSettings) []byte
}

type JobService interface {
	Start()    // Should fatalf rather than returning an error
	Shutdown() // Should log warning rather than return an error
	Enqueue(
		versionedType string,
		body any,
		ctx context.Context,
	) (*ent.Job, WrappedError)
	EnqueueEncoded(
		versionedType string,
		encodedBody json.RawMessage,
		ctx context.Context,
	) (*ent.Job, WrappedError)
	EnqueueWithModifier(
		versionedType string,
		body any,
		modifications func(jobCreate *ent.JobCreate),
		ctx context.Context,
	) (*ent.Job, WrappedError)
	EnqueueEncodedWithModifier(
		versionedType string,
		encodedBody json.RawMessage,
		modifications func(jobCreate *ent.JobCreate),
		ctx context.Context,
	) (*ent.Job, WrappedError)
	WaitForJobs()
	Encode(versionedType string, body any) (json.RawMessage, WrappedError)
}
type TwoFactorActionService interface {
	Create(
		versionedType string,
		expiresAt time.Time,
		body any,
		ctx context.Context,
	) (*ent.TwoFactorAction, string, WrappedError)
	Confirm(actionID uuid.UUID, code string, ctx context.Context) (*ent.Job, WrappedError)
	DeleteExpiredActions(ctx context.Context) WrappedError
}

type SchedulerService interface {
	Start()    // Should fatalf rather than returning an error
	Shutdown() // Should log warning rather than return an error
}

type LimiterService interface {
	RequestSession(eventName string, amount int, user string) (LimiterSession, WrappedError)
	DeleteInactiveUsers()
}
type LimiterSession interface {
	AdjustTo(amount int) WrappedError
	Cancel()
}

type SetupService interface {
	GetStatus(ctx context.Context) (*SetupStatus, WrappedError)
	GenerateAdminSetupConstants(password string) (*AdminAuthEnvVars, string, WrappedError)
	// Only used for setup, otherwise use app.Core.CheckAdminCredentials instead
	CheckTotpCode(totpCode string, totpSecret string) bool
}

type SetupStatus struct {
	IsComplete                   bool
	IsEnvComplete                bool
	AreAdminMessengersConfigured bool
}
type AdminAuthEnvVars struct {
	//nolint:tagliatelle
	AdminPasswordHash string `json:"ADMIN_PASSWORD_HASH"`
	//nolint:tagliatelle
	AdminPasswordSalt string `json:"ADMIN_PASSWORD_SALT"`
	//nolint:tagliatelle
	AdminTotpSecret string `json:"ADMIN_TOTP_SECRET"`
}
