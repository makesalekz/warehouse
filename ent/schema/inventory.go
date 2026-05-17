package schema

import (
	"time"

	"gitlab.calendaria.team/services/warehouse/ent/enum"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Inventory struct {
	ent.Schema
}

func (Inventory) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("tenant_id").Immutable(),
		field.Int64("warehouse_id").Immutable(),
		field.Enum("status").GoType(enum.InventoryStatus("")).Default(enum.InventoryInProgress.Value()),
		field.Int64("created_by").Immutable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("completed_at").Optional().Nillable(),
	}
}

func (Inventory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("warehouse", Warehouse.Type).
			Ref("inventories").
			Field("warehouse_id").
			Required().
			Unique().
			Immutable(),
		edge.To("items", InventoryItem.Type),
	}
}

func (Inventory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "warehouse_id", "status"),
	}
}
