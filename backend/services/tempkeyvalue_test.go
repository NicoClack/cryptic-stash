package services_test

import (
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/services"
	"github.com/NicoClack/cryptic-stash/backend/tempkeyvalue"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestTempKeyValueService_GivenValidDefinition_SetsAndGetsValue(t *testing.T) {
	t.Parallel()
	clock := clockwork.NewFakeClock()
	app := &common.App{Clock: clock}
	service := services.NewTempKeyValue(app, func(group *tempkeyvalue.RegistryGroup) {
		group.Register(&tempkeyvalue.Definition{
			Name: "storeName",
			Type: "",
		})
	})

	service.Set("storeName", "key", "value", clock.Now().Add(time.Minute))

	var value string
	exists := service.Get("storeName", "key", &value)
	require.True(t, exists)
	require.Equal(t, "value", value)
}

func TestTempKeyValueServicePruneAll(t *testing.T) {
	t.Parallel()
	clock := clockwork.NewFakeClock()
	app := &common.App{Clock: clock}
	service := services.NewTempKeyValue(app, func(group *tempkeyvalue.RegistryGroup) {
		group.Register(&tempkeyvalue.Definition{
			Name: "storeName",
			Type: "",
		})
	})

	service.Set("storeName", "expiredKey", "expiredValue", clock.Now().Add(500*time.Millisecond))
	service.Set("storeName", "activeKey", "activeValue", clock.Now().Add(5*time.Second))

	clock.Advance(2 * time.Second)
	service.PruneAll()

	var expiredValue string
	exists := service.Get("storeName", "expiredKey", &expiredValue)
	require.False(t, exists)

	var activeValue string
	exists = service.Get("storeName", "activeKey", &activeValue)
	require.True(t, exists)
	require.Equal(t, "activeValue", activeValue)
}
