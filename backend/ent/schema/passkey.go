package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Passkey holds the schema definition for a WebAuthn credential belonging to a user.
type Passkey struct {
	ent.Schema
}

func (Passkey) Fields() []ent.Field {
	// TODO: encrypt some of this with server key, as webauthn.Credential suggests
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.String("name").MinLen(1).MaxLen(64),
		field.Bytes("credentialID").Unique(), // TODO: add max length
		field.Bytes("publicKey"),
		field.UUID("aaguid", uuid.Nil).Optional(),
		field.Uint32("signCount").Default(0),
		field.Bool("isSecondGroup").Default(false),
		field.UUID("userID", uuid.Nil),
	}
}

func (Passkey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("passkeys").
			Field("userID").Unique().Required(),
		edge.To("sessions", Session.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// TODO: add index for userID + credentialID?
