package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// RefreshToken holds the schema definition for the RefreshToken entity.
type RefreshToken struct {
	ent.Schema
}

// Fields of the RefreshToken.
func (RefreshToken) Fields() []ent.Field {
	return []ent.Field{
		//UUIDv7
		field.String("id").NotEmpty().Immutable().Unique(),
		field.String("token_hash").NotEmpty().Immutable().Unique(), // is it unique though?
		field.String("family_id").NotEmpty().Immutable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("expires_at").Immutable(),
		field.Time("revoked_at").Optional().Nillable(),
	}
}

// Edges of the RefreshToken.
func (RefreshToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", User.Type).
			Ref("refresh_tokens").
			Unique().
			Required(),
	}
}
