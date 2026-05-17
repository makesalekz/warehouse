package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/shopspring/decimal"
)

type InventoryItem struct {
	ent.Schema
}

func (InventoryItem) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("inventory_id").Immutable(),
		field.Int64("product_id").Immutable(),
		field.Float("expected_qty").
			GoType(decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric"}).
			Immutable(),
		field.Float("actual_qty").
			GoType(decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric"}).
			Optional().
			Nillable(),
		field.Float("difference").
			GoType(decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric"}).
			Optional().
			Nillable(),
	}
}

func (InventoryItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("inventory", Inventory.Type).
			Ref("items").
			Field("inventory_id").
			Required().
			Unique().
			Immutable(),
	}
}

func (InventoryItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("inventory_id", "product_id").Unique(),
	}
}
