package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SignupLink holds the schema definition for the SignupLink entity.
type SignupLink struct {
	ent.Schema
}

// Fields of the SignupLink.
func (SignupLink) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		// For differentiating between signup links and to provide a suggested username
		field.String("name").MaxLen(32),
		field.Bytes("code"). // Hashed with SHA256
					Unique().
					MinLen(32).
					MaxLen(32),
		field.Time("expiresAt"),
		field.String("userAgent").Default(""),
		field.String("ip").Default(""),
		field.UUID("userID", uuid.Nil).Optional(), // The user that was created by this signup link, if any
	}
}

// Edges of the SignupLink.
func (SignupLink) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("signupLink").
			Field("userID").Unique(),
	}
}

func (SignupLink) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code"),
	}
}
