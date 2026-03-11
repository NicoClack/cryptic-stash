package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// KeyValue holds the schema definition for the KeyValue entity.
type KeyValue struct {
	ent.Schema
}

// Fields of the KeyValue.
func (KeyValue) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.String("key").MinLen(1).MaxLen(128),
		field.JSON("value", json.RawMessage{}),
	}
}

// Edges of the KeyValue.
func (KeyValue) Edges() []ent.Edge {
	return nil
}

func (KeyValue) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("key").Unique(),
	}
}
