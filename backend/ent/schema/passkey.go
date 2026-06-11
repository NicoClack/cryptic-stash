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

// Ent codegen has trouble without this alias
type EncryptedCredential struct {
	EncryptedField[webauthn.Credential]
}

func (Passkey) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.String("name").MinLen(1).MaxLen(64),
		field.Bool("allowSuperUser"),
		field.Bytes("credentialID").Unique().MinLen(16).MaxLen(1023),
		field.Bytes("credential").GoType(EncryptedCredential{
			EncryptedField: EncryptedField[webauthn.Credential]{
				KeyName: "auth_1",
			},
		}),
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

func (Passkey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("userID", "credentialID"),
		index.Fields("userID", "name").Unique(),
	}
}
