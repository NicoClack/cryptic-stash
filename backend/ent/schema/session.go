package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Session holds the schema definition for a user login session.
type Session struct {
	ent.Schema
}

func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.Bytes("hashedToken"). // Using SHA-256
						Unique().
						MinLen(32).
						MaxLen(32),
		field.Time("expiresAt"),
		field.String("userAgent").Default(""),
		field.String("ip").Default(""),
		field.UUID("userID", uuid.Nil),
	}
}

func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("sessions").
			Field("userID").Required().Unique(),
	}
}

func (Session) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("hashedToken", "userID"),
	}
}
