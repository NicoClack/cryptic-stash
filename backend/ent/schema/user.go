package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
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
		field.String("username").Unique().NotEmpty(),
		// Admins might be able to be locked in the future
		field.Bool("locked").Default(false),
		field.Time("lockedUntil").Nillable().Optional(),
		field.Time("downloadSessionsValidFrom"),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("stash", Stash.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).Unique(),
		edge.To("messengers", UserMessenger.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("downloadSessions", DownloadSession.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("signupLink", SignupLink.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).Unique(),
		edge.To("logs", LogEntry.Type).
			Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}
