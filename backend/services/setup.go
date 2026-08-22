package services

import (
	"context"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/setup"
)

type Setup struct {
	App *common.App
}

func NewSetupService(app *common.App) *Setup {
	return &Setup{
		App: app,
	}
}

func (service *Setup) GetStatus(ctx context.Context) (*common.SetupStatus, common.WrappedError) {
	return setup.GetStatus(
		ctx,
		service.App.Core.AdminID(),
		service.App.Messengers,
		service.App.Env,
	)
}

func (service *Setup) GenerateAdminSetupConstants(
	password string,
) (*common.AdminAuthEnvVars, string, common.WrappedError) {
	return setup.GenerateAdminSetupConstants(
		password,
		service.App.Env.ADMIN_PASSWORD_HASH_SETTINGS,
		service.App.Env.IS_DEV,
		service.App.Core,
	)
}

func (service *Setup) CheckTotpCode(totpCode string, totpSecret string) bool {
	return setup.CheckTotpCode(totpCode, totpSecret)
}
