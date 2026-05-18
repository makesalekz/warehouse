package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
	"github.com/makesalekz/warehouse/ent"
	"github.com/makesalekz/warehouse/internal/data"
	utils_v1 "github.com/makesalekz/utils/api/utils/v1"
)

type WarehouseUsecase struct {
	log                *log.Helper
	warehouseRepo      data.WarehousesRepo
	stockItemsRepo     data.StockItemsRepo
	stockMovementsRepo data.StockMovementsRepo
	inventoriesRepo    data.InventoriesRepo
	publisher          data.LowStockPublisher
}

func NewWarehouseUsecase(logger log.Logger, warehouseRepo data.WarehousesRepo, stockItemsRepo data.StockItemsRepo, stockMovementsRepo data.StockMovementsRepo, inventoriesRepo data.InventoriesRepo, publisher data.LowStockPublisher) *WarehouseUsecase {
	return &WarehouseUsecase{
		log:                log.NewHelper(logger),
		warehouseRepo:      warehouseRepo,
		stockItemsRepo:     stockItemsRepo,
		stockMovementsRepo: stockMovementsRepo,
		inventoriesRepo:    inventoriesRepo,
		publisher:          publisher,
	}
}

func (uc *WarehouseUsecase) Create(ctx context.Context, dto data.WarehouseDto) (*ent.Warehouse, error) {
	return uc.warehouseRepo.Create(ctx, dto)
}

func (uc *WarehouseUsecase) Get(ctx context.Context, tenantID, id int64) (*ent.Warehouse, error) {
	return uc.warehouseRepo.Get(ctx, tenantID, id)
}

func (uc *WarehouseUsecase) List(ctx context.Context, tenantID int64, storeID *int64, paginate *utils_v1.PaginateRequest) ([]*ent.Warehouse, int32, error) {
	items, err := uc.warehouseRepo.List(ctx, tenantID, storeID, paginate)
	if err != nil {
		return nil, 0, err
	}

	total, err := uc.warehouseRepo.Count(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (uc *WarehouseUsecase) Update(ctx context.Context, dto data.WarehouseDto) (*ent.Warehouse, error) {
	return uc.warehouseRepo.Update(ctx, dto)
}

func (uc *WarehouseUsecase) Delete(ctx context.Context, tenantID, id int64) error {
	return uc.warehouseRepo.Delete(ctx, tenantID, id)
}

func (uc *WarehouseUsecase) GetStockItems(ctx context.Context, tenantID, warehouseID int64, paginate *utils_v1.PaginateRequest) ([]*ent.StockItem, int32, error) {
	items, err := uc.stockItemsRepo.ListByWarehouse(ctx, tenantID, warehouseID, paginate)
	if err != nil {
		return nil, 0, err
	}

	total, err := uc.stockItemsRepo.CountByWarehouse(ctx, tenantID, warehouseID)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (uc *WarehouseUsecase) GetStockByProduct(ctx context.Context, tenantID, productID int64) ([]*ent.StockItem, error) {
	return uc.stockItemsRepo.ListByProduct(ctx, tenantID, productID)
}

func (uc *WarehouseUsecase) CreateReceipt(ctx context.Context, dto data.ReceiptDto) ([]*ent.StockMovement, error) {
	return uc.stockMovementsRepo.CreateReceipt(ctx, dto)
}

func (uc *WarehouseUsecase) CreateTransfer(ctx context.Context, dto data.TransferDto) ([]*ent.StockMovement, error) {
	movements, err := uc.stockMovementsRepo.CreateTransfer(ctx, dto)
	if err != nil {
		return nil, err
	}

	// Check low stock on source warehouse for transferred products
	uc.checkLowStock(ctx, dto.TenantID, dto.SourceWarehouseID, transferProductIDs(dto.Items))

	return movements, nil
}

func (uc *WarehouseUsecase) CreateWriteOff(ctx context.Context, dto data.WriteOffDto) ([]*ent.StockMovement, error) {
	movements, err := uc.stockMovementsRepo.CreateWriteOff(ctx, dto)
	if err != nil {
		return nil, err
	}

	// Check low stock for written-off products
	uc.checkLowStock(ctx, dto.TenantID, dto.WarehouseID, writeOffProductIDs(dto.Items))

	return movements, nil
}

func (uc *WarehouseUsecase) CreateGift(ctx context.Context, dto data.GiftDto) ([]*ent.StockMovement, error) {
	return uc.stockMovementsRepo.CreateGift(ctx, dto)
}

func (uc *WarehouseUsecase) ListMovements(ctx context.Context, tenantID, warehouseID int64, movementType *string, paginate *utils_v1.PaginateRequest) ([]*ent.StockMovement, int32, error) {
	items, err := uc.stockMovementsRepo.List(ctx, tenantID, warehouseID, movementType, paginate)
	if err != nil {
		return nil, 0, err
	}

	total, err := uc.stockMovementsRepo.Count(ctx, tenantID, warehouseID, movementType)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (uc *WarehouseUsecase) StartInventory(ctx context.Context, dto data.StartInventoryDto) (*ent.Inventory, []*ent.InventoryItem, error) {
	return uc.inventoriesRepo.Start(ctx, dto)
}

func (uc *WarehouseUsecase) SetInventoryItem(ctx context.Context, dto data.SetInventoryItemDto) (*ent.InventoryItem, error) {
	return uc.inventoriesRepo.SetItem(ctx, dto)
}

func (uc *WarehouseUsecase) CompleteInventory(ctx context.Context, dto data.CompleteInventoryDto) (*ent.Inventory, []*ent.InventoryItem, error) {
	return uc.inventoriesRepo.Complete(ctx, dto)
}

func (uc *WarehouseUsecase) GetInventory(ctx context.Context, tenantID, inventoryID int64) (*ent.Inventory, []*ent.InventoryItem, error) {
	return uc.inventoriesRepo.Get(ctx, tenantID, inventoryID)
}

func (uc *WarehouseUsecase) SetMinQuantity(ctx context.Context, tenantID, warehouseID, productID int64, minQty decimal.Decimal) (*ent.StockItem, error) {
	return uc.stockItemsRepo.SetMinQuantity(ctx, tenantID, warehouseID, productID, minQty)
}

func (uc *WarehouseUsecase) GetLowStockItems(ctx context.Context, tenantID, warehouseID int64) ([]*ent.StockItem, error) {
	return uc.stockItemsRepo.ListLowStock(ctx, tenantID, warehouseID)
}

// checkLowStock checks if any of the given products in the warehouse have
// dropped to or below their min_quantity threshold, and publishes events.
func (uc *WarehouseUsecase) checkLowStock(ctx context.Context, tenantID, warehouseID int64, productIDs []int64) {
	for _, pid := range productIDs {
		item, err := uc.stockItemsRepo.GetByWarehouseAndProduct(ctx, tenantID, warehouseID, pid)
		if err != nil {
			continue
		}
		if item.MinQuantity.IsPositive() && item.Quantity.LessThanOrEqual(item.MinQuantity) {
			uc.publisher.Publish(data.LowStockEvent{
				TenantID:    tenantID,
				WarehouseID: warehouseID,
				ProductID:   pid,
				Quantity:    item.Quantity.String(),
				MinQuantity: item.MinQuantity.String(),
			})
		}
	}
}

func transferProductIDs(items []data.TransferItemDto) []int64 {
	ids := make([]int64, len(items))
	for i, item := range items {
		ids[i] = item.ProductID
	}
	return ids
}

func writeOffProductIDs(items []data.WriteOffItemDto) []int64 {
	ids := make([]int64, len(items))
	for i, item := range items {
		ids[i] = item.ProductID
	}
	return ids
}
