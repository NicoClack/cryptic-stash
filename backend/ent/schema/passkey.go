package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// Passkey holds the schema definition for a WebAuthn credential belonging to a user.
type Passkey struct {
	ent.Schema
}

func (Passkey) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.String("name").MinLen(1).MaxLen(64),
		field.Bool("allowSudo"),
		field.Bytes("credentialID").Unique().MinLen(16).MaxLen(1023),
		field.Bytes("credential").
			GoType(webauthn.Credential{}).
			ValueScanner(EncryptedField[webauthn.Credential]{KeyName: "auth_1"}),
		field.Bool("isSecondGroup").Default(false),
		field.Time("lastUsedAt").Optional(),
		field.UUID("userID", uuid.Nil),
	}
}

func (Passkey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("passkeys").
			Field("userID").Unique().Required(),
		edge.To("sessions", Session.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("elevatedSessions", Session.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		// ^ Ideally we'd demote these sessions but sessions are short lived, so it's fine if we just delete them.
		// The user is unlikely to be logged into multiple devices simultaneously
	}
}

func (Passkey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("userID", "credentialID"),
		index.Fields("userID", "name").Unique(),
	}
}
