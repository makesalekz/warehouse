package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	v1 "gitlab.calendaria.team/services/warehouse/api/warehouse/v1"
	"gitlab.calendaria.team/services/warehouse/ent"
	"gitlab.calendaria.team/services/warehouse/ent/enum"
	"gitlab.calendaria.team/services/warehouse/internal/biz"
	"gitlab.calendaria.team/services/warehouse/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	"gitlab.calendaria.team/services/utils/v2/auth"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errNotFound = errors.New("not found")

// --- Mock WarehousesRepo ---

type mockWarehousesRepo struct {
	warehouses map[int64]*ent.Warehouse
	nextID     int64
}

func newMockWarehousesRepo() *mockWarehousesRepo {
	return &mockWarehousesRepo{
		warehouses: make(map[int64]*ent.Warehouse),
		nextID:     1,
	}
}

func (m *mockWarehousesRepo) Create(_ context.Context, dto data.WarehouseDto) (*ent.Warehouse, error) {
	w := &ent.Warehouse{
		ID:          m.nextID,
		TenantID:    dto.TenantID,
		Name:        dto.Name,
		Type:        dto.Type,
		Description: dto.Description,
		Address:     dto.Address,
		IsActive:    dto.IsActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.warehouses[m.nextID] = w
	m.nextID++
	return w, nil
}

func (m *mockWarehousesRepo) Get(_ context.Context, tenantID, id int64) (*ent.Warehouse, error) {
	w, ok := m.warehouses[id]
	if !ok || w.TenantID != tenantID {
		return nil, errNotFound
	}
	return w, nil
}

func (m *mockWarehousesRepo) List(_ context.Context, tenantID int64, storeID *int64, paginate *utils_v1.PaginateRequest) ([]*ent.Warehouse, error) {
	var result []*ent.Warehouse
	for _, w := range m.warehouses {
		if w.TenantID == tenantID {
			if storeID != nil && (w.StoreID == nil || *w.StoreID != *storeID) {
				continue
			}
			if paginate.GetFromId() != 0 && w.ID <= paginate.GetFromId() {
				continue
			}
			result = append(result, w)
		}
	}
	limit := int(paginate.GetLimit())
	if limit == 0 {
		limit = 100
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockWarehousesRepo) Update(_ context.Context, dto data.WarehouseDto) (*ent.Warehouse, error) {
	w, ok := m.warehouses[dto.ID]
	if !ok || w.TenantID != dto.TenantID {
		return nil, errNotFound
	}
	w.Name = dto.Name
	w.Type = dto.Type
	w.Description = dto.Description
	w.Address = dto.Address
	w.IsActive = dto.IsActive
	w.UpdatedAt = time.Now()
	return w, nil
}

func (m *mockWarehousesRepo) Delete(_ context.Context, tenantID, id int64) error {
	w, ok := m.warehouses[id]
	if !ok || w.TenantID != tenantID {
		return nil
	}
	delete(m.warehouses, id)
	return nil
}

func (m *mockWarehousesRepo) Count(_ context.Context, tenantID int64) (int32, error) {
	var count int32
	for _, w := range m.warehouses {
		if w.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

// --- Mock StockItemsRepo ---

type mockStockItemsRepo struct {
	items  map[int64]*ent.StockItem
	nextID int64
}

func newMockStockItemsRepo() *mockStockItemsRepo {
	return &mockStockItemsRepo{
		items:  make(map[int64]*ent.StockItem),
		nextID: 1,
	}
}

func (m *mockStockItemsRepo) Upsert(_ context.Context, dto data.StockItemDto) (*ent.StockItem, error) {
	// Check if exists (simulate unique constraint)
	for _, item := range m.items {
		if item.TenantID == dto.TenantID && item.WarehouseID == dto.WarehouseID && item.ProductID == dto.ProductID {
			item.Quantity = dto.Quantity
			item.MinQuantity = dto.MinQuantity
			item.UpdatedAt = time.Now()
			return item, nil
		}
	}
	s := &ent.StockItem{
		ID:          m.nextID,
		TenantID:    dto.TenantID,
		WarehouseID: dto.WarehouseID,
		ProductID:   dto.ProductID,
		Quantity:    dto.Quantity,
		MinQuantity: dto.MinQuantity,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.items[m.nextID] = s
	m.nextID++
	return s, nil
}

func (m *mockStockItemsRepo) ListByWarehouse(_ context.Context, tenantID, warehouseID int64, paginate *utils_v1.PaginateRequest) ([]*ent.StockItem, error) {
	var result []*ent.StockItem
	for _, item := range m.items {
		if item.TenantID == tenantID && item.WarehouseID == warehouseID {
			if paginate.GetFromId() != 0 && item.ID <= paginate.GetFromId() {
				continue
			}
			result = append(result, item)
		}
	}
	limit := int(paginate.GetLimit())
	if limit == 0 {
		limit = 100
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockStockItemsRepo) CountByWarehouse(_ context.Context, tenantID, warehouseID int64) (int32, error) {
	var count int32
	for _, item := range m.items {
		if item.TenantID == tenantID && item.WarehouseID == warehouseID {
			count++
		}
	}
	return count, nil
}

func (m *mockStockItemsRepo) ListByProduct(_ context.Context, tenantID, productID int64) ([]*ent.StockItem, error) {
	var result []*ent.StockItem
	for _, item := range m.items {
		if item.TenantID == tenantID && item.ProductID == productID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (m *mockStockItemsRepo) SetMinQuantity(_ context.Context, tenantID, warehouseID, productID int64, minQty decimal.Decimal) (*ent.StockItem, error) {
	for _, item := range m.items {
		if item.TenantID == tenantID && item.WarehouseID == warehouseID && item.ProductID == productID {
			item.MinQuantity = minQty
			item.UpdatedAt = time.Now()
			return item, nil
		}
	}
	// Upsert: create with qty=0 if not exists
	s := &ent.StockItem{
		ID:          m.nextID,
		TenantID:    tenantID,
		WarehouseID: warehouseID,
		ProductID:   productID,
		Quantity:    decimal.Zero,
		MinQuantity: minQty,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.items[m.nextID] = s
	m.nextID++
	return s, nil
}

func (m *mockStockItemsRepo) ListLowStock(_ context.Context, tenantID, warehouseID int64) ([]*ent.StockItem, error) {
	var result []*ent.StockItem
	for _, item := range m.items {
		if item.TenantID == tenantID && item.WarehouseID == warehouseID && item.MinQuantity.IsPositive() && item.Quantity.LessThanOrEqual(item.MinQuantity) {
			result = append(result, item)
		}
	}
	return result, nil
}

func (m *mockStockItemsRepo) GetByWarehouseAndProduct(_ context.Context, tenantID, warehouseID, productID int64) (*ent.StockItem, error) {
	for _, item := range m.items {
		if item.TenantID == tenantID && item.WarehouseID == warehouseID && item.ProductID == productID {
			return item, nil
		}
	}
	return nil, errNotFound
}

// --- Mock StockMovementsRepo ---

type mockStockMovementsRepo struct {
	movements      map[int64]*ent.StockMovement
	nextID         int64
	stockItems     *mockStockItemsRepo
	warehousesRepo *mockWarehousesRepo
}

func newMockStockMovementsRepo(stockItems *mockStockItemsRepo, warehousesRepo *mockWarehousesRepo) *mockStockMovementsRepo {
	return &mockStockMovementsRepo{
		movements:      make(map[int64]*ent.StockMovement),
		nextID:         1,
		stockItems:     stockItems,
		warehousesRepo: warehousesRepo,
	}
}

func (m *mockStockMovementsRepo) CreateTransfer(_ context.Context, dto data.TransferDto) ([]*ent.StockMovement, error) {
	// Verify both warehouses exist for the tenant
	sourceFound := false
	targetFound := false
	for _, w := range m.warehousesRepo.warehouses {
		if w.TenantID == dto.TenantID && w.ID == dto.SourceWarehouseID {
			sourceFound = true
		}
		if w.TenantID == dto.TenantID && w.ID == dto.TargetWarehouseID {
			targetFound = true
		}
	}
	if !sourceFound {
		return nil, fmt.Errorf("source warehouse not found")
	}
	if !targetFound {
		return nil, fmt.Errorf("target warehouse not found")
	}

	movements := make([]*ent.StockMovement, 0, len(dto.Items)*2)
	for _, item := range dto.Items {
		// Check source stock
		var sourceStock *ent.StockItem
		for _, si := range m.stockItems.items {
			if si.TenantID == dto.TenantID && si.WarehouseID == dto.SourceWarehouseID && si.ProductID == item.ProductID {
				sourceStock = si
				break
			}
		}
		if sourceStock == nil || sourceStock.Quantity.LessThan(item.Quantity) {
			return nil, fmt.Errorf("insufficient stock for product %d", item.ProductID)
		}
	}

	// All validations passed — apply changes
	for _, item := range dto.Items {
		// Decrease source
		for _, si := range m.stockItems.items {
			if si.TenantID == dto.TenantID && si.WarehouseID == dto.SourceWarehouseID && si.ProductID == item.ProductID {
				si.Quantity = si.Quantity.Sub(item.Quantity)
				si.UpdatedAt = time.Now()
				break
			}
		}

		// Outgoing movement
		outMov := &ent.StockMovement{
			ID:          m.nextID,
			TenantID:    dto.TenantID,
			WarehouseID: dto.SourceWarehouseID,
			Type:        enum.Transfer,
			ProductID:   item.ProductID,
			Quantity:    item.Quantity.Neg(),
			CreatedBy:   dto.CreatedBy,
			CreatedAt:   time.Now(),
		}
		m.movements[m.nextID] = outMov
		m.nextID++
		movements = append(movements, outMov)

		// Increase target (upsert)
		found := false
		for _, si := range m.stockItems.items {
			if si.TenantID == dto.TenantID && si.WarehouseID == dto.TargetWarehouseID && si.ProductID == item.ProductID {
				si.Quantity = si.Quantity.Add(item.Quantity)
				si.UpdatedAt = time.Now()
				found = true
				break
			}
		}
		if !found {
			si := &ent.StockItem{
				ID:          m.stockItems.nextID,
				TenantID:    dto.TenantID,
				WarehouseID: dto.TargetWarehouseID,
				ProductID:   item.ProductID,
				Quantity:    item.Quantity,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			m.stockItems.items[m.stockItems.nextID] = si
			m.stockItems.nextID++
		}

		// Incoming movement
		inMov := &ent.StockMovement{
			ID:          m.nextID,
			TenantID:    dto.TenantID,
			WarehouseID: dto.TargetWarehouseID,
			Type:        enum.Transfer,
			ProductID:   item.ProductID,
			Quantity:    item.Quantity,
			CreatedBy:   dto.CreatedBy,
			CreatedAt:   time.Now(),
		}
		m.movements[m.nextID] = inMov
		m.nextID++
		movements = append(movements, inMov)
	}

	return movements, nil
}

func (m *mockStockMovementsRepo) CreateReceipt(_ context.Context, dto data.ReceiptDto) ([]*ent.StockMovement, error) {
	movements := make([]*ent.StockMovement, 0, len(dto.Items))
	for _, item := range dto.Items {
		mov := &ent.StockMovement{
			ID:          m.nextID,
			TenantID:    dto.TenantID,
			WarehouseID: dto.WarehouseID,
			Type:        enum.Receipt,
			ProductID:   item.ProductID,
			Quantity:    item.Quantity,
			CreatedBy:   dto.CreatedBy,
			CreatedAt:   time.Now(),
		}
		if dto.CounterpartyID != nil {
			mov.CounterpartyID = dto.CounterpartyID
		}
		m.movements[m.nextID] = mov
		m.nextID++
		movements = append(movements, mov)

		// Simulate stock item upsert with increment
		found := false
		for _, si := range m.stockItems.items {
			if si.TenantID == dto.TenantID && si.WarehouseID == dto.WarehouseID && si.ProductID == item.ProductID {
				si.Quantity = si.Quantity.Add(item.Quantity)
				si.UpdatedAt = time.Now()
				found = true
				break
			}
		}
		if !found {
			si := &ent.StockItem{
				ID:          m.stockItems.nextID,
				TenantID:    dto.TenantID,
				WarehouseID: dto.WarehouseID,
				ProductID:   item.ProductID,
				Quantity:    item.Quantity,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			m.stockItems.items[m.stockItems.nextID] = si
			m.stockItems.nextID++
		}
	}
	return movements, nil
}

func (m *mockStockMovementsRepo) CreateWriteOff(_ context.Context, dto data.WriteOffDto) ([]*ent.StockMovement, error) {
	// Validate all items first (atomic)
	for _, item := range dto.Items {
		var sourceStock *ent.StockItem
		for _, si := range m.stockItems.items {
			if si.TenantID == dto.TenantID && si.WarehouseID == dto.WarehouseID && si.ProductID == item.ProductID {
				sourceStock = si
				break
			}
		}
		if sourceStock == nil || sourceStock.Quantity.LessThan(item.Quantity) {
			return nil, fmt.Errorf("insufficient stock for product %d", item.ProductID)
		}
	}

	// All validations passed — apply changes
	movements := make([]*ent.StockMovement, 0, len(dto.Items))
	for _, item := range dto.Items {
		// Decrease stock
		for _, si := range m.stockItems.items {
			if si.TenantID == dto.TenantID && si.WarehouseID == dto.WarehouseID && si.ProductID == item.ProductID {
				si.Quantity = si.Quantity.Sub(item.Quantity)
				si.UpdatedAt = time.Now()
				break
			}
		}

		reason := item.Reason
		mov := &ent.StockMovement{
			ID:          m.nextID,
			TenantID:    dto.TenantID,
			WarehouseID: dto.WarehouseID,
			Type:        enum.WriteOff,
			ProductID:   item.ProductID,
			Quantity:    item.Quantity.Neg(),
			Reason:      &reason,
			CreatedBy:   dto.CreatedBy,
			CreatedAt:   time.Now(),
		}
		m.movements[m.nextID] = mov
		m.nextID++
		movements = append(movements, mov)
	}

	return movements, nil
}

func (m *mockStockMovementsRepo) CreateGift(_ context.Context, dto data.GiftDto) ([]*ent.StockMovement, error) {
	movements := make([]*ent.StockMovement, 0, len(dto.Items))
	for _, item := range dto.Items {
		counterpartyID := dto.CounterpartyID
		mov := &ent.StockMovement{
			ID:             m.nextID,
			TenantID:       dto.TenantID,
			WarehouseID:    dto.WarehouseID,
			Type:           enum.Gift,
			ProductID:      item.ProductID,
			Quantity:       item.Quantity,
			CounterpartyID: &counterpartyID,
			CreatedBy:      dto.CreatedBy,
			CreatedAt:      time.Now(),
		}
		m.movements[m.nextID] = mov
		m.nextID++
		movements = append(movements, mov)

		// Upsert stock item
		found := false
		for _, si := range m.stockItems.items {
			if si.TenantID == dto.TenantID && si.WarehouseID == dto.WarehouseID && si.ProductID == item.ProductID {
				si.Quantity = si.Quantity.Add(item.Quantity)
				si.UpdatedAt = time.Now()
				found = true
				break
			}
		}
		if !found {
			si := &ent.StockItem{
				ID:          m.stockItems.nextID,
				TenantID:    dto.TenantID,
				WarehouseID: dto.WarehouseID,
				ProductID:   item.ProductID,
				Quantity:    item.Quantity,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			m.stockItems.items[m.stockItems.nextID] = si
			m.stockItems.nextID++
		}
	}
	return movements, nil
}

func (m *mockStockMovementsRepo) List(_ context.Context, tenantID, warehouseID int64, movementType *string, paginate *utils_v1.PaginateRequest) ([]*ent.StockMovement, error) {
	var result []*ent.StockMovement
	for _, mov := range m.movements {
		if mov.TenantID != tenantID || mov.WarehouseID != warehouseID {
			continue
		}
		if movementType != nil && *movementType != "" && string(mov.Type) != *movementType {
			continue
		}
		if paginate.GetFromId() != 0 && mov.ID <= paginate.GetFromId() {
			continue
		}
		result = append(result, mov)
	}
	limit := int(paginate.GetLimit())
	if limit == 0 {
		limit = 100
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockStockMovementsRepo) Count(_ context.Context, tenantID, warehouseID int64, movementType *string) (int32, error) {
	var count int32
	for _, mov := range m.movements {
		if mov.TenantID != tenantID || mov.WarehouseID != warehouseID {
			continue
		}
		if movementType != nil && *movementType != "" && string(mov.Type) != *movementType {
			continue
		}
		count++
	}
	return count, nil
}

// --- Mock InventoriesRepo ---

type mockInventoriesRepo struct {
	inventories map[int64]*ent.Inventory
	items       map[int64][]*ent.InventoryItem
	nextID      int64
	nextItemID  int64
	stockItems  *mockStockItemsRepo
}

func newMockInventoriesRepo(stockItems *mockStockItemsRepo) *mockInventoriesRepo {
	return &mockInventoriesRepo{
		inventories: make(map[int64]*ent.Inventory),
		items:       make(map[int64][]*ent.InventoryItem),
		nextID:      1,
		nextItemID:  1,
		stockItems:  stockItems,
	}
}

func (m *mockInventoriesRepo) Start(_ context.Context, dto data.StartInventoryDto) (*ent.Inventory, []*ent.InventoryItem, error) {
	inv := &ent.Inventory{
		ID:          m.nextID,
		TenantID:    dto.TenantID,
		WarehouseID: dto.WarehouseID,
		Status:      enum.InventoryInProgress,
		CreatedBy:   dto.CreatedBy,
		CreatedAt:   time.Now(),
	}
	m.inventories[m.nextID] = inv
	m.nextID++

	var invItems []*ent.InventoryItem
	for _, si := range m.stockItems.items {
		if si.TenantID == dto.TenantID && si.WarehouseID == dto.WarehouseID {
			item := &ent.InventoryItem{
				ID:          m.nextItemID,
				InventoryID: inv.ID,
				ProductID:   si.ProductID,
				ExpectedQty: si.Quantity,
			}
			m.nextItemID++
			invItems = append(invItems, item)
		}
	}
	m.items[inv.ID] = invItems

	return inv, invItems, nil
}

func (m *mockInventoriesRepo) SetItem(_ context.Context, dto data.SetInventoryItemDto) (*ent.InventoryItem, error) {
	inv, ok := m.inventories[dto.InventoryID]
	if !ok || inv.TenantID != dto.TenantID {
		return nil, fmt.Errorf("inventory not found")
	}
	if inv.Status == enum.InventoryCompleted {
		return nil, fmt.Errorf("inventory already completed")
	}

	for _, item := range m.items[dto.InventoryID] {
		if item.ProductID == dto.ProductID {
			item.ActualQty = &dto.ActualQty
			diff := dto.ActualQty.Sub(item.ExpectedQty)
			item.Difference = &diff
			return item, nil
		}
	}

	// Not in snapshot — create with expected=0
	diff := dto.ActualQty.Sub(decimal.Zero)
	newItem := &ent.InventoryItem{
		ID:          m.nextItemID,
		InventoryID: dto.InventoryID,
		ProductID:   dto.ProductID,
		ExpectedQty: decimal.Zero,
		ActualQty:   &dto.ActualQty,
		Difference:  &diff,
	}
	m.nextItemID++
	m.items[dto.InventoryID] = append(m.items[dto.InventoryID], newItem)
	return newItem, nil
}

func (m *mockInventoriesRepo) Complete(_ context.Context, dto data.CompleteInventoryDto) (*ent.Inventory, []*ent.InventoryItem, error) {
	inv, ok := m.inventories[dto.InventoryID]
	if !ok || inv.TenantID != dto.TenantID {
		return nil, nil, fmt.Errorf("inventory not found")
	}
	if inv.Status == enum.InventoryCompleted {
		return nil, nil, fmt.Errorf("inventory already completed")
	}

	inv.Status = enum.InventoryCompleted
	now := time.Now()
	inv.CompletedAt = &now

	return inv, m.items[inv.ID], nil
}

func (m *mockInventoriesRepo) Get(_ context.Context, tenantID, inventoryID int64) (*ent.Inventory, []*ent.InventoryItem, error) {
	inv, ok := m.inventories[inventoryID]
	if !ok || inv.TenantID != tenantID {
		return nil, nil, fmt.Errorf("inventory not found")
	}
	return inv, m.items[inv.ID], nil
}

// --- Mock LowStockPublisher ---

type mockLowStockPublisher struct {
	events []data.LowStockEvent
}

func (m *mockLowStockPublisher) Publish(event data.LowStockEvent) {
	m.events = append(m.events, event)
}

// --- Test setup ---

func setupService() (*WarehouseService, *mockWarehousesRepo, *mockStockItemsRepo) {
	warehousesRepo := newMockWarehousesRepo()
	stockItemsRepo := newMockStockItemsRepo()
	movementsRepo := newMockStockMovementsRepo(stockItemsRepo, warehousesRepo)
	inventoriesRepo := newMockInventoriesRepo(stockItemsRepo)
	publisher := &mockLowStockPublisher{}
	uc := biz.NewWarehouseUsecase(log.DefaultLogger, warehousesRepo, stockItemsRepo, movementsRepo, inventoriesRepo, publisher)
	svc := NewWarehouseService(uc)
	return svc, warehousesRepo, stockItemsRepo
}

func setupServiceWithMovements() (*WarehouseService, *mockWarehousesRepo, *mockStockItemsRepo, *mockStockMovementsRepo) {
	warehousesRepo := newMockWarehousesRepo()
	stockItemsRepo := newMockStockItemsRepo()
	movementsRepo := newMockStockMovementsRepo(stockItemsRepo, warehousesRepo)
	inventoriesRepo := newMockInventoriesRepo(stockItemsRepo)
	publisher := &mockLowStockPublisher{}
	uc := biz.NewWarehouseUsecase(log.DefaultLogger, warehousesRepo, stockItemsRepo, movementsRepo, inventoriesRepo, publisher)
	svc := NewWarehouseService(uc)
	return svc, warehousesRepo, stockItemsRepo, movementsRepo
}

func setupServiceWithPublisher() (*WarehouseService, *mockWarehousesRepo, *mockStockItemsRepo, *mockStockMovementsRepo, *mockLowStockPublisher) {
	warehousesRepo := newMockWarehousesRepo()
	stockItemsRepo := newMockStockItemsRepo()
	movementsRepo := newMockStockMovementsRepo(stockItemsRepo, warehousesRepo)
	inventoriesRepo := newMockInventoriesRepo(stockItemsRepo)
	publisher := &mockLowStockPublisher{}
	uc := biz.NewWarehouseUsecase(log.DefaultLogger, warehousesRepo, stockItemsRepo, movementsRepo, inventoriesRepo, publisher)
	svc := NewWarehouseService(uc)
	return svc, warehousesRepo, stockItemsRepo, movementsRepo, publisher
}

func setupServiceWithInventory() (*WarehouseService, *mockWarehousesRepo, *mockStockItemsRepo, *mockInventoriesRepo) {
	warehousesRepo := newMockWarehousesRepo()
	stockItemsRepo := newMockStockItemsRepo()
	movementsRepo := newMockStockMovementsRepo(stockItemsRepo, warehousesRepo)
	inventoriesRepo := newMockInventoriesRepo(stockItemsRepo)
	publisher := &mockLowStockPublisher{}
	uc := biz.NewWarehouseUsecase(log.DefaultLogger, warehousesRepo, stockItemsRepo, movementsRepo, inventoriesRepo, publisher)
	svc := NewWarehouseService(uc)
	return svc, warehousesRepo, stockItemsRepo, inventoriesRepo
}

func ctxWithTenant(tenantID int64) context.Context {
	return auth.NewTenantContext(context.Background(), tenantID)
}

func ctxWithTenantAndActor(tenantID, actorID int64) context.Context {
	ctx := auth.NewTenantContext(context.Background(), tenantID)
	return auth.NewActorContext(ctx, actorID)
}

// --- Warehouse CRUD Tests ---

func TestCreateWarehouse(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	resp, err := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{
		Name:        "Основной склад",
		Type:        "MAIN",
		Description: "Главный склад магазина",
		Address:     "ул. Абая 10",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Warehouse)
	assert.Equal(t, "Основной склад", resp.Warehouse.Name)
	assert.Equal(t, "MAIN", resp.Warehouse.Type)
	assert.Equal(t, "Главный склад магазина", resp.Warehouse.Description)
	assert.Equal(t, "ул. Абая 10", resp.Warehouse.Address)
	assert.True(t, resp.Warehouse.IsActive)
	assert.Equal(t, int64(1), resp.Warehouse.TenantId)
	assert.NotEmpty(t, resp.Warehouse.CreatedAt)
}

func TestCreateWarehouse_NoTenant(t *testing.T) {
	svc, _, _ := setupService()
	ctx := context.Background()

	_, err := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{
		Name: "Склад",
		Type: "MAIN",
	})
	require.Error(t, err)
}

func TestCreateWarehouse_InvalidType(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	resp, err := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{
		Name: "��клад",
		Type: "INVALID",
	})
	require.NoError(t, err)
	// Defaults to MAIN
	assert.Equal(t, "MAIN", resp.Warehouse.Type)
}

func TestCreateWarehouse_AllTypes(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	types := []string{"MAIN", "HALL", "TRANSIT"}
	for _, wt := range types {
		resp, err := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{
			Name: "Склад " + wt,
			Type: wt,
		})
		require.NoError(t, err)
		assert.Equal(t, wt, resp.Warehouse.Type)
	}
}

func TestGetWarehouse(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	created, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{
		Name: "Основной",
		Type: "MAIN",
	})

	resp, err := svc.GetWarehouse(ctx, &v1.GetWarehouseRequest{Id: created.Warehouse.Id})
	require.NoError(t, err)
	assert.Equal(t, "Основной", resp.Warehouse.Name)
}

func TestGetWarehouse_NotFound(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	_, err := svc.GetWarehouse(ctx, &v1.GetWarehouseRequest{Id: 999})
	require.Error(t, err)
}

func TestGetWarehouse_TenantIsolation(t *testing.T) {
	svc, _, _ := setupService()
	ctx1 := ctxWithTenant(1)
	ctx2 := ctxWithTenant(2)

	created, _ := svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{
		Name: "Склад",
		Type: "MAIN",
	})

	_, err := svc.GetWarehouse(ctx2, &v1.GetWarehouseRequest{Id: created.Warehouse.Id})
	require.Error(t, err)
}

func TestListWarehouses(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Склад 1", Type: "MAIN"})
	svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})
	svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Транзит", Type: "TRANSIT"})

	resp, err := svc.ListWarehouses(ctx, &v1.ListWarehousesRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 3)
	assert.NotNil(t, resp.Paginate)
	assert.Equal(t, int32(3), *resp.Paginate.Total)
}

func TestListWarehouses_TenantIsolation(t *testing.T) {
	svc, _, _ := setupService()
	ctx1 := ctxWithTenant(1)
	ctx2 := ctxWithTenant(2)

	svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{Name: "Склад 1", Type: "MAIN"})
	svc.CreateWarehouse(ctx2, &v1.CreateWarehouseRequest{Name: "Склад 2", Type: "MAIN"})

	resp, err := svc.ListWarehouses(ctx1, &v1.ListWarehousesRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "Склад 1", resp.Items[0].Name)
}

func TestUpdateWarehouse(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	created, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{
		Name: "Основной",
		Type: "MAIN",
	})

	resp, err := svc.UpdateWarehouse(ctx, &v1.UpdateWarehouseRequest{
		Id:       created.Warehouse.Id,
		Name:     "Основной обновлённый",
		Type:     "HALL",
		Address:  "Новый адрес",
		IsActive: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "Основной обновлённый", resp.Warehouse.Name)
	assert.Equal(t, "HALL", resp.Warehouse.Type)
	assert.Equal(t, "Новый адрес", resp.Warehouse.Address)
}

func TestUpdateWarehouse_NotFound(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	_, err := svc.UpdateWarehouse(ctx, &v1.UpdateWarehouseRequest{
		Id:   999,
		Name: "Не существует",
		Type: "MAIN",
	})
	require.Error(t, err)
}

func TestUpdateWarehouse_Deactivate(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	created, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{
		Name: "Склад",
		Type: "MAIN",
	})

	resp, err := svc.UpdateWarehouse(ctx, &v1.UpdateWarehouseRequest{
		Id:       created.Warehouse.Id,
		Name:     "Склад",
		Type:     "MAIN",
		IsActive: false,
	})
	require.NoError(t, err)
	assert.False(t, resp.Warehouse.IsActive)
}

func TestDeleteWarehouse(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	created, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{
		Name: "Удалить",
		Type: "MAIN",
	})

	_, err := svc.DeleteWarehouse(ctx, &v1.DeleteWarehouseRequest{Id: created.Warehouse.Id})
	require.NoError(t, err)

	_, err = svc.GetWarehouse(ctx, &v1.GetWarehouseRequest{Id: created.Warehouse.Id})
	require.Error(t, err)
}

func TestDeleteWarehouse_TenantIsolation(t *testing.T) {
	svc, _, _ := setupService()
	ctx1 := ctxWithTenant(1)
	ctx2 := ctxWithTenant(2)

	created, _ := svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{
		Name: "Склад",
		Type: "MAIN",
	})

	// Tenant 2 tries to delete tenant 1's warehouse
	_, err := svc.DeleteWarehouse(ctx2, &v1.DeleteWarehouseRequest{Id: created.Warehouse.Id})
	require.NoError(t, err) // no error, but should not affect tenant 1

	// Warehouse should still exist for tenant 1
	resp, err := svc.GetWarehouse(ctx1, &v1.GetWarehouseRequest{Id: created.Warehouse.Id})
	require.NoError(t, err)
	assert.Equal(t, "Склад", resp.Warehouse.Name)
}

// --- Stock Items Tests ---

func TestGetStockItems(t *testing.T) {
	svc, _, stockRepo := setupService()
	ctx := ctxWithTenant(1)

	// Create warehouse
	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// Seed stock items directly via repo mock
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(50),
		MinQuantity: decimal.NewFromInt(5),
	})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   200,
		Quantity:    decimal.NewFromInt(30),
		MinQuantity: decimal.NewFromInt(10),
	})

	resp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{
		WarehouseId: wh.Warehouse.Id,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.NotNil(t, resp.Paginate)
	assert.Equal(t, int32(2), *resp.Paginate.Total)
}

func TestGetStockItems_TenantIsolation(t *testing.T) {
	svc, _, stockRepo := setupService()
	ctx1 := ctxWithTenant(1)
	ctx2 := ctxWithTenant(2)

	wh, _ := svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{Name: "Склад", Type: "MAIN"})

	stockRepo.Upsert(ctx1, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(50),
	})

	// Tenant 2 should not see tenant 1's stock
	resp, err := svc.GetStockItems(ctx2, &v1.GetStockItemsRequest{
		WarehouseId: wh.Warehouse.Id,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 0)
}

func TestGetStockByProduct(t *testing.T) {
	svc, _, stockRepo := setupService()
	ctx := ctxWithTenant(1)

	// Create two warehouses
	wh1, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	wh2, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	productID := int64(100)

	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh1.Warehouse.Id,
		ProductID:   productID,
		Quantity:    decimal.NewFromInt(50),
		MinQuantity: decimal.NewFromInt(5),
	})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh2.Warehouse.Id,
		ProductID:   productID,
		Quantity:    decimal.NewFromInt(10),
		MinQuantity: decimal.NewFromInt(2),
	})

	resp, err := svc.GetStockByProduct(ctx, &v1.GetStockByProductRequest{
		ProductId: productID,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 2)

	// Verify both warehouses are represented
	warehouseIDs := make(map[int64]bool)
	for _, item := range resp.Items {
		warehouseIDs[item.WarehouseId] = true
		assert.Equal(t, productID, item.ProductId)
	}
	assert.True(t, warehouseIDs[wh1.Warehouse.Id])
	assert.True(t, warehouseIDs[wh2.Warehouse.Id])
}

func TestGetStockByProduct_TenantIsolation(t *testing.T) {
	svc, _, stockRepo := setupService()
	ctx1 := ctxWithTenant(1)
	ctx2 := ctxWithTenant(2)

	wh, _ := svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{Name: "Склад", Type: "MAIN"})

	stockRepo.Upsert(ctx1, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(50),
	})

	// Tenant 2 should not see tenant 1's stock
	resp, err := svc.GetStockByProduct(ctx2, &v1.GetStockByProductRequest{ProductId: 100})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 0)
}

func TestGetStockByProduct_NoTenant(t *testing.T) {
	svc, _, _ := setupService()
	ctx := context.Background()

	_, err := svc.GetStockByProduct(ctx, &v1.GetStockByProductRequest{ProductId: 100})
	require.Error(t, err)
}

func TestGetStockItems_NoTenant(t *testing.T) {
	svc, _, _ := setupService()
	ctx := context.Background()

	_, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: 1})
	require.Error(t, err)
}

func TestStockItem_UpsertUpdatesExisting(t *testing.T) {
	svc, _, stockRepo := setupService()
	ctx := ctxWithTenant(1)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Склад", Type: "MAIN"})

	// First upsert
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(50),
		MinQuantity: decimal.NewFromInt(5),
	})

	// Second upsert with updated quantity
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(75),
		MinQuantity: decimal.NewFromInt(10),
	})

	resp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1) // Should be 1 item, not 2
	assert.Equal(t, "75", resp.Items[0].Quantity)
	assert.Equal(t, "10", resp.Items[0].MinQuantity)
}

// --- Receipt Tests ---

func TestCreateReceipt_NoTenant(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := context.Background()

	_, err := svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: 1,
		Items:       []*v1.ReceiptItem{{ProductId: 1, Quantity: "10"}},
	})
	require.Error(t, err)
}

func TestCreateReceipt_NoActor(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenant(1) // no actor

	_, err := svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: 1,
		Items:       []*v1.ReceiptItem{{ProductId: 1, Quantity: "10"}},
	})
	require.Error(t, err)
}

func TestCreateReceipt_EmptyItems(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: 1,
		Items:       []*v1.ReceiptItem{},
	})
	require.Error(t, err)
}

func TestCreateReceipt_EmptyWarehouseId(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		Items: []*v1.ReceiptItem{{ProductId: 1, Quantity: "10"}},
	})
	require.Error(t, err)
}

func TestCreateReceipt_InvalidQuantity(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: 1,
		Items:       []*v1.ReceiptItem{{ProductId: 1, Quantity: "-5"}},
	})
	require.Error(t, err)

	_, err = svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: 1,
		Items:       []*v1.ReceiptItem{{ProductId: 1, Quantity: "0"}},
	})
	require.Error(t, err)

	_, err = svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: 1,
		Items:       []*v1.ReceiptItem{{ProductId: 1, Quantity: "abc"}},
	})
	require.Error(t, err)
}

func TestCreateReceipt_NewProduct(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	// Create warehouse first
	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	resp, err := svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 1)
	assert.Equal(t, "RECEIPT", resp.Movements[0].Type)
	assert.Equal(t, int64(100), resp.Movements[0].ProductId)
	assert.Equal(t, "50", resp.Movements[0].Quantity)
	assert.Equal(t, wh.Warehouse.Id, resp.Movements[0].WarehouseId)
	assert.NotEmpty(t, resp.Movements[0].CreatedAt)

	// Verify stock item was created
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, stockResp.Items, 1)
	assert.Equal(t, "50", stockResp.Items[0].Quantity)

	_ = stockRepo
}

func TestCreateReceipt_ExistingProduct_IncrementsQuantity(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// Seed existing stock
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(50),
	})

	// Receipt adds 30 more
	resp, err := svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "30"}},
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 1)
	assert.Equal(t, "30", resp.Movements[0].Quantity)

	// Stock should be 50 + 30 = 80
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, stockResp.Items, 1)
	assert.Equal(t, "80", stockResp.Items[0].Quantity)
}

func TestCreateReceipt_MultipleItems(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	resp, err := svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items: []*v1.ReceiptItem{
			{ProductId: 100, Quantity: "50"},
			{ProductId: 200, Quantity: "30"},
			{ProductId: 300, Quantity: "10"},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 3)

	// Verify all stock items created
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, stockResp.Items, 3)
}

func TestCreateReceipt_WithCounterparty(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	counterpartyID := int64(555)
	resp, err := svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId:    wh.Warehouse.Id,
		Items:          []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
		CounterpartyId: &counterpartyID,
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 1)
	require.NotNil(t, resp.Movements[0].CounterpartyId)
	assert.Equal(t, int64(555), *resp.Movements[0].CounterpartyId)
}

func TestCreateReceipt_TenantIsolation(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx1 := ctxWithTenantAndActor(1, 10)
	ctx2 := ctxWithTenantAndActor(2, 20)

	wh1, _ := svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{Name: "Склад 1", Type: "MAIN"})
	wh2, _ := svc.CreateWarehouse(ctx2, &v1.CreateWarehouseRequest{Name: "Склад 2", Type: "MAIN"})

	// Tenant 1 receipt
	svc.CreateReceipt(ctx1, &v1.CreateReceiptRequest{
		WarehouseId: wh1.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
	})

	// Tenant 2 receipt
	svc.CreateReceipt(ctx2, &v1.CreateReceiptRequest{
		WarehouseId: wh2.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "30"}},
	})

	// Tenant 1 should only see their stock
	stockResp, err := svc.GetStockItems(ctx1, &v1.GetStockItemsRequest{WarehouseId: wh1.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, stockResp.Items, 1)
	assert.Equal(t, "50", stockResp.Items[0].Quantity)
}

// --- ListMovements Tests ---

func TestListMovements(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// Create receipts
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items: []*v1.ReceiptItem{
			{ProductId: 100, Quantity: "50"},
			{ProductId: 200, Quantity: "30"},
		},
	})

	resp, err := svc.ListMovements(ctx, &v1.ListMovementsRequest{
		WarehouseId: wh.Warehouse.Id,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.NotNil(t, resp.Paginate)
	assert.Equal(t, int32(2), *resp.Paginate.Total)
}

func TestListMovements_TypeFilter(t *testing.T) {
	svc, _, _, movRepo := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// Create receipt
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
	})

	// Manually add a TRANSFER movement to the mock
	movRepo.movements[movRepo.nextID] = &ent.StockMovement{
		ID:          movRepo.nextID,
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		Type:        enum.Transfer,
		ProductID:   200,
		Quantity:    decimal.NewFromInt(10),
		CreatedBy:   10,
		CreatedAt:   time.Now(),
	}
	movRepo.nextID++

	// Filter by RECEIPT
	receiptType := "RECEIPT"
	resp, err := svc.ListMovements(ctx, &v1.ListMovementsRequest{
		WarehouseId: wh.Warehouse.Id,
		Type:        &receiptType,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "RECEIPT", resp.Items[0].Type)

	// Filter by TRANSFER
	transferType := "TRANSFER"
	resp, err = svc.ListMovements(ctx, &v1.ListMovementsRequest{
		WarehouseId: wh.Warehouse.Id,
		Type:        &transferType,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "TRANSFER", resp.Items[0].Type)
}

func TestListMovements_TenantIsolation(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx1 := ctxWithTenantAndActor(1, 10)
	ctx2 := ctxWithTenantAndActor(2, 20)

	wh1, _ := svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{Name: "Склад 1", Type: "MAIN"})
	wh2, _ := svc.CreateWarehouse(ctx2, &v1.CreateWarehouseRequest{Name: "Склад 2", Type: "MAIN"})

	svc.CreateReceipt(ctx1, &v1.CreateReceiptRequest{
		WarehouseId: wh1.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
	})
	svc.CreateReceipt(ctx2, &v1.CreateReceiptRequest{
		WarehouseId: wh2.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "30"}},
	})

	// Tenant 1 should only see their movements
	resp, err := svc.ListMovements(ctx1, &v1.ListMovementsRequest{WarehouseId: wh1.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "50", resp.Items[0].Quantity)
}

func TestListMovements_NoTenant(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := context.Background()

	_, err := svc.ListMovements(ctx, &v1.ListMovementsRequest{WarehouseId: 1})
	require.Error(t, err)
}

func TestListMovements_Pagination(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// Create 3 movements
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items: []*v1.ReceiptItem{
			{ProductId: 100, Quantity: "10"},
			{ProductId: 200, Quantity: "20"},
			{ProductId: 300, Quantity: "30"},
		},
	})

	// Get first page (limit 2)
	resp, err := svc.ListMovements(ctx, &v1.ListMovementsRequest{
		WarehouseId: wh.Warehouse.Id,
		Paginate:    &utils_v1.PaginateRequest{Limit: 2},
	})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, int32(3), *resp.Paginate.Total)
}

// --- Transfer Tests ---

func TestCreateTransfer_HappyPath(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	// Seed stock on source
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: source.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
	})

	// Transfer 20 units
	resp, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "20"}},
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 2)

	// First movement: outgoing from source (negative qty)
	assert.Equal(t, source.Warehouse.Id, resp.Movements[0].WarehouseId)
	assert.Equal(t, "TRANSFER", resp.Movements[0].Type)
	assert.Equal(t, "-20", resp.Movements[0].Quantity)

	// Second movement: incoming to target (positive qty)
	assert.Equal(t, target.Warehouse.Id, resp.Movements[1].WarehouseId)
	assert.Equal(t, "TRANSFER", resp.Movements[1].Type)
	assert.Equal(t, "20", resp.Movements[1].Quantity)

	// Verify source stock decreased: 50 - 20 = 30
	sourceStock, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: source.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, sourceStock.Items, 1)
	assert.Equal(t, "30", sourceStock.Items[0].Quantity)

	// Verify target stock increased: 0 + 20 = 20
	targetStock, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: target.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, targetStock.Items, 1)
	assert.Equal(t, "20", targetStock.Items[0].Quantity)
}

func TestCreateTransfer_MultipleItems(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	// Seed stock on source
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: source.Warehouse.Id,
		Items: []*v1.ReceiptItem{
			{ProductId: 100, Quantity: "50"},
			{ProductId: 200, Quantity: "30"},
		},
	})

	// Transfer both products
	resp, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items: []*v1.TransferItem{
			{ProductId: 100, Quantity: "10"},
			{ProductId: 200, Quantity: "5"},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 4) // 2 per item

	// Verify source stock
	sourceStock, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: source.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, sourceStock.Items, 2)

	// Verify target stock
	targetStock, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: target.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, targetStock.Items, 2)
}

func TestCreateTransfer_InsufficientStock(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	// Seed only 10 units
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: source.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "10"}},
	})

	// Try to transfer 20 — should fail
	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "20"}},
	})
	require.Error(t, err)
	assert.True(t, v1.IsInsufficientStock(err))

	// Source stock should remain unchanged
	sourceStock, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: source.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, sourceStock.Items, 1)
	assert.Equal(t, "10", sourceStock.Items[0].Quantity)
}

func TestCreateTransfer_NoStockOnSource(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	// No stock at all — should fail
	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "5"}},
	})
	require.Error(t, err)
	assert.True(t, v1.IsInsufficientStock(err))
}

func TestCreateTransfer_AtomicRollback(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	// Seed: product 100 has 50, product 200 has 5
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: source.Warehouse.Id,
		Items: []*v1.ReceiptItem{
			{ProductId: 100, Quantity: "50"},
			{ProductId: 200, Quantity: "5"},
		},
	})

	// Try to transfer: product 100 (10) OK, product 200 (20) should fail
	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items: []*v1.TransferItem{
			{ProductId: 100, Quantity: "10"},
			{ProductId: 200, Quantity: "20"}, // insufficient
		},
	})
	require.Error(t, err)

	// Atomic: product 100 should NOT have been decremented
	sourceStock, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: source.Warehouse.Id})
	require.NoError(t, err)
	for _, item := range sourceStock.Items {
		if item.ProductId == 100 {
			assert.Equal(t, "50", item.Quantity, "product 100 should remain at 50 after rollback")
		}
		if item.ProductId == 200 {
			assert.Equal(t, "5", item.Quantity, "product 200 should remain at 5 after rollback")
		}
	}

	// Target should have no stock
	targetStock, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: target.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, targetStock.Items, 0)
}

func TestCreateTransfer_SameWarehouse(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: wh.Warehouse.Id,
		TargetWarehouseId: wh.Warehouse.Id,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "10"}},
	})
	require.Error(t, err)
}

func TestCreateTransfer_NoTenant(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := context.Background()

	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: 1,
		TargetWarehouseId: 2,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "10"}},
	})
	require.Error(t, err)
}

func TestCreateTransfer_NoActor(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenant(1) // no actor

	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: 1,
		TargetWarehouseId: 2,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "10"}},
	})
	require.Error(t, err)
}

func TestCreateTransfer_EmptyItems(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: 1,
		TargetWarehouseId: 2,
		Items:             []*v1.TransferItem{},
	})
	require.Error(t, err)
}

func TestCreateTransfer_InvalidQuantity(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	// Negative quantity
	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: 1,
		TargetWarehouseId: 2,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "-5"}},
	})
	require.Error(t, err)

	// Zero quantity
	_, err = svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: 1,
		TargetWarehouseId: 2,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "0"}},
	})
	require.Error(t, err)

	// Non-numeric
	_, err = svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: 1,
		TargetWarehouseId: 2,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "abc"}},
	})
	require.Error(t, err)
}

func TestCreateTransfer_EmptySourceWarehouseId(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		TargetWarehouseId: 2,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "10"}},
	})
	require.Error(t, err)
}

func TestCreateTransfer_EmptyTargetWarehouseId(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: 1,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "10"}},
	})
	require.Error(t, err)
}

func TestCreateTransfer_TenantIsolation(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx1 := ctxWithTenantAndActor(1, 10)
	ctx2 := ctxWithTenantAndActor(2, 20)

	// Create warehouse for tenant 1
	source, _ := svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{Name: "Источник", Type: "MAIN"})
	// Create warehouse for tenant 2
	target, _ := svc.CreateWarehouse(ctx2, &v1.CreateWarehouseRequest{Name: "Приёмник", Type: "HALL"})

	// Seed stock on source
	svc.CreateReceipt(ctx1, &v1.CreateReceiptRequest{
		WarehouseId: source.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
	})

	// Tenant 1 tries to transfer to tenant 2's warehouse — should fail
	_, err := svc.CreateTransfer(ctx1, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "10"}},
	})
	require.Error(t, err)
}

func TestCreateTransfer_TransferAllStock(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	// Seed 10 units
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: source.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "10"}},
	})

	// Transfer all 10 units
	resp, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "10"}},
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 2)

	// Source should have 0
	sourceStock, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: source.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, sourceStock.Items, 1)
	assert.Equal(t, "0", sourceStock.Items[0].Quantity)

	// Target should have 10
	targetStock, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: target.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, targetStock.Items, 1)
	assert.Equal(t, "10", targetStock.Items[0].Quantity)
}

func TestCreateTransfer_TransferToExistingStock(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	// Seed stock on both warehouses
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: source.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
	})
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: target.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "10"}},
	})

	// Transfer 20 units
	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "20"}},
	})
	require.NoError(t, err)

	// Source: 50 - 20 = 30
	sourceStock, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: source.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, sourceStock.Items, 1)
	assert.Equal(t, "30", sourceStock.Items[0].Quantity)

	// Target: 10 + 20 = 30
	targetStock, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: target.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, targetStock.Items, 1)
	assert.Equal(t, "30", targetStock.Items[0].Quantity)
}

func TestCreateTransfer_MovementsVisibleInListMovements(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	// Seed stock
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: source.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
	})

	// Transfer
	svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "10"}},
	})

	// List movements for source — should have receipt + transfer
	transferType := "TRANSFER"
	sourceMovs, err := svc.ListMovements(ctx, &v1.ListMovementsRequest{
		WarehouseId: source.Warehouse.Id,
		Type:        &transferType,
	})
	require.NoError(t, err)
	assert.Len(t, sourceMovs.Items, 1)
	assert.Equal(t, "TRANSFER", sourceMovs.Items[0].Type)

	// List movements for target
	targetMovs, err := svc.ListMovements(ctx, &v1.ListMovementsRequest{
		WarehouseId: target.Warehouse.Id,
		Type:        &transferType,
	})
	require.NoError(t, err)
	assert.Len(t, targetMovs.Items, 1)
	assert.Equal(t, "TRANSFER", targetMovs.Items[0].Type)
}

// --- WriteOff Tests ---

func TestCreateWriteOff_HappyPath(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// Seed stock
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
	})

	// Write off 10 units
	resp, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "10", Reason: "Брак"}},
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 1)
	assert.Equal(t, "WRITE_OFF", resp.Movements[0].Type)
	assert.Equal(t, "-10", resp.Movements[0].Quantity)
	assert.Equal(t, int64(100), resp.Movements[0].ProductId)
	require.NotNil(t, resp.Movements[0].Reason)
	assert.Equal(t, "Брак", *resp.Movements[0].Reason)

	// Verify stock decreased: 50 - 10 = 40
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, stockResp.Items, 1)
	assert.Equal(t, "40", stockResp.Items[0].Quantity)
}

func TestCreateWriteOff_MultipleItems(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// Seed stock
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items: []*v1.ReceiptItem{
			{ProductId: 100, Quantity: "50"},
			{ProductId: 200, Quantity: "30"},
		},
	})

	// Write off multiple items
	resp, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: wh.Warehouse.Id,
		Items: []*v1.WriteOffItem{
			{ProductId: 100, Quantity: "5", Reason: "Просрочено"},
			{ProductId: 200, Quantity: "3", Reason: "Повреждено"},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 2)

	// Verify stock: 50-5=45, 30-3=27
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, stockResp.Items, 2)
}

func TestCreateWriteOff_InsufficientStock(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// Seed only 10 units
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "10"}},
	})

	// Try to write off 20 — should fail
	_, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "20", Reason: "Утеря"}},
	})
	require.Error(t, err)
	assert.True(t, v1.IsInsufficientStock(err))

	// Stock should remain unchanged
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, stockResp.Items, 1)
	assert.Equal(t, "10", stockResp.Items[0].Quantity)
}

func TestCreateWriteOff_NoStock(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// No stock at all
	_, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "5", Reason: "Утеря"}},
	})
	require.Error(t, err)
	assert.True(t, v1.IsInsufficientStock(err))
}

func TestCreateWriteOff_AtomicRollback(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// Seed: product 100 has 50, product 200 has 5
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items: []*v1.ReceiptItem{
			{ProductId: 100, Quantity: "50"},
			{ProductId: 200, Quantity: "5"},
		},
	})

	// Try write-off: product 100 (10) OK, product 200 (20) should fail
	_, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: wh.Warehouse.Id,
		Items: []*v1.WriteOffItem{
			{ProductId: 100, Quantity: "10", Reason: "Брак"},
			{ProductId: 200, Quantity: "20", Reason: "Утеря"}, // insufficient
		},
	})
	require.Error(t, err)

	// Atomic: product 100 should NOT have been decremented
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	for _, item := range stockResp.Items {
		if item.ProductId == 100 {
			assert.Equal(t, "50", item.Quantity, "product 100 should remain at 50 after rollback")
		}
		if item.ProductId == 200 {
			assert.Equal(t, "5", item.Quantity, "product 200 should remain at 5 after rollback")
		}
	}
}

func TestCreateWriteOff_WriteOffAllStock(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "10"}},
	})

	// Write off all 10 units
	resp, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "10", Reason: "Списание"}},
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 1)

	// Stock should be 0
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, stockResp.Items, 1)
	assert.Equal(t, "0", stockResp.Items[0].Quantity)
}

func TestCreateWriteOff_EmptyReason(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: 1,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "10", Reason: ""}},
	})
	require.Error(t, err)
}

func TestCreateWriteOff_WhitespaceReason(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: 1,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "10", Reason: "   "}},
	})
	require.Error(t, err)
}

func TestCreateWriteOff_NoTenant(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := context.Background()

	_, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: 1,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "10", Reason: "Брак"}},
	})
	require.Error(t, err)
}

func TestCreateWriteOff_NoActor(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenant(1)

	_, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: 1,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "10", Reason: "Брак"}},
	})
	require.Error(t, err)
}

func TestCreateWriteOff_EmptyItems(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: 1,
		Items:       []*v1.WriteOffItem{},
	})
	require.Error(t, err)
}

func TestCreateWriteOff_EmptyWarehouseId(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		Items: []*v1.WriteOffItem{{ProductId: 100, Quantity: "10", Reason: "Брак"}},
	})
	require.Error(t, err)
}

func TestCreateWriteOff_InvalidQuantity(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: 1,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "-5", Reason: "Брак"}},
	})
	require.Error(t, err)

	_, err = svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: 1,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "0", Reason: "Брак"}},
	})
	require.Error(t, err)

	_, err = svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: 1,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "abc", Reason: "Брак"}},
	})
	require.Error(t, err)
}

func TestCreateWriteOff_MovementsVisibleInListMovements(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
	})

	svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "5", Reason: "Брак"}},
	})

	writeOffType := "WRITE_OFF"
	movs, err := svc.ListMovements(ctx, &v1.ListMovementsRequest{
		WarehouseId: wh.Warehouse.Id,
		Type:        &writeOffType,
	})
	require.NoError(t, err)
	assert.Len(t, movs.Items, 1)
	assert.Equal(t, "WRITE_OFF", movs.Items[0].Type)
}

func TestCreateWriteOff_TenantIsolation(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx1 := ctxWithTenantAndActor(1, 10)
	ctx2 := ctxWithTenantAndActor(2, 20)

	wh1, _ := svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{Name: "Склад 1", Type: "MAIN"})
	wh2, _ := svc.CreateWarehouse(ctx2, &v1.CreateWarehouseRequest{Name: "Склад 2", Type: "MAIN"})

	// Seed stock for both tenants
	svc.CreateReceipt(ctx1, &v1.CreateReceiptRequest{
		WarehouseId: wh1.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "50"}},
	})
	svc.CreateReceipt(ctx2, &v1.CreateReceiptRequest{
		WarehouseId: wh2.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "30"}},
	})

	// Write off from tenant 1
	svc.CreateWriteOff(ctx1, &v1.CreateWriteOffRequest{
		WarehouseId: wh1.Warehouse.Id,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "10", Reason: "Брак"}},
	})

	// Tenant 1: 50 - 10 = 40
	stockResp, err := svc.GetStockItems(ctx1, &v1.GetStockItemsRequest{WarehouseId: wh1.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, stockResp.Items, 1)
	assert.Equal(t, "40", stockResp.Items[0].Quantity)

	// Tenant 2: should remain 30 (unaffected)
	stockResp2, err := svc.GetStockItems(ctx2, &v1.GetStockItemsRequest{WarehouseId: wh2.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, stockResp2.Items, 1)
	assert.Equal(t, "30", stockResp2.Items[0].Quantity)
}

// --- Gift Tests ---

func TestCreateGift_HappyPath(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	resp, err := svc.CreateGift(ctx, &v1.CreateGiftRequest{
		WarehouseId:    wh.Warehouse.Id,
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "20"}},
		CounterpartyId: 555,
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 1)
	assert.Equal(t, "GIFT", resp.Movements[0].Type)
	assert.Equal(t, "20", resp.Movements[0].Quantity)
	assert.Equal(t, int64(100), resp.Movements[0].ProductId)
	require.NotNil(t, resp.Movements[0].CounterpartyId)
	assert.Equal(t, int64(555), *resp.Movements[0].CounterpartyId)

	// Verify stock increased
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, stockResp.Items, 1)
	assert.Equal(t, "20", stockResp.Items[0].Quantity)
}

func TestCreateGift_MultipleItems(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	resp, err := svc.CreateGift(ctx, &v1.CreateGiftRequest{
		WarehouseId: wh.Warehouse.Id,
		Items: []*v1.GiftItem{
			{ProductId: 100, Quantity: "20"},
			{ProductId: 200, Quantity: "10"},
		},
		CounterpartyId: 555,
	})
	require.NoError(t, err)
	require.Len(t, resp.Movements, 2)

	// Verify all stock items created
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, stockResp.Items, 2)
}

func TestCreateGift_AddsToExistingStock(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// Seed existing stock
	svc.CreateReceipt(ctx, &v1.CreateReceiptRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.ReceiptItem{{ProductId: 100, Quantity: "30"}},
	})

	// Gift adds 20 more
	_, err := svc.CreateGift(ctx, &v1.CreateGiftRequest{
		WarehouseId:    wh.Warehouse.Id,
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "20"}},
		CounterpartyId: 555,
	})
	require.NoError(t, err)

	// Stock should be 30 + 20 = 50
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, stockResp.Items, 1)
	assert.Equal(t, "50", stockResp.Items[0].Quantity)
}

func TestCreateGift_NoCounterparty(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateGift(ctx, &v1.CreateGiftRequest{
		WarehouseId:    1,
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "20"}},
		CounterpartyId: 0,
	})
	require.Error(t, err)
}

func TestCreateGift_NoTenant(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := context.Background()

	_, err := svc.CreateGift(ctx, &v1.CreateGiftRequest{
		WarehouseId:    1,
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "20"}},
		CounterpartyId: 555,
	})
	require.Error(t, err)
}

func TestCreateGift_NoActor(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenant(1)

	_, err := svc.CreateGift(ctx, &v1.CreateGiftRequest{
		WarehouseId:    1,
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "20"}},
		CounterpartyId: 555,
	})
	require.Error(t, err)
}

func TestCreateGift_EmptyItems(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateGift(ctx, &v1.CreateGiftRequest{
		WarehouseId:    1,
		Items:          []*v1.GiftItem{},
		CounterpartyId: 555,
	})
	require.Error(t, err)
}

func TestCreateGift_EmptyWarehouseId(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateGift(ctx, &v1.CreateGiftRequest{
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "20"}},
		CounterpartyId: 555,
	})
	require.Error(t, err)
}

func TestCreateGift_InvalidQuantity(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.CreateGift(ctx, &v1.CreateGiftRequest{
		WarehouseId:    1,
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "-5"}},
		CounterpartyId: 555,
	})
	require.Error(t, err)

	_, err = svc.CreateGift(ctx, &v1.CreateGiftRequest{
		WarehouseId:    1,
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "0"}},
		CounterpartyId: 555,
	})
	require.Error(t, err)

	_, err = svc.CreateGift(ctx, &v1.CreateGiftRequest{
		WarehouseId:    1,
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "abc"}},
		CounterpartyId: 555,
	})
	require.Error(t, err)
}

func TestCreateGift_MovementsVisibleInListMovements(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	svc.CreateGift(ctx, &v1.CreateGiftRequest{
		WarehouseId:    wh.Warehouse.Id,
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "20"}},
		CounterpartyId: 555,
	})

	giftType := "GIFT"
	movs, err := svc.ListMovements(ctx, &v1.ListMovementsRequest{
		WarehouseId: wh.Warehouse.Id,
		Type:        &giftType,
	})
	require.NoError(t, err)
	assert.Len(t, movs.Items, 1)
	assert.Equal(t, "GIFT", movs.Items[0].Type)
}

func TestCreateGift_TenantIsolation(t *testing.T) {
	svc, _, _, _ := setupServiceWithMovements()
	ctx1 := ctxWithTenantAndActor(1, 10)
	ctx2 := ctxWithTenantAndActor(2, 20)

	wh1, _ := svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{Name: "Склад 1", Type: "MAIN"})
	wh2, _ := svc.CreateWarehouse(ctx2, &v1.CreateWarehouseRequest{Name: "Склад 2", Type: "MAIN"})

	svc.CreateGift(ctx1, &v1.CreateGiftRequest{
		WarehouseId:    wh1.Warehouse.Id,
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "20"}},
		CounterpartyId: 555,
	})
	svc.CreateGift(ctx2, &v1.CreateGiftRequest{
		WarehouseId:    wh2.Warehouse.Id,
		Items:          []*v1.GiftItem{{ProductId: 100, Quantity: "10"}},
		CounterpartyId: 666,
	})

	// Tenant 1 should only see their stock
	stockResp, err := svc.GetStockItems(ctx1, &v1.GetStockItemsRequest{WarehouseId: wh1.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, stockResp.Items, 1)
	assert.Equal(t, "20", stockResp.Items[0].Quantity)
}

// --- SetMinQuantity Tests ---

func TestSetMinQuantity_HappyPath(t *testing.T) {
	svc, _, stockRepo := setupService()
	ctx := ctxWithTenant(1)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(50),
	})

	resp, err := svc.SetMinQuantity(ctx, &v1.SetMinQuantityRequest{
		WarehouseId: wh.Warehouse.Id,
		ProductId:   100,
		MinQuantity: "10",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Item)
	assert.Equal(t, "10", resp.Item.MinQuantity)
	assert.Equal(t, int64(100), resp.Item.ProductId)
}

func TestSetMinQuantity_ZeroValue(t *testing.T) {
	svc, _, stockRepo := setupService()
	ctx := ctxWithTenant(1)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(50),
		MinQuantity: decimal.NewFromInt(10),
	})

	resp, err := svc.SetMinQuantity(ctx, &v1.SetMinQuantityRequest{
		WarehouseId: wh.Warehouse.Id,
		ProductId:   100,
		MinQuantity: "0",
	})
	require.NoError(t, err)
	assert.Equal(t, "0", resp.Item.MinQuantity)
}

func TestSetMinQuantity_NoTenant(t *testing.T) {
	svc, _, _ := setupService()
	ctx := context.Background()

	_, err := svc.SetMinQuantity(ctx, &v1.SetMinQuantityRequest{
		WarehouseId: 1,
		ProductId:   100,
		MinQuantity: "10",
	})
	require.Error(t, err)
}

func TestSetMinQuantity_EmptyWarehouseId(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	_, err := svc.SetMinQuantity(ctx, &v1.SetMinQuantityRequest{
		ProductId:   100,
		MinQuantity: "10",
	})
	require.Error(t, err)
}

func TestSetMinQuantity_EmptyProductId(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	_, err := svc.SetMinQuantity(ctx, &v1.SetMinQuantityRequest{
		WarehouseId: 1,
		MinQuantity: "10",
	})
	require.Error(t, err)
}

func TestSetMinQuantity_NegativeValue(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	_, err := svc.SetMinQuantity(ctx, &v1.SetMinQuantityRequest{
		WarehouseId: 1,
		ProductId:   100,
		MinQuantity: "-5",
	})
	require.Error(t, err)
}

func TestSetMinQuantity_InvalidValue(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	_, err := svc.SetMinQuantity(ctx, &v1.SetMinQuantityRequest{
		WarehouseId: 1,
		ProductId:   100,
		MinQuantity: "abc",
	})
	require.Error(t, err)
}

// --- GetLowStockItems Tests ---

func TestGetLowStockItems_ReturnsOnlyLow(t *testing.T) {
	svc, _, stockRepo := setupService()
	ctx := ctxWithTenant(1)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	// Product 100: qty=5, min=10 => LOW
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(5),
		MinQuantity: decimal.NewFromInt(10),
	})
	// Product 200: qty=50, min=10 => OK
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   200,
		Quantity:    decimal.NewFromInt(50),
		MinQuantity: decimal.NewFromInt(10),
	})
	// Product 300: qty=10, min=10 => LOW (equal)
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   300,
		Quantity:    decimal.NewFromInt(10),
		MinQuantity: decimal.NewFromInt(10),
	})
	// Product 400: qty=0, min=0 => NOT low (min not set)
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   400,
		Quantity:    decimal.NewFromInt(0),
		MinQuantity: decimal.Zero,
	})

	resp, err := svc.GetLowStockItems(ctx, &v1.GetLowStockItemsRequest{
		WarehouseId: wh.Warehouse.Id,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 2) // products 100 and 300

	productIDs := map[int64]bool{}
	for _, item := range resp.Items {
		productIDs[item.ProductId] = true
	}
	assert.True(t, productIDs[100])
	assert.True(t, productIDs[300])
}

func TestGetLowStockItems_NoTenant(t *testing.T) {
	svc, _, _ := setupService()
	ctx := context.Background()

	_, err := svc.GetLowStockItems(ctx, &v1.GetLowStockItemsRequest{WarehouseId: 1})
	require.Error(t, err)
}

func TestGetLowStockItems_EmptyWarehouseId(t *testing.T) {
	svc, _, _ := setupService()
	ctx := ctxWithTenant(1)

	_, err := svc.GetLowStockItems(ctx, &v1.GetLowStockItemsRequest{})
	require.Error(t, err)
}

func TestGetLowStockItems_TenantIsolation(t *testing.T) {
	svc, _, stockRepo := setupService()
	ctx1 := ctxWithTenant(1)
	ctx2 := ctxWithTenant(2)

	wh1, _ := svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{Name: "Склад 1", Type: "MAIN"})
	wh2, _ := svc.CreateWarehouse(ctx2, &v1.CreateWarehouseRequest{Name: "Склад 2", Type: "MAIN"})

	stockRepo.Upsert(ctx1, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh1.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(2),
		MinQuantity: decimal.NewFromInt(10),
	})
	stockRepo.Upsert(ctx2, data.StockItemDto{
		TenantID:    2,
		WarehouseID: wh2.Warehouse.Id,
		ProductID:   200,
		Quantity:    decimal.NewFromInt(1),
		MinQuantity: decimal.NewFromInt(5),
	})

	resp, err := svc.GetLowStockItems(ctx1, &v1.GetLowStockItemsRequest{WarehouseId: wh1.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, int64(100), resp.Items[0].ProductId)
}

// --- Low Stock Notification Tests ---

func TestTransfer_PublishesLowStockEvent(t *testing.T) {
	svc, _, stockRepo, _, publisher := setupServiceWithPublisher()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	// Seed stock with min_quantity set
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: source.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(15),
		MinQuantity: decimal.NewFromInt(10),
	})

	// Transfer 10 units: 15 - 10 = 5, which is <= min(10)
	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "10"}},
	})
	require.NoError(t, err)

	require.Len(t, publisher.events, 1)
	assert.Equal(t, int64(1), publisher.events[0].TenantID)
	assert.Equal(t, source.Warehouse.Id, publisher.events[0].WarehouseID)
	assert.Equal(t, int64(100), publisher.events[0].ProductID)
	assert.Equal(t, "5", publisher.events[0].Quantity)
	assert.Equal(t, "10", publisher.events[0].MinQuantity)
}

func TestTransfer_NoEventWhenAboveThreshold(t *testing.T) {
	svc, _, stockRepo, _, publisher := setupServiceWithPublisher()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: source.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(50),
		MinQuantity: decimal.NewFromInt(10),
	})

	// Transfer 5: 50 - 5 = 45 > 10, no event
	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "5"}},
	})
	require.NoError(t, err)

	assert.Len(t, publisher.events, 0)
}

func TestTransfer_NoEventWhenMinQuantityZero(t *testing.T) {
	svc, _, stockRepo, _, publisher := setupServiceWithPublisher()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: source.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(10),
		MinQuantity: decimal.Zero,
	})

	// Transfer all: 10 - 10 = 0, but min=0 so no event
	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items:             []*v1.TransferItem{{ProductId: 100, Quantity: "10"}},
	})
	require.NoError(t, err)

	assert.Len(t, publisher.events, 0)
}

func TestTransfer_MultipleItems_OnlyLowOneFires(t *testing.T) {
	svc, _, stockRepo, _, publisher := setupServiceWithPublisher()
	ctx := ctxWithTenantAndActor(1, 10)

	source, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	target, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Зал", Type: "HALL"})

	// Product 100: qty=15, min=10 — will drop below
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: source.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(15),
		MinQuantity: decimal.NewFromInt(10),
	})
	// Product 200: qty=50, min=5 — will stay above
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: source.Warehouse.Id,
		ProductID:   200,
		Quantity:    decimal.NewFromInt(50),
		MinQuantity: decimal.NewFromInt(5),
	})

	_, err := svc.CreateTransfer(ctx, &v1.CreateTransferRequest{
		SourceWarehouseId: source.Warehouse.Id,
		TargetWarehouseId: target.Warehouse.Id,
		Items: []*v1.TransferItem{
			{ProductId: 100, Quantity: "10"},
			{ProductId: 200, Quantity: "5"},
		},
	})
	require.NoError(t, err)

	// Only product 100 should trigger (5 <= 10)
	require.Len(t, publisher.events, 1)
	assert.Equal(t, int64(100), publisher.events[0].ProductID)
}

func TestWriteOff_PublishesLowStockEvent(t *testing.T) {
	svc, _, stockRepo, _, publisher := setupServiceWithPublisher()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID:    1,
		WarehouseID: wh.Warehouse.Id,
		ProductID:   100,
		Quantity:    decimal.NewFromInt(12),
		MinQuantity: decimal.NewFromInt(10),
	})

	// Write off 5 units: 12 - 5 = 7, which is <= 10
	_, err := svc.CreateWriteOff(ctx, &v1.CreateWriteOffRequest{
		WarehouseId: wh.Warehouse.Id,
		Items:       []*v1.WriteOffItem{{ProductId: 100, Quantity: "5", Reason: "Брак"}},
	})
	require.NoError(t, err)

	require.Len(t, publisher.events, 1)
	assert.Equal(t, int64(1), publisher.events[0].TenantID)
	assert.Equal(t, wh.Warehouse.Id, publisher.events[0].WarehouseID)
	assert.Equal(t, int64(100), publisher.events[0].ProductID)
	assert.Equal(t, "7", publisher.events[0].Quantity)
	assert.Equal(t, "10", publisher.events[0].MinQuantity)
}

// --- Inventory Tests ---

func TestStartInventory_HappyPath(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 200, Quantity: decimal.NewFromInt(30),
	})

	resp, err := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	require.NotNil(t, resp.Inventory)
	assert.Equal(t, "IN_PROGRESS", resp.Inventory.Status)
	assert.Equal(t, wh.Warehouse.Id, resp.Inventory.WarehouseId)
	assert.Equal(t, int64(1), resp.Inventory.TenantId)
	assert.Equal(t, int64(10), resp.Inventory.CreatedBy)
	assert.Len(t, resp.Inventory.Items, 2)
	assert.Nil(t, resp.Inventory.CompletedAt)
}

func TestStartInventory_EmptyWarehouse(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Пустой", Type: "MAIN"})

	resp, err := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	assert.Equal(t, "IN_PROGRESS", resp.Inventory.Status)
	assert.Len(t, resp.Inventory.Items, 0)
}

func TestStartInventory_NoTenant(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := context.Background()

	_, err := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: 1})
	require.Error(t, err)
}

func TestStartInventory_NoActor(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := ctxWithTenant(1)

	_, err := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: 1})
	require.Error(t, err)
}

func TestStartInventory_EmptyWarehouseId(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.StartInventory(ctx, &v1.StartInventoryRequest{})
	require.Error(t, err)
}

func TestStartInventory_SnapshotsCorrectQuantities(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})

	qty, _ := decimal.NewFromString("42.5")
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: qty,
	})

	resp, err := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, resp.Inventory.Items, 1)
	assert.Equal(t, "42.5", resp.Inventory.Items[0].ExpectedQty)
	assert.Nil(t, resp.Inventory.Items[0].ActualQty)
	assert.Nil(t, resp.Inventory.Items[0].Difference)
}

func TestSetInventoryItem_HappyPath(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})

	inv, _ := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})

	resp, err := svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: inv.Inventory.Id,
		ProductId:   100,
		ActualQty:   "45",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Item)
	assert.Equal(t, int64(100), resp.Item.ProductId)
	require.NotNil(t, resp.Item.ActualQty)
	assert.Equal(t, "45", *resp.Item.ActualQty)
	require.NotNil(t, resp.Item.Difference)
	assert.Equal(t, "-5", *resp.Item.Difference)
}

func TestSetInventoryItem_Surplus(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})

	inv, _ := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})

	resp, err := svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: inv.Inventory.Id,
		ProductId:   100,
		ActualQty:   "55",
	})
	require.NoError(t, err)
	assert.Equal(t, "5", *resp.Item.Difference)
}

func TestSetInventoryItem_NewProduct(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})

	inv, _ := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})

	resp, err := svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: inv.Inventory.Id,
		ProductId:   999,
		ActualQty:   "7",
	})
	require.NoError(t, err)
	assert.Equal(t, "0", resp.Item.ExpectedQty)
	assert.Equal(t, "7", *resp.Item.ActualQty)
	assert.Equal(t, "7", *resp.Item.Difference)
}

func TestSetInventoryItem_ZeroQty(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})

	inv, _ := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})

	resp, err := svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: inv.Inventory.Id,
		ProductId:   100,
		ActualQty:   "0",
	})
	require.NoError(t, err)
	assert.Equal(t, "0", *resp.Item.ActualQty)
	assert.Equal(t, "-50", *resp.Item.Difference)
}

func TestSetInventoryItem_InvalidQty(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	_, err := svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: 1, ProductId: 100, ActualQty: "-5",
	})
	require.Error(t, err)

	_, err = svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: 1, ProductId: 100, ActualQty: "abc",
	})
	require.Error(t, err)
}

func TestSetInventoryItem_NoTenant(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	_, err := svc.SetInventoryItem(context.Background(), &v1.SetInventoryItemRequest{
		InventoryId: 1, ProductId: 100, ActualQty: "10",
	})
	require.Error(t, err)
}

func TestSetInventoryItem_EmptyInventoryId(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)
	_, err := svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{ProductId: 100, ActualQty: "10"})
	require.Error(t, err)
}

func TestSetInventoryItem_EmptyProductId(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)
	_, err := svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{InventoryId: 1, ActualQty: "10"})
	require.Error(t, err)
}

func TestSetInventoryItem_CompletedInventory(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})

	inv, _ := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})
	svc.CompleteInventory(ctx, &v1.CompleteInventoryRequest{InventoryId: inv.Inventory.Id})

	_, err := svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: inv.Inventory.Id, ProductId: 100, ActualQty: "40",
	})
	require.Error(t, err)
}

func TestSetInventoryItem_InventoryNotFound(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)
	_, err := svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: 999, ProductId: 100, ActualQty: "10",
	})
	require.Error(t, err)
}

func TestCompleteInventory_HappyPath(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})

	inv, _ := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})
	svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: inv.Inventory.Id, ProductId: 100, ActualQty: "48",
	})

	resp, err := svc.CompleteInventory(ctx, &v1.CompleteInventoryRequest{InventoryId: inv.Inventory.Id})
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", resp.Inventory.Status)
	assert.NotNil(t, resp.Inventory.CompletedAt)
	assert.Len(t, resp.Inventory.Items, 1)
}

func TestCompleteInventory_DoesNotAdjustStock(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})

	inv, _ := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})
	svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: inv.Inventory.Id, ProductId: 100, ActualQty: "40",
	})
	svc.CompleteInventory(ctx, &v1.CompleteInventoryRequest{InventoryId: inv.Inventory.Id})

	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	require.Len(t, stockResp.Items, 1)
	assert.Equal(t, "50", stockResp.Items[0].Quantity)
}

func TestCompleteInventory_AlreadyCompleted(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})

	inv, _ := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})
	svc.CompleteInventory(ctx, &v1.CompleteInventoryRequest{InventoryId: inv.Inventory.Id})

	_, err := svc.CompleteInventory(ctx, &v1.CompleteInventoryRequest{InventoryId: inv.Inventory.Id})
	require.Error(t, err)
}

func TestCompleteInventory_NotFound(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)
	_, err := svc.CompleteInventory(ctx, &v1.CompleteInventoryRequest{InventoryId: 999})
	require.Error(t, err)
}

func TestCompleteInventory_NoTenant(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	_, err := svc.CompleteInventory(context.Background(), &v1.CompleteInventoryRequest{InventoryId: 1})
	require.Error(t, err)
}

func TestCompleteInventory_EmptyInventoryId(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)
	_, err := svc.CompleteInventory(ctx, &v1.CompleteInventoryRequest{})
	require.Error(t, err)
}

func TestGetInventory_HappyPath(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 200, Quantity: decimal.NewFromInt(30),
	})

	inv, _ := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})

	resp, err := svc.GetInventory(ctx, &v1.GetInventoryRequest{InventoryId: inv.Inventory.Id})
	require.NoError(t, err)
	assert.Equal(t, inv.Inventory.Id, resp.Inventory.Id)
	assert.Equal(t, "IN_PROGRESS", resp.Inventory.Status)
	assert.Len(t, resp.Inventory.Items, 2)
}

func TestGetInventory_NotFound(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)
	_, err := svc.GetInventory(ctx, &v1.GetInventoryRequest{InventoryId: 999})
	require.Error(t, err)
}

func TestGetInventory_NoTenant(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	_, err := svc.GetInventory(context.Background(), &v1.GetInventoryRequest{InventoryId: 1})
	require.Error(t, err)
}

func TestGetInventory_EmptyInventoryId(t *testing.T) {
	svc, _, _, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)
	_, err := svc.GetInventory(ctx, &v1.GetInventoryRequest{})
	require.Error(t, err)
}

func TestGetInventory_TenantIsolation(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx1 := ctxWithTenantAndActor(1, 10)
	ctx2 := ctxWithTenantAndActor(2, 20)

	wh, _ := svc.CreateWarehouse(ctx1, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	stockRepo.Upsert(ctx1, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})

	inv, _ := svc.StartInventory(ctx1, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})

	_, err := svc.GetInventory(ctx2, &v1.GetInventoryRequest{InventoryId: inv.Inventory.Id})
	require.Error(t, err)
}

func TestInventory_FullFlow(t *testing.T) {
	svc, _, stockRepo, _ := setupServiceWithInventory()
	ctx := ctxWithTenantAndActor(1, 10)

	wh, _ := svc.CreateWarehouse(ctx, &v1.CreateWarehouseRequest{Name: "Основной", Type: "MAIN"})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 100, Quantity: decimal.NewFromInt(50),
	})
	stockRepo.Upsert(ctx, data.StockItemDto{
		TenantID: 1, WarehouseID: wh.Warehouse.Id, ProductID: 200, Quantity: decimal.NewFromInt(30),
	})

	// Start
	inv, err := svc.StartInventory(ctx, &v1.StartInventoryRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	assert.Equal(t, "IN_PROGRESS", inv.Inventory.Status)
	assert.Len(t, inv.Inventory.Items, 2)

	// Set items
	resp1, err := svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: inv.Inventory.Id, ProductId: 100, ActualQty: "48",
	})
	require.NoError(t, err)
	assert.Equal(t, "-2", *resp1.Item.Difference)

	resp2, err := svc.SetInventoryItem(ctx, &v1.SetInventoryItemRequest{
		InventoryId: inv.Inventory.Id, ProductId: 200, ActualQty: "32",
	})
	require.NoError(t, err)
	assert.Equal(t, "2", *resp2.Item.Difference)

	// Complete
	completed, err := svc.CompleteInventory(ctx, &v1.CompleteInventoryRequest{InventoryId: inv.Inventory.Id})
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", completed.Inventory.Status)
	assert.NotNil(t, completed.Inventory.CompletedAt)

	// Get and verify
	got, err := svc.GetInventory(ctx, &v1.GetInventoryRequest{InventoryId: inv.Inventory.Id})
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", got.Inventory.Status)
	assert.Len(t, got.Inventory.Items, 2)

	// Stock remains unchanged
	stockResp, err := svc.GetStockItems(ctx, &v1.GetStockItemsRequest{WarehouseId: wh.Warehouse.Id})
	require.NoError(t, err)
	assert.Len(t, stockResp.Items, 2)
}
