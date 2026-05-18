package data

import (
	"context"
	"fmt"

	"github.com/makesalekz/warehouse/ent"
	"github.com/makesalekz/warehouse/ent/enum"
	"github.com/makesalekz/warehouse/ent/warehouse"
	utils_v1 "github.com/makesalekz/utils/api/utils/v1"
)

type WarehousesRepo interface {
	Create(ctx context.Context, dto WarehouseDto) (*ent.Warehouse, error)
	Get(ctx context.Context, tenantID, id int64) (*ent.Warehouse, error)
	List(ctx context.Context, tenantID int64, storeID *int64, paginate *utils_v1.PaginateRequest) ([]*ent.Warehouse, error)
	Update(ctx context.Context, dto WarehouseDto) (*ent.Warehouse, error)
	Delete(ctx context.Context, tenantID, id int64) error
	Count(ctx context.Context, tenantID int64) (int32, error)
	FindHallByTenant(ctx context.Context, tenantID int64) (*ent.Warehouse, error)
	FindMainByTenant(ctx context.Context, tenantID int64) (*ent.Warehouse, error)
}

type warehousesRepo struct {
	db *ent.Client
}

func NewWarehousesRepo(d *Data) WarehousesRepo {
	return &warehousesRepo{db: d.db}
}

func (r *warehousesRepo) Create(ctx context.Context, dto WarehouseDto) (*ent.Warehouse, error) {
	create := r.db.Warehouse.Create().
		SetTenantID(dto.TenantID).
		SetName(dto.Name).
		SetType(dto.Type).
		SetDescription(dto.Description).
		SetAddress(dto.Address).
		SetIsActive(dto.IsActive)
	if dto.StoreID != nil {
		create.SetStoreID(*dto.StoreID)
	}
	return create.Save(ctx)
}

func (r *warehousesRepo) Get(ctx context.Context, tenantID, id int64) (*ent.Warehouse, error) {
	return r.db.Warehouse.Query().
		Where(warehouse.ID(id), warehouse.TenantID(tenantID)).
		Only(ctx)
}

func (r *warehousesRepo) List(ctx context.Context, tenantID int64, storeID *int64, paginate *utils_v1.PaginateRequest) ([]*ent.Warehouse, error) {
	query := r.db.Warehouse.Query().Where(warehouse.TenantID(tenantID))

	if storeID != nil {
		query.Where(warehouse.StoreID(*storeID))
	}

	if paginate.GetFromId() != 0 {
		query.Where(warehouse.IDGT(paginate.GetFromId()))
	}

	limit := int(paginate.GetLimit())
	if limit == 0 {
		limit = 100
	}

	return query.Limit(limit).Order(ent.Asc(warehouse.FieldID)).All(ctx)
}

func (r *warehousesRepo) Update(ctx context.Context, dto WarehouseDto) (*ent.Warehouse, error) {
	update := r.db.Warehouse.UpdateOneID(dto.ID).
		Where(warehouse.TenantID(dto.TenantID)).
		SetName(dto.Name).
		SetType(dto.Type).
		SetDescription(dto.Description).
		SetAddress(dto.Address).
		SetIsActive(dto.IsActive)
	if dto.StoreID != nil {
		update.SetStoreID(*dto.StoreID)
	} else {
		update.ClearStoreID()
	}
	return update.Save(ctx)
}

func (r *warehousesRepo) Delete(ctx context.Context, tenantID, id int64) error {
	_, err := r.db.Warehouse.Delete().
		Where(warehouse.ID(id), warehouse.TenantID(tenantID)).
		Exec(ctx)
	return err
}

func (r *warehousesRepo) Count(ctx context.Context, tenantID int64) (int32, error) {
	count, err := r.db.Warehouse.Query().
		Where(warehouse.TenantID(tenantID)).
		Count(ctx)
	return int32(count), err
}

// FindHallByTenant returns the first active HALL warehouse for the given tenant.
// MVP: assumes one HALL warehouse per tenant.
func (r *warehousesRepo) FindHallByTenant(ctx context.Context, tenantID int64) (*ent.Warehouse, error) {
	w, err := r.db.Warehouse.Query().
		Where(
			warehouse.TenantID(tenantID),
			warehouse.TypeEQ(enum.Hall),
			warehouse.IsActive(true),
		).
		Order(ent.Asc(warehouse.FieldID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("no active HALL warehouse found for tenant %d: %w", tenantID, err)
	}
	return w, nil
}

// FindMainByTenant returns the first active MAIN warehouse for the given tenant.
// MVP: assumes one MAIN warehouse per tenant.
func (r *warehousesRepo) FindMainByTenant(ctx context.Context, tenantID int64) (*ent.Warehouse, error) {
	w, err := r.db.Warehouse.Query().
		Where(
			warehouse.TenantID(tenantID),
			warehouse.TypeEQ(enum.Main),
			warehouse.IsActive(true),
		).
		Order(ent.Asc(warehouse.FieldID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("no active MAIN warehouse found for tenant %d: %w", tenantID, err)
	}
	return w, nil
}
