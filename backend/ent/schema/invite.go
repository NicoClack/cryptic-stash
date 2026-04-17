package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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
		field.Bytes("webauthnChallenge").Optional().Nillable().
			MinLen(32).MaxLen(32),
		field.Time("challengeExpiresAt").Optional().Nillable(),
		field.String("userAgent").Default(""),
		field.String("ip").Default(""),
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
