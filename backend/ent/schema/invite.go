package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// Invite holds the schema definition for the Invite entity.
type Invite struct {
	ent.Schema
}

// Fields of the Invite.
func (Invite) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.String("email").MinLen(3).MaxLen(128),
		field.Bytes("hashedCode"). // Using SHA-256
						Unique().
						MinLen(32).
						MaxLen(32),
		field.Time("expiresAt"),
		// Used to make an invite expire early without userID being set.
		field.Enum("expiredReason").
			Values("revoked", "username_taken").
			Optional().Nillable(),
		field.Bytes("webAuthnSession").
			GoType(&webauthn.SessionData{}).
			ValueScanner(EncryptedField[*webauthn.SessionData]{KeyName: "auth_1"}).
			Optional().Nillable(),

		// Details about the client that used the invite, rather than the creator
		field.Bytes("userAgent").
			GoType(new(string)).
			ValueScanner(EncryptedField[*string]{
				KeyName: "security_pii_logging_1",
				Validators: []EncryptedValidator[*string]{
					MinLen[*string](1),
					MaxLen[*string](512),
				},
			}).
			Optional().Nillable(),
		field.Bytes("ip").
			GoType(new(string)).
			ValueScanner(EncryptedField[*string]{
				KeyName: "security_pii_logging_1",
				Validators: []EncryptedValidator[*string]{
					MinLen[*string](1),
					MaxLen[*string](45),
				},
			}).
			Optional().Nillable(),

		field.UUID("userID", uuid.Nil).Optional(), // The user that was created by this invite, if any
	}
}

// Edges of the Invite.
func (Invite) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("invite").
			Field("userID").Unique(),
	}
}

func (Invite) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("hashedCode"),
		index.Fields("createdAt"),
	}
}
