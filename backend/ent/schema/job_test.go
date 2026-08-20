package schema_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/stretchr/testify/require"
)

func TestJob_EncryptedFields(t *testing.T) {
	t.Parallel()

	t.Run("body can be read back", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()
		dbClient := db.Client()

		body := json.RawMessage(`{"foo":"bar","data":{"value":42}}`)
		now := time.Now()
		jobOb := dbClient.Job.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetType("test_job").
			SetVersion(1).
			SetPriority(1).
			SetWeight(1).
			SetBody(body).
			SetDueAt(now).
			SetOriginallyDueAt(now).
			SaveX(ctx)

		jobOb = dbClient.Job.GetX(ctx, jobOb.ID)
		require.Equal(t, string(body), string(jobOb.Body))
	})

	t.Run("json.RawMessage (body) is encrypted as raw JSON bytes, not double-encoded", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()

		body := json.RawMessage(`{"secret":"hidden-value"}`)
		now := time.Now()
		jobOb := db.Client().Job.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetType("test_job").
			SetVersion(1).
			SetPriority(1).
			SetWeight(1).
			SetBody(body).
			SetDueAt(now).
			SetOriginallyDueAt(now).
			SaveX(ctx)

		assertEncryptedInDB(t, db,
			"SELECT body FROM jobs WHERE id = ?", []any{jobOb.ID.String()},
			[]byte(body), "job_1",
			// ^ Not wrapped in JSON quotes
		)
	})
}
