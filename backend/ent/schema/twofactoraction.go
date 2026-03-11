package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// TwoFactorAction holds the schema definition for the TwoFactorAction entity.
type TwoFactorAction struct {
	ent.Schema
}

// Fields of the TwoFactorAction.
func (TwoFactorAction) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.Nil).Default(uuid.New),
		field.Time("createdAt"),
		field.Time("updatedAt").UpdateDefault(time.Now),
		field.String("type").MinLen(1).MaxLen(128),
		field.Int("version"),
		field.JSON("body", json.RawMessage{}),
		field.Time("expiresAt"),
		field.String("code").MaxLen(9).MaxLen(9),
	}
}

// Edges of the TwoFactorAction.
func (TwoFactorAction) Edges() []ent.Edge {
	return nil
}

func (TwoFactorAction) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code"),
	}
}
