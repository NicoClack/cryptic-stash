package definitions

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/messengers"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/websocket"
)

var ErrWrapperDiscord = common.NewDynamicErrorWrapper(func(err error) common.WrappedError {
	wrappedErr := common.ErrWrapperAPI.Wrap(err)
	if wrappedErr == nil {
		return nil
	}

	isUrlErr := common.IsErrorType[*url.Error](err)
	isWebsocketErr := common.IsErrorType[*websocket.CloseError](err)
	if isUrlErr || isWebsocketErr {
		wrappedErr.ConfigureRetriesMut(10, time.Second*5, 1.5)
		if isUrlErr {
			wrappedErr.AddDebugValuesMut(common.DebugValue{
				Name: "retried url.Error",
			})
		} else if isWebsocketErr {
			wrappedErr.AddDebugValuesMut(common.DebugValue{
				Name: "retried websocket.CloseError",
			})
		}
		return wrappedErr
	}
	// TODO: retry other HTTP errors?
	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) && restErr.Message != nil {
		if restErr.Message.Code == discordgo.ErrCodeOpeningDirectMessagesTooFast {
			wrappedErr.ConfigureRetriesMut(10, time.Second*5, 2)
			wrappedErr.AddDebugValuesMut(common.DebugValue{
				Name: "retried discordgo.ErrCodeOpeningDirectMessagesTooFast",
			})
			return wrappedErr
		}
	}

	var rateLimitErr *discordgo.RateLimitError
	if errors.As(err, &rateLimitErr) {
		wrappedErr.ConfigureRetriesMut(3, max(rateLimitErr.RetryAfter, 5*time.Second), 1)
		wrappedErr.AddDebugValuesMut(common.DebugValue{
			Name:    "retried discordgo.RateLimitError",
			Message: fmt.Sprintf("RetryAfter: %v", rateLimitErr.RetryAfter),
		})
		return wrappedErr
	}

	return wrappedErr
})

type Discord1Options struct {
	UserID string `json:"userId"`
}
type Discord1Body struct {
	Options          *Discord1Options `json:"options"`
	FormattedMessage string           `json:"formattedMessage"`
}

func Discord1(app *common.App) *messengers.Definition {
	getSession := func() (*discordgo.Session, common.WrappedError) {
		session, stdErr := discordgo.New("Bot " + app.Env.DISCORD_TOKEN)
		if stdErr != nil {
			return nil, ErrWrapperDiscord.Wrap(stdErr)
		}

		session.ShouldRetryOnRateLimit = false
		return session, nil
	}
	session, wrappedErr := getSession()
	if wrappedErr != nil {
		log.Fatalf("error creating startup test Discord session:\n%v", wrappedErr)
	}
	stdErr := session.Close()
	if stdErr != nil {
		log.Fatalf("error closing startup test Discord session:\n%v", stdErr)
	}

	return &messengers.Definition{
		ID:                "discord",
		Version:           1,
		Name:              "Discord",
		OptionsType:       &Discord1Options{},
		OptionsSchemaPath: "discord/v1/options.schema.json",
		Prepare: func(prepareCtx *messengers.PrepareContext) (any, error) {
			options := &Discord1Options{}
			wrappedErr := prepareCtx.DecodeOptions(options)
			if wrappedErr != nil {
				return nil, wrappedErr
			}

			formattedMessage, wrappedErr := messengers.FormatDefaultMessage(prepareCtx.Message)
			if wrappedErr != nil {
				return nil, wrappedErr
			}

			return &Discord1Body{
				Options:          options,
				FormattedMessage: formattedMessage.Body,
			}, nil
		},
		BodyType: &Discord1Body{},
		Handler: func(messengerCtx *messengers.Context) error {
			body := &Discord1Body{}
			wrappedErr := messengerCtx.Decode(body)
			if wrappedErr != nil {
				return wrappedErr
			}

			session, wrappedErr := getSession()
			if wrappedErr != nil {
				return wrappedErr
			}
			stdErr := session.Open()
			if stdErr != nil {
				return ErrWrapperDiscord.Wrap(stdErr)
			}

			// TODO: why does calling close and returning here cause a log?
			// Looks like it's trying to send a heartbeat after closing
			// Maybe need to explicitly stop listening for VC events since they aren't being used?
			defer func() {
				stdErr := session.Close()
				if stdErr != nil {
					messengerCtx.Logger.Warn("error closing Discord session", "error", stdErr)
				}
			}()

			// TODO: it's probably worth caching this to reduce how often we're rate limited
			channel, stdErr := session.UserChannelCreate(body.Options.UserID)
			if stdErr != nil {
				return ErrWrapperDiscord.Wrap(stdErr)
			}
			_, stdErr = session.ChannelMessageSend(channel.ID, body.FormattedMessage)
			if stdErr != nil {
				return ErrWrapperDiscord.Wrap(stdErr)
			}

			messengerCtx.ConfirmSent()
			return nil
		},
	}
}
