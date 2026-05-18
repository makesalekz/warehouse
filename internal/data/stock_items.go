package data

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/shopspring/decimal"
	"github.com/makesalekz/warehouse/ent"
	"github.com/makesalekz/warehouse/ent/stockitem"
	utils_v1 "github.com/makesalekz/utils/api/utils/v1"
)

type StockItemsRepo interface {
	Upsert(ctx context.Context, dto StockItemDto) (*ent.StockItem, error)
	ListByWarehouse(ctx context.Context, tenantID, warehouseID int64, paginate *utils_v1.PaginateRequest) ([]*ent.StockItem, error)
	CountByWarehouse(ctx context.Context, tenantID, warehouseID int64) (int32, error)
	ListByProduct(ctx context.Context, tenantID, productID int64) ([]*ent.StockItem, error)
	SetMinQuantity(ctx context.Context, tenantID, warehouseID, productID int64, minQty decimal.Decimal) (*ent.StockItem, error)
	ListLowStock(ctx context.Context, tenantID, warehouseID int64) ([]*ent.StockItem, error)
	GetByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID int64) (*ent.StockItem, error)
	DecrementQuantity(ctx context.Context, tenantID, warehouseID, productID int64, qty decimal.Decimal) error
	IncrementQuantity(ctx context.Context, tenantID, warehouseID, productID int64, qty decimal.Decimal) error
}

type stockItemsRepo struct {
	db *ent.Client
}

func NewStockItemsRepo(d *Data) StockItemsRepo {
	return &stockItemsRepo{db: d.db}
}

func (r *stockItemsRepo) Upsert(ctx context.Context, dto StockItemDto) (*ent.StockItem, error) {
	id, err := r.db.StockItem.Create().
		SetTenantID(dto.TenantID).
		SetWarehouseID(dto.WarehouseID).
		SetProductID(dto.ProductID).
		SetQuantity(dto.Quantity).
		SetMinQuantity(dto.MinQuantity).
		OnConflictColumns(stockitem.FieldTenantID, stockitem.FieldWarehouseID, stockitem.FieldProductID).
		SetQuantity(dto.Quantity).
		SetMinQuantity(dto.MinQuantity).
		ID(ctx)
	if err != nil {
		return nil, err
	}
	return r.db.StockItem.Get(ctx, id)
}

func (r *stockItemsRepo) ListByWarehouse(ctx context.Context, tenantID, warehouseID int64, paginate *utils_v1.PaginateRequest) ([]*ent.StockItem, error) {
	query := r.db.StockItem.Query().
		Where(stockitem.TenantID(tenantID), stockitem.WarehouseID(warehouseID))

	if paginate.GetFromId() != 0 {
		query.Where(stockitem.IDGT(paginate.GetFromId()))
	}

	limit := int(paginate.GetLimit())
	if limit == 0 {
		limit = 100
	}

	return query.Limit(limit).Order(ent.Asc(stockitem.FieldID)).All(ctx)
}

func (r *stockItemsRepo) CountByWarehouse(ctx context.Context, tenantID, warehouseID int64) (int32, error) {
	count, err := r.db.StockItem.Query().
		Where(stockitem.TenantID(tenantID), stockitem.WarehouseID(warehouseID)).
		Count(ctx)
	return int32(count), err
}

func (r *stockItemsRepo) ListByProduct(ctx context.Context, tenantID, productID int64) ([]*ent.StockItem, error) {
	return r.db.StockItem.Query().
		Where(stockitem.TenantID(tenantID), stockitem.ProductID(productID)).
		All(ctx)
}

func (r *stockItemsRepo) SetMinQuantity(ctx context.Context, tenantID, warehouseID, productID int64, minQty decimal.Decimal) (*ent.StockItem, error) {
	id, err := r.db.StockItem.Create().
		SetTenantID(tenantID).
		SetWarehouseID(warehouseID).
		SetProductID(productID).
		SetQuantity(decimal.Zero).
		SetMinQuantity(minQty).
		OnConflictColumns(stockitem.FieldTenantID, stockitem.FieldWarehouseID, stockitem.FieldProductID).
		SetMinQuantity(minQty).
		ID(ctx)
	if err != nil {
		return nil, err
	}
	return r.db.StockItem.Get(ctx, id)
}

func (r *stockItemsRepo) ListLowStock(ctx context.Context, tenantID, warehouseID int64) ([]*ent.StockItem, error) {
	return r.db.StockItem.Query().
		Where(
			stockitem.TenantID(tenantID),
			stockitem.WarehouseID(warehouseID),
			stockitem.MinQuantityGT(decimal.Zero),
			func(s *sql.Selector) {
				s.Where(sql.ColumnsLTE(stockitem.FieldQuantity, stockitem.FieldMinQuantity))
			},
		).
		All(ctx)
}

func (r *stockItemsRepo) GetByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID int64) (*ent.StockItem, error) {
	return r.db.StockItem.Query().
		Where(
			stockitem.TenantID(tenantID),
			stockitem.WarehouseID(warehouseID),
			stockitem.ProductID(productID),
		).
		Only(ctx)
}

func (r *stockItemsRepo) DecrementQuantity(ctx context.Context, tenantID, warehouseID, productID int64, qty decimal.Decimal) error {
	// Upsert: if stock item doesn't exist, create with negative qty; if exists, subtract.
	return r.db.StockItem.Create().
		SetTenantID(tenantID).
		SetWarehouseID(warehouseID).
		SetProductID(productID).
		SetQuantity(qty.Neg()).
		OnConflictColumns(stockitem.FieldTenantID, stockitem.FieldWarehouseID, stockitem.FieldProductID).
		AddQuantity(qty.Neg()).
		Exec(ctx)
}

func (r *stockItemsRepo) IncrementQuantity(ctx context.Context, tenantID, warehouseID, productID int64, qty decimal.Decimal) error {
	return r.db.StockItem.Create().
		SetTenantID(tenantID).
		SetWarehouseID(warehouseID).
		SetProductID(productID).
		SetQuantity(qty).
		OnConflictColumns(stockitem.FieldTenantID, stockitem.FieldWarehouseID, stockitem.FieldProductID).
		AddQuantity(qty).
		Exec(ctx)
}
