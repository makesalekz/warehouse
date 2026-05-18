package schema

import (
	"time"

	"github.com/makesalekz/warehouse/ent/enum"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/shopspring/decimal"
)

type StockMovement struct {
	ent.Schema
}

func (StockMovement) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("tenant_id").Immutable(),
		field.Int64("warehouse_id").Immutable(),
		field.Enum("type").GoType(enum.MovementType("")).Immutable(),
		field.Int64("product_id").Immutable(),
		field.Float("quantity").
			GoType(decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric"}).
			Immutable(),
		field.Int64("counterparty_id").Optional().Nillable().Immutable(),
		field.String("reason").Optional().Nillable().Immutable(),
		field.Int64("created_by").Immutable(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (StockMovement) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("warehouse", Warehouse.Type).
			Ref("stock_movements").
			Field("warehouse_id").
			Required().
			Unique().
			Immutable(),
	}
}

func (StockMovement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "warehouse_id", "created_at"),
		index.Fields("tenant_id", "warehouse_id", "type"),
	}
}
