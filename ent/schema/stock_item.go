package schema

import (
	"github.com/makesalekz/warehouse/ent/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/shopspring/decimal"
)

type StockItem struct {
	ent.Schema
}

func (StockItem) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("tenant_id").Immutable(),
		field.Int64("warehouse_id"),
		field.Int64("product_id"),
		field.Float("quantity").
			GoType(decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric"}).
			Optional(),
		field.Float("min_quantity").
			GoType(decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric"}).
			Optional(),
	}
}

func (StockItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("warehouse", Warehouse.Type).
			Ref("stock_items").
			Field("warehouse_id").
			Required().
			Unique(),
	}
}

func (StockItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "warehouse_id", "product_id").Unique(),
	}
}

func (StockItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.CreateUpdateMixin{},
	}
}
