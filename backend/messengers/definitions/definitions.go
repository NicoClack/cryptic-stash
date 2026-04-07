package definitions

import (
	"embed"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/messengers"
)

var (
	//go:embed assets
	assetsFs embed.FS
)
var AssetFiles = common.EmbeddedDirectory{
	Path: "assets",
	FS:   assetsFs,
}

func Register(registry *messengers.Registry) {
	env := registry.App.Env

	registry.SetEmbeddedDir(AssetFiles)
	if env.ENABLE_DEVELOP_MESSENGER {
		registry.Register(Develop1())
	}
	if env.DISCORD_TOKEN != "" {
		registry.Register(Discord1(registry.App))
	}
	if env.SMTP_HOST != "" {
		registry.Register(SMTP1(registry.App))
	}
}
