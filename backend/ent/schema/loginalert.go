package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// LoginAlert holds the schema definition for the LoginAlert entity.
type LoginAlert struct {
	ent.Schema
}

// Fields of the LoginAlert.
func (LoginAlert) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("sentAt"),
		field.Bool("confirmed"),
		field.UUID("downloadSessionID", uuid.Nil),
	}
}

// Edges of the LoginAlert.
func (LoginAlert) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("downloadSession", DownloadSession.Type).Ref("loginAlerts").
			Field("downloadSessionID").Unique().Required(),
	}
}
