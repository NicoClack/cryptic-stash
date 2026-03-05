package testhelpers

import (
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/messengers"
	"github.com/google/uuid"
)

type MockMessenger struct {
	Name         string
	SentMessages []SentMessage
}
type SentMessage struct {
	Type   string
	UserID uuid.UUID
}

func NewMockMessenger(name string) *MockMessenger {
	return &MockMessenger{
		Name:         name,
		SentMessages: []SentMessage{},
	}
}

func (mockMessenger *MockMessenger) Register(registry *messengers.Registry) {
	type Body struct {
		Message SentMessage
	}
	registry.Register(&messengers.Definition{
		ID:      mockMessenger.Name,
		Version: 1,
		Prepare: func(prepareCtx *messengers.PrepareContext) (any, error) {
			return &Body{
				Message: SentMessage{
					Type:   string(prepareCtx.Message.Type),
					UserID: prepareCtx.Message.User.ID,
				},
			}, nil
		},
		BodyType: &Body{},
		Handler: func(messengerCtx *messengers.Context) error {
			body := Body{}
			wrappedErr := messengerCtx.Decode(&body)
			if wrappedErr != nil {
				return wrappedErr
			}

			mockMessenger.SentMessages = append(mockMessenger.SentMessages, body.Message)
			messengerCtx.ConfirmSent()
			return nil
		},
	})
}
func (mockMessenger *MockMessenger) VersionedName() string {
	return common.GetVersionedType(mockMessenger.Name, 1)
}
