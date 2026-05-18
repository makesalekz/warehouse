package data

import (
	"github.com/shopspring/decimal"

	"github.com/makesalekz/warehouse/ent/enum"
)

type WarehouseDto struct {
	ID          int64
	TenantID    int64
	Name        string
	Type        enum.WarehouseType
	Description string
	Address     string
	IsActive    bool
	StoreID     *int64
}

type StockItemDto struct {
	ID          int64
	TenantID    int64
	WarehouseID int64
	ProductID   int64
	Quantity    decimal.Decimal
	MinQuantity decimal.Decimal
}

type ReceiptItemDto struct {
	ProductID int64
	Quantity  decimal.Decimal
}

type ReceiptDto struct {
	TenantID       int64
	WarehouseID    int64
	CounterpartyID *int64
	CreatedBy      int64
	Items          []ReceiptItemDto
}

type TransferItemDto struct {
	ProductID int64
	Quantity  decimal.Decimal
}

type TransferDto struct {
	TenantID          int64
	SourceWarehouseID int64
	TargetWarehouseID int64
	CreatedBy         int64
	Items             []TransferItemDto
}

type WriteOffItemDto struct {
	ProductID int64
	Quantity  decimal.Decimal
	Reason    string
}

type WriteOffDto struct {
	TenantID    int64
	WarehouseID int64
	CreatedBy   int64
	Items       []WriteOffItemDto
}

type GiftItemDto struct {
	ProductID int64
	Quantity  decimal.Decimal
}

type GiftDto struct {
	TenantID       int64
	WarehouseID    int64
	CounterpartyID int64
	CreatedBy      int64
	Items          []GiftItemDto
}

type StartInventoryDto struct {
	TenantID    int64
	WarehouseID int64
	CreatedBy   int64
}

type SetInventoryItemDto struct {
	TenantID    int64
	InventoryID int64
	ProductID   int64
	ActualQty   decimal.Decimal
}

type CompleteInventoryDto struct {
	TenantID    int64
	InventoryID int64
}
