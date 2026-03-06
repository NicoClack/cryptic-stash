package setup

import (
	"encoding/base64"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/pquerna/otp/totp"
)

func GenerateAdminSetupConstants(
	password string,
	passwordSettings *common.PasswordHashSettings,
	isDev bool,
	core common.CoreService,
) (
	*common.AdminAuthEnvVars,
	string,
	common.WrappedError,
) {
	issuer := "Cryptic Stash"
	if isDev {
		issuer = "Cryptic Stash (Development)"
	}
	key, stdErr := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: common.AdminUsername,
	})
	if stdErr != nil {
		return nil, "", ErrWrapperGenerateAdminSetupConstants.Wrap(
			ErrWrapperTotp.Wrap(stdErr),
		)
	}
	totpSecret := key.Secret()

	salt := core.GenerateSalt()
	adminPasswordHash := core.HashPassword(password, salt, passwordSettings)

	return &common.AdminAuthEnvVars{
			AdminPasswordHash: base64.StdEncoding.EncodeToString(adminPasswordHash),
			AdminPasswordSalt: base64.StdEncoding.EncodeToString(salt),
			AdminTotpSecret:   totpSecret,
			StashEncryptionKey: base64.StdEncoding.EncodeToString(
				common.CryptoRandomBytes(common.EncryptionKeyLength),
			),
		},
		key.URL(),
		nil
}

func CheckTotpCode(totpCode string, totpSecret string) bool {
	return totp.Validate(totpCode, totpSecret)
}
