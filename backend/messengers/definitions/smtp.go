package definitions

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	stdmail "net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/messengers"
)

const (
	smtpDialTimeout = 5 * time.Second
)

var ErrWrapperSMTP = common.NewDynamicErrorWrapper(func(err error) common.WrappedError {
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

	var protocolErr *textproto.Error
	if errors.As(err, &protocolErr) {
		// Note: SMTP statuses work slightly differently to HTTP
		if protocolErr.Code >= 400 && protocolErr.Code < 500 {
			wrappedErr.ConfigureRetriesMut(10, 5*time.Second, 2)
			wrappedErr.AddDebugValuesMut(common.DebugValue{
				Name:    "retried SMTP 4xx response",
				Message: fmt.Sprintf("status code: %d", protocolErr.Code),
			})
		}
		return wrappedErr
	}

	return wrappedErr
})

type SMTP1Options struct {
	Email string `json:"email"`
}

type SMTP1Body struct {
	ToAddress        stdmail.Address `json:"toAddress"`
	Subject          string          `json:"subject"`
	FormattedMessage string          `json:"formattedMessage"`
}

func formatSMTPMessage(fromAddress stdmail.Address, toAddress stdmail.Address, subject string, body string) []byte {
	messageBuffer := bytes.NewBuffer(nil)
	messageBuffer.WriteString("From: ")
	messageBuffer.WriteString(fromAddress.String())
	messageBuffer.WriteString("\r\n")
	messageBuffer.WriteString("To: ")
	messageBuffer.WriteString(toAddress.String())
	messageBuffer.WriteString("\r\n")
	messageBuffer.WriteString("Subject: ")
	messageBuffer.WriteString(subject)
	messageBuffer.WriteString("\r\n")
	messageBuffer.WriteString("MIME-Version: 1.0\r\n")
	messageBuffer.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	messageBuffer.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")

	messageBuffer.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
	return messageBuffer.Bytes()
}

func SMTP1(app *common.App) *messengers.Definition {
	fromAddress, stdErr := stdmail.ParseAddress(app.Env.SMTP_FROM_EMAIL)
	if stdErr != nil {
		log.Fatalf("invalid SMTP_FROM_EMAIL: %v", stdErr)
	}

	hostPort := net.JoinHostPort(app.Env.SMTP_HOST, strconv.Itoa(app.Env.SMTP_PORT))
	from := stdmail.Address{
		Name:    app.Env.SMTP_FROM_NAME,
		Address: fromAddress.Address,
	}

	return &messengers.Definition{
		ID:                "smtp",
		Version:           1,
		Name:              "Email",
		OptionsType:       &SMTP1Options{},
		OptionsSchemaPath: "smtp/v1/options.schema.json",
		Prepare: func(prepareCtx *messengers.PrepareContext) (any, error) {
			options := &SMTP1Options{}
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

			return &SMTP1Body{
				ToAddress:        *toAddress,
				Subject:          formattedMessage.Subject,
				FormattedMessage: formattedMessage.Body,
			}, nil
		},
		BodyType: &SMTP1Body{},
		Handler: func(messengerCtx *messengers.Context) error {
			body := &SMTP1Body{}
			wrappedErr := messengerCtx.Decode(body)
			if wrappedErr != nil {
				return wrappedErr
			}

			dialer := &net.Dialer{Timeout: smtpDialTimeout}

			var connection net.Conn
			if app.Env.SMTP_IMPLICIT_TLS {
				tlsDialer := &tls.Dialer{
					NetDialer: dialer,
					Config: &tls.Config{
						ServerName: app.Env.SMTP_HOST,
						MinVersion: tls.VersionTLS12,
					},
				}
				tlsConn, stdErr := tlsDialer.DialContext(messengerCtx.Context, "tcp", hostPort)
				if stdErr != nil {
					return ErrWrapperSMTP.Wrap(stdErr)
				}
				connection = tlsConn
			} else {
				var stdErr error
				connection, stdErr = dialer.DialContext(messengerCtx.Context, "tcp", hostPort)
				if stdErr != nil {
					return ErrWrapperSMTP.Wrap(stdErr)
				}
			}

			wrappedErr = common.AddContextToConnection(connection, messengerCtx.Context)
			if wrappedErr != nil {
				_ = connection.Close()
				return ErrWrapperSMTP.Wrap(wrappedErr)
			}

			client, stdErr := smtp.NewClient(connection, app.Env.SMTP_HOST)
			if stdErr != nil {
				_ = connection.Close()
				return ErrWrapperSMTP.Wrap(stdErr)
			}
			defer client.Close()

			if !app.Env.SMTP_IMPLICIT_TLS {
				if ok, _ := client.Extension("STARTTLS"); ok {
					stdErr = client.StartTLS(&tls.Config{
						ServerName: app.Env.SMTP_HOST,
						MinVersion: tls.VersionTLS12,
					})
					if stdErr != nil {
						return ErrWrapperSMTP.Wrap(stdErr)
					}
				} else if app.Env.SMTP_REQUIRE_TLS {
					_ = client.Quit()
					return ErrWrapperSMTP.Wrap(
						fmt.Errorf("SMTP_REQUIRE_TLS is true but SMTP server doesn't support STARTTLS"),
					)
				}
			}

			stdErr = client.Auth(smtp.PlainAuth(
				"",
				app.Env.SMTP_USERNAME,
				app.Env.SMTP_PASSWORD,
				app.Env.SMTP_HOST,
			))
			if stdErr != nil {
				return ErrWrapperSMTP.Wrap(stdErr)
			}

			stdErr = client.Mail(from.Address)
			if stdErr != nil {
				return ErrWrapperSMTP.Wrap(stdErr)
			}
			stdErr = client.Rcpt(body.ToAddress.Address)
			if stdErr != nil {
				return ErrWrapperSMTP.Wrap(stdErr)
			}
			writer, stdErr := client.Data()
			if stdErr != nil {
				return ErrWrapperSMTP.Wrap(stdErr)
			}
			_, stdErr = writer.Write(
				formatSMTPMessage(from, body.ToAddress, body.Subject, body.FormattedMessage),
			)
			if stdErr != nil {
				_ = writer.Close()
				return ErrWrapperSMTP.Wrap(stdErr)
			}
			stdErr = writer.Close()
			if stdErr != nil {
				return ErrWrapperSMTP.Wrap(stdErr)
			}

			stdErr = client.Quit()
			if stdErr != nil {
				return ErrWrapperSMTP.Wrap(stdErr)
			}
			messengerCtx.ConfirmSent()
			return nil
		},
	}
}
