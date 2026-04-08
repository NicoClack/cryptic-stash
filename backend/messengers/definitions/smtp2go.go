package definitions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	stdmail "net/mail"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/messengers"
)

const (
	smtp2goResponseBodyByteLimit = 64 * 1024
)

var ErrWrapperSMTP2GO = common.NewDynamicErrorWrapper(func(err error) common.WrappedError {
	wrappedErr := common.ErrWrapperAPI.Wrap(err)
	if wrappedErr == nil {
		return nil
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		wrappedErr.ConfigureRetriesMut(10, 5*time.Second, 1.5)
		wrappedErr.AddDebugValuesMut(common.DebugValue{
			Name: "retried net.Error",
		})
		return wrappedErr
	}

	// TODO: review
	var restErr *common.RESTError
	if errors.As(err, &restErr) {
		statusCode := restErr.Response.StatusCode
		// TODO: only retry specific statuses?
		if statusCode == http.StatusTooManyRequests {
			wrappedErr.ConfigureRetriesMut(10, 5*time.Second, 2)
			wrappedErr.AddDebugValuesMut(common.DebugValue{
				Name: "retried SMTP2GO 429 response",
			})
			return wrappedErr
		}
		if statusCode >= 500 && statusCode < 600 {
			wrappedErr.ConfigureRetriesMut(10, 5*time.Second, 1.5)
			wrappedErr.AddDebugValuesMut(common.DebugValue{
				Name: "retried SMTP2GO 5xx response",
			})
			return wrappedErr
		}
	}

	return wrappedErr
})

type SMTP2GO1Options struct {
	Email string `json:"email"`
}

type SMTP2GO1Body struct {
	ToAddress        stdmail.Address `json:"toAddress"`
	Subject          string          `json:"subject"`
	FormattedMessage string          `json:"formattedMessage"`
}

type smtp2goSendRequest struct {
	Sender   string   `json:"sender"`
	To       []string `json:"to"`
	Subject  string   `json:"subject"`
	TextBody string   `json:"text_body"`
}
type smtp2goSendResponse struct {
	RequestID string              `json:"request_id"`
	Data      smtp2goResponseData `json:"data"`
	Error     string              `json:"error"`
}
type smtp2goResponseData struct {
	Succeeded int              `json:"succeeded"`
	Failed    int              `json:"failed"`
	Failures  []smtp2goFailure `json:"failures"`
}
type smtp2goFailure struct {
	Email   string `json:"email"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func SMTP2GO1(app *common.App) *messengers.Definition {
	fromAddress, stdErr := stdmail.ParseAddress(app.Env.SMTP2GO_FROM_EMAIL)
	if stdErr != nil {
		log.Fatalf("invalid SMTP2GO_FROM_EMAIL: %v", stdErr)
	}

	from := stdmail.Address{
		Name:    app.Env.SMTP2GO_FROM_NAME,
		Address: fromAddress.Address,
	}
	httpClient := &http.Client{}

	return &messengers.Definition{
		ID:                "smtp2go",
		Version:           1,
		Name:              "Email",
		OptionsType:       &SMTP2GO1Options{},
		OptionsSchemaPath: "smtp2go/v1/options.schema.json",
		Prepare: func(prepareCtx *messengers.PrepareContext) (any, error) {
			options := &SMTP2GO1Options{}
			wrappedErr := prepareCtx.DecodeOptions(options)
			if wrappedErr != nil {
				return nil, wrappedErr
			}

			toAddress, stdErr := stdmail.ParseAddress(options.Email)
			if stdErr != nil {
				return nil, stdErr
			}
			formattedMessage, wrappedErr := messengers.FormatDefaultMessage(prepareCtx.Message)
			if wrappedErr != nil {
				return nil, wrappedErr
			}

			return &SMTP2GO1Body{
				ToAddress:        *toAddress,
				Subject:          formattedMessage.Subject,
				FormattedMessage: formattedMessage.Body,
			}, nil
		},
		BodyType: &SMTP2GO1Body{},
		Handler: func(messengerCtx *messengers.Context) error {
			body := &SMTP2GO1Body{}
			wrappedErr := messengerCtx.Decode(body)
			if wrappedErr != nil {
				return wrappedErr
			}

			requestBody, stdErr := json.Marshal(&smtp2goSendRequest{
				Sender:   from.String(),
				To:       []string{body.ToAddress.Address},
				Subject:  body.Subject,
				TextBody: body.FormattedMessage,
			})
			if stdErr != nil {
				return ErrWrapperSMTP2GO.Wrap(stdErr)
			}

			req, stdErr := http.NewRequestWithContext(
				messengerCtx.Context,
				http.MethodPost,
				app.Env.SMTP2GO_BASE_URL+"/email/send",
				bytes.NewReader(requestBody),
			)
			if stdErr != nil {
				return ErrWrapperSMTP2GO.Wrap(stdErr)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Smtp2go-Api-Key", app.Env.SMTP2GO_API_KEY)

			resp, stdErr := httpClient.Do(req)
			if stdErr != nil {
				return ErrWrapperSMTP2GO.Wrap(stdErr)
			}
			defer resp.Body.Close()
			respBytes, stdErr := io.ReadAll(io.LimitReader(resp.Body, smtp2goResponseBodyByteLimit))
			if stdErr != nil {
				return ErrWrapperSMTP2GO.Wrap(stdErr)
			}

			if resp.StatusCode != http.StatusOK {
				return ErrWrapperSMTP2GO.Wrap(common.NewWrappedRESTError(resp))
			}
			apiResponseOb := &smtp2goSendResponse{}
			stdErr = json.Unmarshal(respBytes, apiResponseOb)
			if stdErr != nil {
				return ErrWrapperSMTP2GO.Wrap(stdErr)
			}

			if apiResponseOb.Data.Succeeded < 1 {
				return ErrWrapperSMTP2GO.Wrap(fmt.Errorf(
					"smtp2go email failed but with 200 status. response body:\n%v",
					string(respBytes),
				))
			}

			messengerCtx.ConfirmSent()
			return nil
		},
	}
}
