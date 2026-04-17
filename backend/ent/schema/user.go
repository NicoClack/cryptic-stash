package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.String("username").Unique().NotEmpty(),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("stashes", Stash.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("messengers", UserMessenger.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("invite", Invite.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).Unique(),
		edge.To("passkeys", Passkey.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("sessions", Session.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("logs", LogEntry.Type).
			Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("createdAt"),
	}
}
