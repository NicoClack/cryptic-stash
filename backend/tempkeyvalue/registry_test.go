package tempkeyvalue_test

import (
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/tempkeyvalue"
	"github.com/stretchr/testify/require"
)

func TestRegistryGet_PanicsOnUnknownStoreName(t *testing.T) {
	t.Parallel()
	registry := tempkeyvalue.NewRegistry()

	require.Panics(t, func() {
		var value string
		registry.Get("missing", "key", &value, time.Now())
	})
}

func TestRegistrySet_PanicsOnWrongValueType(t *testing.T) {
	t.Parallel()
	registry := tempkeyvalue.NewRegistry()
	registry.Register(&tempkeyvalue.Definition{
		Name: "storeName",
		Type: "",
	})

	require.Panics(t, func() {
		registry.Set("storeName", "key", 123, time.Now().Add(time.Minute))
	})
}

func TestRegistryGet_PanicsOnWrongPointerType(t *testing.T) {
	t.Parallel()
	registry := tempkeyvalue.NewRegistry()
	registry.Register(&tempkeyvalue.Definition{
		Name: "storeName",
		Type: "",
	})

	require.Panics(t, func() {
		var value int
		registry.Get("storeName", "key", &value, time.Now())
	})
}

func TestRegistrySetAndGet(t *testing.T) {
	t.Parallel()
	registry := tempkeyvalue.NewRegistry()
	registry.Register(&tempkeyvalue.Definition{
		Name: "storeName",
		Type: "",
	})
	now := time.Now()
	registry.Set("storeName", "key", "value", now.Add(time.Minute))

	var value string
	exists := registry.Get("storeName", "key", &value, now)
	require.True(t, exists)
	require.Equal(t, "value", value)
}

func TestRegistryPruneAll_RemovesExpiredValues(t *testing.T) {
	t.Parallel()
	registry := tempkeyvalue.NewRegistry()
	registry.Register(&tempkeyvalue.Definition{
		Name: "storeName",
		Type: "",
	})
	now := time.Now()
	registry.Set("storeName", "activeKey", "value", now.Add(time.Second))
	registry.Set("storeName", "expiredKey", "value", now.Add(-time.Second))
	registry.PruneAll(now)

	var value string
	exists := registry.Get("storeName", "activeKey", &value, now)
	require.True(t, exists)
	require.Equal(t, "value", value)

	exists = registry.Get("storeName", "expiredKey", &value, now)
	require.False(t, exists)
}
