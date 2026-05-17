package schema

import (
	"gitlab.calendaria.team/services/warehouse/ent/enum"
	"gitlab.calendaria.team/services/warehouse/ent/mixins"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Warehouse struct {
	ent.Schema
}

func (Warehouse) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("tenant_id").Immutable(),
		field.String("name").NotEmpty(),
		field.Enum("type").GoType(enum.WarehouseType("")).Default(enum.Main.Value()),
		field.String("description").Optional().Default(""),
		field.String("address").Optional().Default(""),
		field.Bool("is_active").Default(true),
		field.Int64("store_id").Optional().Nillable(),
	}
}

func (Warehouse) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("stock_items", StockItem.Type),
		edge.To("stock_movements", StockMovement.Type),
		edge.To("inventories", Inventory.Type),
	}
}

func (Warehouse) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
	}
}

func (Warehouse) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.CreateUpdateMixin{},
		mixins.SoftDeleteMixin{},
	}
}
