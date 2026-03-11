package messengers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path"
	"reflect"
	"slices"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/usermessenger"
	"github.com/NicoClack/cryptic-stash/backend/jobs"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/google/uuid"
	"github.com/xeipuuv/gojsonschema"
)

const JobNamePrefix = "messengers"

type Registry struct {
	App        *common.App
	messengers map[string]*Definition
	// RegisterJobs can be called with a prefix, so this is JobNamePrefix + that
	jobNamePrefix string
	embeddedDir   common.EmbeddedDirectory
}

type Definition struct {
	ID      string
	Version int
	Name    string
	// Supplemental messengers often can't tell if a message was successfully sent. So they increase the
	// chance the user is notified but are ignored when assessing if the user was sufficiently notified.
	// This also means that the user needs to have at least one non-supplemental messenger configured,
	// otherwise they'll never be considered sufficiently notified.
	IsSupplemental    bool
	OptionsType       any
	OptionsSchemaPath string
	// Returns the data the Handler needs, typically a struct containing the formatted message and
	// some sort of contact (e.g a username)
	Prepare PrepareFunc
	// The return type of Prepare
	BodyType          any
	Handler           HandlerFunc
	jobDefinition     *jobs.Definition
	reflectedBodyType reflect.Type
	optionsSchema     *common.PublicJSONSchema
}

type PrepareFunc = func(prepareCtx *PrepareContext) (any, error)
type HandlerFunc func(messengerCtx *Context) error

type JobContext = jobs.Context

type bodyWrapperType struct {
	MessageType            common.MessageType
	VersionedMessengerType string
	DownloadSessionIDs     []uuid.UUID
	Inner                  string
}
type Context struct {
	*JobContext
	confirmedSent bool
}

func (ctx *Context) ConfirmSent() {
	ctx.confirmedSent = true
}

func NewRegistry(app *common.App) *Registry {
	return &Registry{
		messengers: make(map[string]*Definition),
		App:        app,
	}
}

func (registry *Registry) SetEmbeddedDir(embeddedDir common.EmbeddedDirectory) {
	if registry.embeddedDir.Path != "" {
		panic("messenger registry's embedded directory has already been set")
	}
	registry.embeddedDir = embeddedDir
}

func (registry *Registry) Register(definition *Definition) {
	versionedType := common.GetVersionedType(definition.ID, definition.Version)
	if _, exists := registry.messengers[versionedType]; exists {
		log.Fatalf("messenger definition with ID \"%s\" already exists", versionedType)
	}
	definition.reflectedBodyType = reflect.TypeOf(definition.BodyType)
	validateDefinition(definition)
	if definition.OptionsSchemaPath != "" {
		definition.optionsSchema = requirePublicSchema(definition.OptionsSchemaPath, registry.embeddedDir)
	}

	definition.jobDefinition = &jobs.Definition{
		ID:      definition.ID,
		Version: definition.Version,
		Handler: func(jobCtx *jobs.Context) error {
			body := &bodyWrapperType{}
			wrappedErr := jobCtx.Decode(body)
			if wrappedErr != nil {
				return ErrWrapperHandlerWrapper.Wrap(wrappedErr)
			}

			newJobCtx := *jobCtx
			newJobCtx.Body = json.RawMessage(body.Inner)
			messengerCtx := &Context{
				JobContext:    &newJobCtx,
				confirmedSent: false,
			}
			stdErr := definition.Handler(messengerCtx)
			if stdErr != nil {
				if body.MessageType == common.MessageAdminError {
					wrappedErr := common.WrapErrorWithCategories(stdErr)
					if wrappedErr.MaxRetries() > 0 || wrappedErr.MaxRetries() == -1 {
						if registry.App.Clock.Since(jobCtx.OriginallyDueAt) >= registry.App.Env.ADMIN_MESSAGE_TIMEOUT {
							jobCtx.Logger.ErrorContext(
								context.WithValue(context.Background(), common.AdminNotificationFallbackKey{}, true),
								"failed to notify admin about an error before ADMIN_MESSAGE_TIMEOUT, "+
									"will now possibly crash to notify them earlier",
								"jobID",
								jobCtx.ID,
							)
						}
					} else {
						jobCtx.Logger.ErrorContext(
							context.WithValue(context.Background(), common.AdminNotificationFallbackKey{}, true),
							"failed to notify admin about an error! will now possibly crash to notify them earlier",
							"jobID",
							jobCtx.ID,
						)
					}
				}
				return stdErr
			}

			if body.MessageType == common.MessageLogin ||
				body.MessageType == common.MessageActiveDownloadSessionReminder {
				// This is in a separate transaction, so we could successfully commit the messenger's transaction
				// but roll back this one. But that's ok since it's best to undercount the successful login alerts sent
				stdErr := dbcommon.WithWriteTx(
					jobCtx.Context, registry.App.Database,
					func(tx *ent.Tx, ctx context.Context) error {
						return tx.LoginAlert.MapCreateBulk(
							body.DownloadSessionIDs,
							func(alertCreate *ent.LoginAlertCreate, i int) {
								alertCreate.
									SetSentAt(registry.App.Clock.Now()).
									SetConfirmed(messengerCtx.confirmedSent).
									SetDownloadSessionID(body.DownloadSessionIDs[i])
							},
						).Exec(ctx)
					},
				)
				if stdErr != nil {
					// TODO: handle missing download session IDs, stop this being atomic
					jobCtx.Logger.Error(
						"failed to create LoginAlert objects for successfully sent message, if not enough objects are created, "+
							"the user won't be able to download their data once their download session becomes valid",
						"error",
						stdErr,
						"downloadSessionIDs",
						body.DownloadSessionIDs,
					)
				}
			}
			return nil
		},
		BodyType: &bodyWrapperType{},
		Weight:   1,
	}
	registry.messengers[versionedType] = definition
}
func validateDefinition(definition *Definition) {
	// TODO
	versionedType := common.GetVersionedType(definition.ID, definition.Version)
	jobs.AssertTypeIsValidBodyType(definition.reflectedBodyType, versionedType)
}
func requirePublicSchema(schemaPath string, embeddedDir common.EmbeddedDirectory) *common.PublicJSONSchema {
	schemaFileBytes, stdErr := embeddedDir.FS.ReadFile(
		path.Join(embeddedDir.Path, schemaPath),
	)
	if stdErr != nil {
		log.Fatalf("couldn't read embedded messenger schema \"%v\". error:\n%v", schemaPath, stdErr)
	}

	loader := gojsonschema.NewStringLoader(string(schemaFileBytes))
	schema, stdErr := gojsonschema.NewSchema(loader)
	if stdErr != nil {
		log.Fatalf("couldn't load messenger schema \"%v\". error:\n%v", schemaPath, stdErr)
	}
	publicSchema, wrappedErr := common.NewPublicJSONSchema(schema, schemaFileBytes)
	if wrappedErr != nil {
		log.Fatalf("couldn't create public messenger schema \"%v\". error:\n%v", schemaPath, wrappedErr)
	}
	return publicSchema
}

func (registry *Registry) RegisterJobs(group *jobs.RegistryGroup) {
	registry.jobNamePrefix = common.JoinPaths(group.Path, JobNamePrefix)
	prefixedGroup := group.Group(JobNamePrefix)
	for _, messenger := range registry.messengers {
		prefixedGroup.Register(messenger.jobDefinition)
	}
}

type PrepareContext struct {
	Message *common.Message
	Options json.RawMessage
}

func (ctx *PrepareContext) DecodeOptions(pointer any) common.WrappedError {
	stdErr := json.Unmarshal(ctx.Options, pointer)
	if stdErr != nil {
		return ErrWrapperDecodeOptions.Wrap(stdErr)
	}
	return nil
}

func (registry *Registry) Send(
	versionedType string, message *common.Message,
	sendTime time.Time,
	ctx context.Context,
) common.WrappedError {
	if message.User.Edges.Messengers == nil {
		panic("Send: user messengers edge must be loaded")
	}
	messengerDef, ok := registry.messengers[versionedType]
	if !ok {
		return ErrWrapperSend.Wrap(ErrUnknownMessengerType)
	}
	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return ErrWrapperSend.Wrap(common.ErrNoTxInContext)
	}

	prepareCtx, wrappedErr := newPrepareContext(versionedType, message)
	if wrappedErr != nil {
		return ErrWrapperSend.Wrap(wrappedErr)
	}
	preparedData, stdErr := messengerDef.Prepare(prepareCtx)
	if stdErr != nil {
		return ErrWrapperSend.Wrap(ErrWrapperPrepare.Wrap(stdErr))
	}
	bodyType := reflect.TypeOf(preparedData)
	if bodyType != messengerDef.reflectedBodyType {
		return ErrWrapperSend.Wrap(ErrWrapperEnqueueJob.Wrap(
			jobs.ErrWrapperEncode.Wrap(jobs.ErrWrapperInvalidBody.Wrap(
				fmt.Errorf("body type %s isn't the expected type %s",
					preparedData, messengerDef.reflectedBodyType),
			)),
		))
	}
	encoded, stdErr := json.Marshal(preparedData)
	if stdErr != nil {
		return ErrWrapperSend.Wrap(ErrWrapperEnqueueJob.Wrap(
			jobs.ErrWrapperEncode.Wrap(jobs.ErrWrapperInvalidBody.Wrap(stdErr)),
		))
	}

	_, wrappedErr = registry.App.Jobs.EnqueueWithModifier(
		common.JoinPaths(registry.jobNamePrefix, versionedType),
		&bodyWrapperType{
			MessageType:            message.Type,
			VersionedMessengerType: versionedType,
			DownloadSessionIDs:     message.DownloadSessionIDs,
			Inner:                  string(encoded),
		},
		func(jobCreate *ent.JobCreate) {
			jobCreate.SetDueAt(sendTime)
		},
		ctx,
	)
	if wrappedErr != nil {
		return ErrWrapperSend.Wrap(ErrWrapperEnqueueJob.Wrap(wrappedErr))
	}

	logger.Info(
		"sending message to user",
		"userID", message.User.ID,
		"messageType", message.Type,
	)
	return nil
}
func newPrepareContext(
	versionedType string,
	message *common.Message,
) (*PrepareContext, common.WrappedError) {
	index := slices.IndexFunc(message.User.Edges.Messengers, func(userMessengerOb *ent.UserMessenger) bool {
		return common.GetVersionedType(userMessengerOb.Type, userMessengerOb.Version) == versionedType
	})
	if index == -1 {
		return nil, ErrMessengerDisabledForUser.Clone()
	}
	userMessengerOb := message.User.Edges.Messengers[index]
	if !userMessengerOb.Enabled {
		return nil, ErrMessengerDisabledForUser.Clone()
	}
	return &PrepareContext{
		Message: message,
		Options: userMessengerOb.Options,
	}, nil
}

func (registry *Registry) SendUsingAll(
	message *common.Message,
	sendTime time.Time,
	ctx context.Context,
) (int, map[string]common.WrappedError, common.WrappedError) {
	errs := make(map[string]common.WrappedError)
	messagesQueued := 0
	for versionedType := range registry.messengers {
		wrappedErr := registry.Send(versionedType, message, sendTime, ctx)
		if wrappedErr == nil {
			messagesQueued++
		} else {
			errs[versionedType] = wrappedErr
			if errors.Is(wrappedErr, ErrMessengerDisabledForUser) {
				continue
			}
			// Just log an error and let the admin deal with this, there's not much the user can do
			registry.App.Logger.Error(
				"failed to enqueue message send",
				"messengerType", versionedType,
				"error", wrappedErr,
			)
		}
	}
	return messagesQueued, errs, nil
}

// Note: lastSendTime will be zero for the first call
func (registry *Registry) SendBulk(
	messages []*common.Message, sendTimeFunc func(lastSendTime time.Time, index int) time.Time, ctx context.Context,
) common.WrappedError {
	if len(messages) == 0 {
		return nil
	}
	sendTime := sendTimeFunc(time.Time{}, 0)
	for index, message := range messages {
		_, _, wrappedErr := registry.SendUsingAll(
			message, sendTime, ctx,
		)
		if wrappedErr != nil {
			return ErrWrapperSendBulk.Wrap(wrappedErr)
		}
		sendTime = sendTimeFunc(sendTime, index+1)
	}
	return nil
}

func (registry *Registry) GetConfiguredMessengerTypes(userOb *ent.User) []string {
	if userOb.Edges.Messengers == nil {
		panic("GetConfiguredMessengerTypes: user messengers edge must be loaded")
	}

	configuredTypes := []string{}
	for versionedType, messengerDef := range registry.messengers {
		prepareCtx, wrappedErr := newPrepareContext(
			versionedType,
			&common.Message{
				Type: common.MessageTest,
				User: userOb,
			},
		)
		if wrappedErr != nil {
			continue
		}
		_, stdErr := messengerDef.Prepare(prepareCtx)
		if stdErr != nil {
			continue
		}
		configuredTypes = append(configuredTypes, versionedType)
	}
	return configuredTypes
}
func (registry *Registry) GetPublicDefinition(versionedType string) (*common.MessengerDefinition, bool) {
	definition, ok := registry.messengers[versionedType]
	if !ok {
		return nil, false
	}
	return getPublicDefinition(definition), true
}
func getPublicDefinition(definition *Definition) *common.MessengerDefinition {
	publicOptionsSchema := json.RawMessage(nil)
	if definition.optionsSchema != nil {
		publicOptionsSchema = definition.optionsSchema.PublicSchema
	}

	//exhaustruct:enforce
	return &common.MessengerDefinition{
		ID:             definition.ID,
		Version:        definition.Version,
		Name:           definition.Name,
		IsSupplemental: definition.IsSupplemental,
		OptionsSchema:  publicOptionsSchema,
	}
}
func (registry *Registry) AllPublicDefinitions() []*common.MessengerDefinition {
	definitions := make([]*common.MessengerDefinition, 0, len(registry.messengers))
	for _, definition := range registry.messengers {
		definitions = append(definitions, getPublicDefinition(definition))
	}
	return definitions
}

func (registry *Registry) EnableMessenger(
	userOb *ent.User,
	versionedType string,
	options json.RawMessage,
	ctx context.Context,
) common.WrappedError {
	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return ErrWrapperEnableMessenger.Wrap(common.ErrNoTxInContext)
	}
	definition, ok := registry.messengers[versionedType]
	if !ok {
		return ErrWrapperEnableMessenger.Wrap(ErrUnknownMessengerType)
	}

	if definition.optionsSchema == nil {
		if options != nil {
			return ErrWrapperEnableMessenger.Wrap(ErrNoOptionsAcceptedForMessenger)
		}
	} else {
		result, stdErr := definition.optionsSchema.Validate(
			gojsonschema.NewBytesLoader(options),
		)
		// TODO: improve this logic
		if stdErr != nil {
			return ErrWrapperEnableMessenger.Wrap(stdErr)
		}
		if !result.Valid() {
			return ErrWrapperEnableMessenger.Wrap(fmt.Errorf("validation failed"))
		}
	}

	now := registry.App.Clock.Now()
	stdErr := tx.UserMessenger.
		Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetType(definition.ID).
		SetVersion(definition.Version).
		SetUserID(userOb.ID).
		SetOptions(options).
		SetEnabled(true).
		SetUpdatedAt(now).
		OnConflictColumns(usermessenger.FieldType, usermessenger.FieldVersion, usermessenger.FieldUserID).
		UpdateUpdatedAt().
		UpdateOptions().
		UpdateEnabled().
		Exec(ctx)
	if stdErr != nil {
		return ErrWrapperEnableMessenger.Wrap(stdErr)
	}

	return nil
}

func (registry *Registry) DisableMessenger(
	userOb *ent.User,
	versionedType string,
	ctx context.Context,
) common.WrappedError {
	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return ErrWrapperDisableMessenger.Wrap(common.ErrNoTxInContext)
	}
	definition, ok := registry.messengers[versionedType]
	if !ok {
		return ErrWrapperDisableMessenger.Wrap(ErrUnknownMessengerType)
	}

	stdErr := tx.UserMessenger.
		Update().
		Where(
			usermessenger.Type(definition.ID),
			usermessenger.Version(definition.Version),
			usermessenger.UserID(userOb.ID),
		).
		SetUpdatedAt(registry.App.Clock.Now()).
		SetEnabled(false).
		Exec(ctx)
	if stdErr != nil {
		return ErrWrapperDisableMessenger.Wrap(stdErr)
	}

	return nil
}
