package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// UserMessenger holds the schema definition for the UserMessenger entity.
type UserMessenger struct {
	ent.Schema
}

// Fields of the UserMessenger.
func (UserMessenger) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.String("type").MinLen(1).MaxLen(128),
		field.Int("version"),
		field.Bool("enabled").Default(true),
		field.JSON("options", json.RawMessage{}),
		field.UUID("userID", uuid.Nil),
	}
}

// Edges of the UserMessenger.
func (UserMessenger) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("messengers").
			Field("userID").Unique().Required(),
		edge.To("loginAlerts", LoginAlert.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (UserMessenger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("userID", "type", "version").Unique(),
	}
}
