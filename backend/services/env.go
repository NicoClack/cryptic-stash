package services

import (
	"log"
	"log/slog"
	"os"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/joho/godotenv"
)

func LoadEnvironmentVariables() *common.Env {
	_, isDevEnvDefined := os.LookupEnv("IS_DEV")
	if !isDevEnvDefined {
		stdErr := godotenv.Load(".env")
		if stdErr != nil {
			// The usual logger hasn't been created yet
			slog.Warn("error loading .env file", "error", stdErr)
		}
	}

	//exhaustruct:enforce
	env := &common.Env{
		IS_DEV:                        common.RequireBoolEnv("IS_DEV"),
		PORT:                          common.RequireIntEnv("PORT"),
		MOUNT_PATH:                    common.RequireEnv("MOUNT_PATH"),
		PROXY_ORIGINAL_IP_HEADER_NAME: common.RequireEnv("PROXY_ORIGINAL_IP_HEADER_NAME"),
		ALLOWED_ORIGINS:               common.RequireStrArrEnv("ALLOWED_ORIGINS"),
		CLEAN_UP_INTERVAL:             common.RequireSecondsEnv("CLEAN_UP_INTERVAL"),
		FULL_GC_INTERVAL:              common.RequireSecondsEnv("FULL_GC_INTERVAL"),

		JOB_POLL_INTERVAL:    common.RequireSecondsEnv("JOB_POLL_INTERVAL"),
		MAX_TOTAL_JOB_WEIGHT: common.RequireIntEnv("MAX_TOTAL_JOB_WEIGHT"),

		ADMIN_PASSWORD_HASH_SETTINGS: &common.PasswordHashSettings{
			Time:    common.RequireUint32Env("ADMIN_PASSWORD_HASH_TIME"),
			Memory:  common.RequireUint32Env("ADMIN_PASSWORD_HASH_MEMORY"),
			Threads: common.RequireUint8Env("ADMIN_PASSWORD_HASH_THREADS"),
		},
		ENABLE_ENV_SETUP:             common.RequireBoolEnv("ENABLE_ENV_SETUP"),
		ADMIN_CODE_ROTATION_INTERVAL: common.RequireSecondsEnv("ADMIN_CODE_ROTATION_INTERVAL"),
		ADMIN_PASSWORD_HASH:          common.OptionalBase64Env("ADMIN_PASSWORD_HASH", []byte{}),
		ADMIN_PASSWORD_SALT:          common.OptionalBase64Env("ADMIN_PASSWORD_SALT", []byte{}),
		ADMIN_TOTP_SECRET:            common.OptionalEnv("ADMIN_TOTP_SECRET", ""),

		UNLOCK_TIME:              common.RequireSecondsEnv("UNLOCK_TIME"),
		AUTH_CODE_VALID_FOR:      common.RequireSecondsEnv("AUTH_CODE_VALID_FOR"),
		USED_AUTH_CODE_VALID_FOR: common.RequireSecondsEnv("USED_AUTH_CODE_VALID_FOR"),
		ACTIVE_DOWNLOAD_SESSION_REMINDER_INTERVAL: common.RequireSecondsEnv(
			"ACTIVE_DOWNLOAD_SESSION_REMINDER_INTERVAL",
		),
		MIN_SUCCESSFUL_MESSAGE_COUNT: common.RequireIntEnv("MIN_SUCCESSFUL_MESSAGE_COUNT"),
		PASSWORD_HASH_SETTINGS: &common.PasswordHashSettings{
			Time:    common.RequireUint32Env("PASSWORD_HASH_TIME"),
			Memory:  common.RequireUint32Env("PASSWORD_HASH_MEMORY"),
			Threads: common.RequireUint8Env("PASSWORD_HASH_THREADS"),
		},

		LOG_STORE_INTERVAL:     common.RequireMillisecondsEnv("LOG_STORE_INTERVAL"),
		ADMIN_MESSAGE_TIMEOUT:  common.RequireSecondsEnv("ADMIN_MESSAGE_TIMEOUT"),
		MIN_ADMIN_MESSAGE_GAP:  common.RequireSecondsEnv("MIN_ADMIN_MESSAGE_GAP"),
		MIN_CRASH_SIGNAL_GAP:   common.RequireSecondsEnv("MIN_CRASH_SIGNAL_GAP"),
		PANIC_ON_ERROR:         common.OptionalBoolEnv("PANIC_ON_ERROR", false),
		MESSAGE_ADMIN_ON_ERROR: common.OptionalBoolEnv("MESSAGE_ADMIN_ON_ERROR", true),

		ENABLE_DEVELOP_MESSENGER: common.OptionalBoolEnv("ENABLE_DEVELOP_MESSENGER", false),
		DISCORD_TOKEN:            common.OptionalEnv("DISCORD_TOKEN", ""),
		SENDGRID_TOKEN:           common.OptionalEnv("SENDGRID_TOKEN", ""),
	}
	ValidateEnvironmentVariables(env)
	return env
}
func ValidateEnvironmentVariables(env *common.Env) {
	if !common.AllOrNone(
		len(env.ADMIN_PASSWORD_HASH) == 0,
		len(env.ADMIN_PASSWORD_SALT) == 0,
		env.ADMIN_TOTP_SECRET == "",
		env.ENABLE_ENV_SETUP,
	) {
		log.Fatal(
			"ADMIN_PASSWORD_HASH, ADMIN_PASSWORD_SALT and ADMIN_TOTP_SECRET must be all set and ENABLE_ENV_SETUP set to " +
				"false, or they must all be unset and ENABLE_ENV_SETUP set to true.",
		)
	}

	if float64(env.AUTH_CODE_VALID_FOR)/float64(env.UNLOCK_TIME) < 1.1 {
		log.Fatalf(
			"AUTH_CODE_VALID_FOR must be at least slightly larger than UNLOCK_TIME because a download requires " +
				"the auth code to be valid and the unlock time needs to have passed",
		)
	}

	if env.ENABLE_ENV_SETUP {
		slog.Warn("setup mode is enabled. please complete the setup in the app and avoid leaving it in this state.")
	}
}
