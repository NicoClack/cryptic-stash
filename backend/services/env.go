package services

import (
	"log"
	"log/slog"
	"os"
	"strings"

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

		INVITE_DEFAULT_EXPIRY: common.RequireSecondsEnv("INVITE_DEFAULT_EXPIRY"),
		INVITE_MAX_EXPIRY:     common.RequireSecondsEnv("INVITE_MAX_EXPIRY"),

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
		STASH_ENCRYPTION_KEY: common.RequireBase64Env("STASH_ENCRYPTION_KEY"),

		LOG_STORE_INTERVAL:     common.RequireMillisecondsEnv("LOG_STORE_INTERVAL"),
		ADMIN_MESSAGE_TIMEOUT:  common.RequireSecondsEnv("ADMIN_MESSAGE_TIMEOUT"),
		MIN_ADMIN_MESSAGE_GAP:  common.RequireSecondsEnv("MIN_ADMIN_MESSAGE_GAP"),
		MIN_CRASH_SIGNAL_GAP:   common.RequireSecondsEnv("MIN_CRASH_SIGNAL_GAP"),
		PANIC_ON_ERROR:         common.OptionalBoolEnv("PANIC_ON_ERROR", false),
		MESSAGE_ADMIN_ON_ERROR: common.OptionalBoolEnv("MESSAGE_ADMIN_ON_ERROR", true),

		ENABLE_DEVELOP_MESSENGER: common.OptionalBoolEnv("ENABLE_DEVELOP_MESSENGER", false),
		DISCORD_TOKEN:            common.OptionalEnv("DISCORD_TOKEN", ""),
		SMTP_HOST:                common.OptionalEnv("SMTP_HOST", ""),
		SMTP_PORT:                common.OptionalIntEnv("SMTP_PORT", 0),
		SMTP_USERNAME:            common.OptionalEnv("SMTP_USERNAME", ""),
		SMTP_PASSWORD:            common.OptionalEnv("SMTP_PASSWORD", ""),
		SMTP_FROM_EMAIL:          common.OptionalEnv("SMTP_FROM_EMAIL", ""),
		SMTP_FROM_NAME:           common.OptionalEnv("SMTP_FROM_NAME", "Cryptic Stash"),
		SMTP_REQUIRE_TLS:         common.OptionalBoolEnv("SMTP_REQUIRE_TLS", true),
		SMTP_IMPLICIT_TLS:        common.OptionalBoolEnv("SMTP_IMPLICIT_TLS", true),
		SMTP2GO_API_KEY:          common.OptionalEnv("SMTP2GO_API_KEY", ""),
		SMTP2GO_BASE_URL:         common.OptionalEnv("SMTP2GO_BASE_URL", "https://api.smtp2go.com/v3"),
		SMTP2GO_FROM_EMAIL:       common.OptionalEnv("SMTP2GO_FROM_EMAIL", ""),
		SMTP2GO_FROM_NAME:        common.OptionalEnv("SMTP2GO_FROM_NAME", "Cryptic Stash"),
	}
	NormalizeEnvironmentVariables(env)
	ValidateEnvironmentVariables(env)
	return env
}
func NormalizeEnvironmentVariables(env *common.Env) {
	env.SMTP2GO_BASE_URL = strings.TrimRight(strings.TrimSpace(env.SMTP2GO_BASE_URL), "/")
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

	if !common.AllOrNone(
		env.SMTP_HOST == "",
		env.SMTP_PORT == 0,
		env.SMTP_FROM_EMAIL == "",
	) {
		log.Fatal(
			"SMTP_HOST, SMTP_PORT, SMTP_FROM_EMAIL must either all be set or all be unset.",
		)
	}
	if !env.SMTP_REQUIRE_TLS && !env.IS_DEV {
		slog.Warn("SMTP_REQUIRE_TLS should be set to true for production environments")
	}

	if !common.AllOrNone(
		env.SMTP2GO_API_KEY == "",
		env.SMTP2GO_FROM_EMAIL == "",
	) {
		log.Fatal(
			"SMTP2GO_API_KEY and SMTP2GO_FROM_EMAIL must either both be set or both be unset.",
		)
	}

	if env.SMTP_HOST != "" && env.SMTP2GO_API_KEY != "" {
		slog.Warn(
			"You have both SMTP and SMTP2GO email options configured, which can be confusing for users. You might want to " +
				"migrate your users to one and disable the other.",
		)
	}
}
