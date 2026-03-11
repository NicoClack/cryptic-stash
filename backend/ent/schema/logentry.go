package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// LogEntry holds the schema definition for the LogEntry entity.
type LogEntry struct {
	ent.Schema
}

// Fields of the LogEntry.
func (LogEntry) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.Time("loggedAt"),      // The entries should be batch created, so a default time wouldn't be accurate
		field.Bool("loggedAtKnown"), // Some logs don't have a time, so an inaccurate time is added during processing
		field.Int("level"),
		field.String("message"),
		field.JSON("attributes", map[string]any{}),
		field.String("sourceFile"),
		field.String("sourceFunction"),
		field.Int("sourceLine"),
		field.String("publicMessage"),
		field.UUID("userID", uuid.Nil).Optional(),
	}
}

// Edges of the LogEntry.
func (LogEntry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("logs").
			Field("userID").Unique(),
	}
}

func (LogEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("loggedAt"),
	}
}
