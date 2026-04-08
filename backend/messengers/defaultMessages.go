package messengers

import (
	"fmt"

	"github.com/NicoClack/cryptic-stash/backend/common"
)

type FormattedMessage struct {
	Subject string
	Body    string
}

func getLoginAttemptMessageBody(message *common.Message) string {
	explanation := "If you're reading this, it most likely wasn't you that's logging in! Please freeze your user " +
		"ASAP and contact your admin as they can make the lock permanent if you want. If you want to be able to safely " +
		"unlock your user, you should update your password with help from your admin."
	return fmt.Sprintf(
		"%v\n\nIF YOU DO NOTHING, we'll assume you're locked out and will ALLOW THE USER TO LOG IN after %v UTC.",
		explanation,
		message.Time.Format("2006-01-02 15:04:05"),
	)
}

var defaultMessageMap = map[common.MessageType]func(message *common.Message) *FormattedMessage{
	common.MessageInvite: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "You've been invited to Cryptic Stash",
			Body:    fmt.Sprintf("%v\nClick here to sign up: %v", message.InviteMessage, message.URL),
		}
	},
	common.MessageUserUpdate: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "Cryptic Stash account updated",
			Body:    "Your account password and/or stash have been updated by your admin.",
		}
	},
	common.MessageLogin: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "LOGIN ATTEMPT for Cryptic Stash",
			Body:    "LOGIN ATTEMPT! " + getLoginAttemptMessageBody(message),
		}
	},
	common.MessageActiveDownloadSessionReminder: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "LOGIN ATTEMPT REMINDER for Cryptic Stash",
			Body:    "REMINDER: YOU HAVE A PENDING LOGIN ATTEMPT! " + getLoginAttemptMessageBody(message),
		}
	},
	common.MessageDownload: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "Cryptic Stash download completed",
			Body: "Your stash has been downloaded. If this wasn't you, " +
				"please rotate your 2FA backup codes immediately and contact your admin!",
		}
	},
	common.MessageTest: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "Cryptic Stash test message",
			Body:    "If you're reading this message, it means your updated contacts are working.",
		}
	},
	common.Message2FA: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "Cryptic Stash 2FA code",
			Body:    fmt.Sprintf("2FA code: %s", message.Code),
		}
	},
	common.MessageLock: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "Cryptic Stash account locked",
			Body: "Your account has been locked by your admin, this will replace your self lock if you have one. " +
				"The lock will remain until your admin removes it.",
		}
	},
	common.MessageUnlock: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "Cryptic Stash account unlocked",
			Body:    "Your account has been unlocked by your admin, you (or anyone else) can now try to log in again.",
		}
	},
	common.MessageSelfLock: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "Cryptic Stash self-lock confirmation",
			Body:    fmt.Sprintf("You have locked your account until %s", message.Time.Format("2006-01-02 15:04:05")),
		}
	},
	common.MessageSelfUnlock: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "Cryptic Stash self-lock expired",
			Body:    "Warning: your account freeze has expired, you (or anyone else) can now try to log in again.",
		}
	},
	common.MessageAdminError: func(message *common.Message) *FormattedMessage {
		return &FormattedMessage{
			Subject: "[Admin] Cryptic Stash error",
			Body: "[Admin] An error has occurred! Please investigate the logs and possibly create an issue at " +
				"https://github.com/NicoClack/cryptic-stash/backend/issues as this might be reducing security",
		}
	},
}

// For messengers like SMS where the messages should be as short as possible with no formatting
func FormatDefaultMessage(message *common.Message) (*FormattedMessage, common.WrappedError) {
	formatter, ok := defaultMessageMap[message.Type]
	if !ok {
		return nil, ErrWrapperFormatMessage.Wrap(
			fmt.Errorf("message type \"%v\" hasn't been implemented", message.Type),
		)
	}

	return formatter(message), nil
}
