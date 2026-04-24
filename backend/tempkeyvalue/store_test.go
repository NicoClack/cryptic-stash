package tempkeyvalue_test

import (
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/tempkeyvalue"
	"github.com/stretchr/testify/require"
)

func TestTempKeyValue_SetAndGet(t *testing.T) {
	t.Parallel()
	store := tempkeyvalue.NewStore[string]()
	now := time.Now()
	store.Set("key", "value", now.Add(time.Minute))

	value, ok := store.Get("key", now)
	require.True(t, ok)
	require.Equal(t, "value", value)
}

func TestTempKeyValue_Delete(t *testing.T) {
	t.Parallel()
	store := tempkeyvalue.NewStore[string]()
	store.Set("key", "value", time.Now().Add(time.Minute))
	store.Delete("key")

	_, ok := store.Get("key", time.Now())
	require.False(t, ok)
}

func TestTempKeyValue_GetExpired_DeletesValue(t *testing.T) {
	t.Parallel()
	store := tempkeyvalue.NewStore[string]()
	now := time.Now()
	store.Set("key", "value", now.Add(-time.Second))

	_, ok := store.Get("key", now)
	require.False(t, ok)
}
