package services

import (
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/tempkeyvalue"
	"github.com/NicoClack/cryptic-stash/backend/tempkeyvalue/definitions"
)

type TempKeyValue struct {
	App      *common.App
	Registry *tempkeyvalue.Registry
}

func NewTempKeyValue(
	app *common.App,
	registerFuncs ...func(group *tempkeyvalue.RegistryGroup),
) *TempKeyValue {
	registry := tempkeyvalue.NewRegistry()
	definitions.Register(registry.Group(""))
	for _, registerFunc := range registerFuncs {
		registerFunc(registry.Group(""))
	}

	return &TempKeyValue{
		App:      app,
		Registry: registry,
	}
}

func (service *TempKeyValue) Get(storeName string, key string, ptr any) bool {
	return service.Registry.Get(storeName, key, ptr, service.App.Clock.Now())
}
func (service *TempKeyValue) Set(storeName string, key string, value any, expiresAt time.Time) {
	service.Registry.Set(storeName, key, value, expiresAt)
}
func (service *TempKeyValue) Delete(storeName string, key string) {
	service.Registry.Delete(storeName, key)
}
func (service *TempKeyValue) Prune(storeName string) {
	service.Registry.Prune(storeName, service.App.Clock.Now())
}
func (service *TempKeyValue) PruneAll() {
	service.Registry.PruneAll(service.App.Clock.Now())
}
