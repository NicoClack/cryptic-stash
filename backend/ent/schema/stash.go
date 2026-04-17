package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Stash holds the schema definition for the Stash entity.
type Stash struct {
	ent.Schema
}

// Fields of the Stash.
func (Stash) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.Time("lastDownloadAt").Optional(),
		field.String("publicName").NotEmpty().MaxLen(256),
		// Encrypted with encryptionDataKey and prefixed with the nonce
		field.Bytes("content").NotEmpty().MaxLen(10_000_000), // 10MB
		field.Bytes("fileName").NotEmpty().MaxLen(256),

		// Encrypted with a key derived from the user's password, then env.STASH_ENCRYPTION_KEY.
		// GCM and nonce prefixes on both layers so the 32 unencrypted length becomes closer to 128 bytes
		field.Bytes("encryptionDataKey").MinLen(32).MaxLen(128),
		field.Bytes("passwordSalt").NotEmpty(),

		field.Uint32("hashTime"),
		field.Uint32("hashMemory"),
		field.Uint8("hashThreads"),

		field.Bool("selfLocked").Default(false),
		field.Bool("adminLocked").Default(false),
		// Creating a temporary lock won't update selfLocked
		field.Time("selfLockedUntil").Nillable().Optional(),
		field.Time("downloadSessionsValidFrom"),

		field.UUID("userID", uuid.Nil),
	}
}

// Edges of the Stash.
func (Stash) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("stashes").
			Field("userID").Unique().Required(),
		edge.To("downloadSessions", DownloadSession.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
