package data

import (
	"context"
	"fmt"
	"time"

	"gitlab.calendaria.team/services/warehouse/ent"
	"gitlab.calendaria.team/services/warehouse/ent/enum"
	"gitlab.calendaria.team/services/warehouse/ent/inventory"
	"gitlab.calendaria.team/services/warehouse/ent/inventoryitem"
	"gitlab.calendaria.team/services/warehouse/ent/stockitem"

	"github.com/shopspring/decimal"
)

type InventoriesRepo interface {
	Start(ctx context.Context, dto StartInventoryDto) (*ent.Inventory, []*ent.InventoryItem, error)
	SetItem(ctx context.Context, dto SetInventoryItemDto) (*ent.InventoryItem, error)
	Complete(ctx context.Context, dto CompleteInventoryDto) (*ent.Inventory, []*ent.InventoryItem, error)
	Get(ctx context.Context, tenantID, inventoryID int64) (*ent.Inventory, []*ent.InventoryItem, error)
}

type inventoriesRepo struct {
	db *ent.Client
}

func NewInventoriesRepo(d *Data) InventoriesRepo {
	return &inventoriesRepo{db: d.db}
}

func (r *inventoriesRepo) Start(ctx context.Context, dto StartInventoryDto) (*ent.Inventory, []*ent.InventoryItem, error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	inv, err := tx.Inventory.Create().
		SetTenantID(dto.TenantID).
		SetWarehouseID(dto.WarehouseID).
		SetStatus(enum.InventoryInProgress).
		SetCreatedBy(dto.CreatedBy).
		Save(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Snapshot current stock items for this warehouse
	stockItems, err := tx.StockItem.Query().
		Where(stockitem.TenantID(dto.TenantID), stockitem.WarehouseID(dto.WarehouseID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	items := make([]*ent.InventoryItem, 0, len(stockItems))
	for _, si := range stockItems {
		item, cErr := tx.InventoryItem.Create().
			SetInventoryID(inv.ID).
			SetProductID(si.ProductID).
			SetExpectedQty(si.Quantity).
			Save(ctx)
		if cErr != nil {
			err = cErr
			return nil, nil, err
		}
		items = append(items, item)
	}

	err = tx.Commit()
	if err != nil {
		return nil, nil, err
	}

	return inv, items, nil
}

func (r *inventoriesRepo) SetItem(ctx context.Context, dto SetInventoryItemDto) (*ent.InventoryItem, error) {
	// Verify inventory exists and belongs to tenant and is not completed
	inv, err := r.db.Inventory.Query().
		Where(inventory.ID(dto.InventoryID), inventory.TenantID(dto.TenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("inventory not found")
		}
		return nil, err
	}

	if inv.Status == enum.InventoryCompleted {
		return nil, fmt.Errorf("inventory already completed")
	}

	// Find existing inventory item for this product
	existingItem, err := r.db.InventoryItem.Query().
		Where(
			inventoryitem.InventoryID(dto.InventoryID),
			inventoryitem.ProductID(dto.ProductID),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			// Product not in snapshot — create with expected_qty=0
			diff := dto.ActualQty.Sub(decimal.Zero)
			return r.db.InventoryItem.Create().
				SetInventoryID(dto.InventoryID).
				SetProductID(dto.ProductID).
				SetExpectedQty(decimal.Zero).
				SetActualQty(dto.ActualQty).
				SetDifference(diff).
				Save(ctx)
		}
		return nil, err
	}

	// Update existing item
	diff := dto.ActualQty.Sub(existingItem.ExpectedQty)
	return r.db.InventoryItem.UpdateOne(existingItem).
		SetActualQty(dto.ActualQty).
		SetDifference(diff).
		Save(ctx)
}

func (r *inventoriesRepo) Complete(ctx context.Context, dto CompleteInventoryDto) (*ent.Inventory, []*ent.InventoryItem, error) {
	inv, err := r.db.Inventory.Query().
		Where(inventory.ID(dto.InventoryID), inventory.TenantID(dto.TenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, fmt.Errorf("inventory not found")
		}
		return nil, nil, err
	}

	if inv.Status == enum.InventoryCompleted {
		return nil, nil, fmt.Errorf("inventory already completed")
	}

	now := time.Now()
	inv, err = r.db.Inventory.UpdateOne(inv).
		SetStatus(enum.InventoryCompleted).
		SetCompletedAt(now).
		Save(ctx)
	if err != nil {
		return nil, nil, err
	}

	items, err := r.db.InventoryItem.Query().
		Where(inventoryitem.InventoryID(inv.ID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return inv, items, nil
}

func (r *inventoriesRepo) Get(ctx context.Context, tenantID, inventoryID int64) (*ent.Inventory, []*ent.InventoryItem, error) {
	inv, err := r.db.Inventory.Query().
		Where(inventory.ID(inventoryID), inventory.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, fmt.Errorf("inventory not found")
		}
		return nil, nil, err
	}

	items, err := r.db.InventoryItem.Query().
		Where(inventoryitem.InventoryID(inv.ID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return inv, items, nil
}
