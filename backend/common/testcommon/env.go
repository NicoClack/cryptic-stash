package testcommon

import (
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
)

func DefaultEnv() *common.Env {
	//exhaustruct:enforce
	return &common.Env{
		IS_DEV:                        true,
		PORT:                          -1,
		MOUNT_PATH:                    "temp-test-storage",
		PROXY_ORIGINAL_IP_HEADER_NAME: "test-proxy-original-ip",
		ALLOWED_ORIGINS:               []string{},
		CLEAN_UP_INTERVAL:             time.Hour,
		FULL_GC_INTERVAL:              0,

		ADMIN_PASSWORD_HASH_SETTINGS: &common.PasswordHashSettings{
			Time:    1,
			Memory:  1024,
			Threads: 1,
		},
		ENABLE_ENV_SETUP:             false,
		ADMIN_CODE_ROTATION_INTERVAL: time.Hour,
		ADMIN_PASSWORD_HASH:          []byte{},
		ADMIN_PASSWORD_SALT:          []byte{},
		ADMIN_TOTP_SECRET:            "",

		JOB_POLL_INTERVAL:    time.Hour * 999,
		MAX_TOTAL_JOB_WEIGHT: 100,

		SIGNUP_LINK_DEFAULT_EXPIRY: time.Hour * 24 * 2,
		SIGNUP_LINK_MAX_EXPIRY:     time.Hour * 24 * 7,

		UNLOCK_TIME:                               time.Hour * 24 * 7,
		AUTH_CODE_VALID_FOR:                       time.Hour * 24 * 3,
		USED_AUTH_CODE_VALID_FOR:                  time.Hour,
		ACTIVE_DOWNLOAD_SESSION_REMINDER_INTERVAL: time.Hour * 24,
		MIN_SUCCESSFUL_MESSAGE_COUNT:              1,
		STASH_ENCRYPTION_KEY:                      []byte{},

		PASSWORD_HASH_SETTINGS: &common.PasswordHashSettings{
			Time:    1,
			Memory:  1024,
			Threads: 1,
		},

		LOG_STORE_INTERVAL:     time.Hour * 999,
		ADMIN_MESSAGE_TIMEOUT:  time.Minute,
		MIN_ADMIN_MESSAGE_GAP:  time.Minute * 5,
		MIN_CRASH_SIGNAL_GAP:   -1,
		PANIC_ON_ERROR:         true,
		MESSAGE_ADMIN_ON_ERROR: false,

		ENABLE_DEVELOP_MESSENGER: false,
		DISCORD_TOKEN:            "",
		SENDGRID_TOKEN:           "",
	}
}
