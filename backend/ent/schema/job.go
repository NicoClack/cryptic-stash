package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Job holds the schema definition for the Job entity.
type Job struct {
	ent.Schema
}

// Fields of the Job.
func (Job) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.Time("dueAt"),
		field.Time("originallyDueAt"), // Due is updated for retries
		field.Time("startedAt").Optional(),
		field.String("type").MinLen(1).MaxLen(128),
		field.Int("version"),
		field.Int8(
			"priority",
		), // Currently duplicates the definition but needed for sorting and might want to make it dynamic in the future
		field.Int("weight"), // Currently duplicates the definition but might make it dynamic in the future
		field.JSON("body", json.RawMessage{}),
		field.Enum("status").Values("pending", "running", "failed").Default("pending"), // Completed jobs are deleted
		field.Int("retries").Default(0),
		field.Float("retriedFraction").Default(0),
		field.Bool("loggedStallWarning").Default(false),
	}
}

// Edges of the Job.
func (Job) Edges() []ent.Edge {
	return nil
}

func (Job) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "priority", "dueAt"),
		index.Fields("dueAt"),
	}
}
