package definitions

import (
	"fmt"
	"os"

	"github.com/NicoClack/cryptic-stash/backend/messengers"
)

type Develop1Body struct {
	FullMessage string `json:"formattedMessage"`
}

func Develop1() *messengers.Definition {
	return &messengers.Definition{
		ID:      "develop",
		Version: 1,
		Name:    "Develop",
		Prepare: func(prepareCtx *messengers.PrepareContext) (any, error) {
			formattedMessage, wrappedErr := messengers.FormatDefaultMessage(prepareCtx.Message)
			if wrappedErr != nil {
				return nil, wrappedErr
			}
			return &Develop1Body{
				FullMessage: fmt.Sprintf(
					"\nmessage sent to user \"%v\":\n%v\n",
					prepareCtx.Message.User.Username, formattedMessage.Body,
				),
			}, nil
		},
		BodyType: &Develop1Body{},
		Handler: func(messengerCtx *messengers.Context) error {
			body := Develop1Body{}
			wrappedErr := messengerCtx.Decode(&body)
			if wrappedErr != nil {
				return wrappedErr
			}

			fmt.Fprintln(os.Stdout, body.FullMessage)
			messengerCtx.ConfirmSent()
			return nil
		},
	}
}
