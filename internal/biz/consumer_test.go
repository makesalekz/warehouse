package biz

import (
	"context"
	"fmt"
	"testing"

	"gitlab.calendaria.team/services/warehouse/ent"
	"gitlab.calendaria.team/services/warehouse/ent/enum"
	"gitlab.calendaria.team/services/warehouse/internal/data"
	"gitlab.calendaria.team/services/warehouse/messages"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Minimal mocks for consumer tests ---

type mockWarehousesRepoForConsumer struct {
	hall *ent.Warehouse
	main *ent.Warehouse
}

func (m *mockWarehousesRepoForConsumer) FindHallByTenant(_ context.Context, _ int64) (*ent.Warehouse, error) {
	if m.hall == nil {
		return nil, assert.AnError
	}
	return m.hall, nil
}

func (m *mockWarehousesRepoForConsumer) FindMainByTenant(_ context.Context, _ int64) (*ent.Warehouse, error) {
	if m.main == nil {
		return nil, assert.AnError
	}
	return m.main, nil
}

func (m *mockWarehousesRepoForConsumer) Create(_ context.Context, _ data.WarehouseDto) (*ent.Warehouse, error) {
	return nil, nil
}
func (m *mockWarehousesRepoForConsumer) Get(_ context.Context, _, _ int64) (*ent.Warehouse, error) {
	return nil, nil
}
func (m *mockWarehousesRepoForConsumer) List(_ context.Context, _ int64, _ *int64, _ *utils_v1.PaginateRequest) ([]*ent.Warehouse, error) {
	return nil, nil
}
func (m *mockWarehousesRepoForConsumer) Update(_ context.Context, _ data.WarehouseDto) (*ent.Warehouse, error) {
	return nil, nil
}
func (m *mockWarehousesRepoForConsumer) Delete(_ context.Context, _, _ int64) error { return nil }
func (m *mockWarehousesRepoForConsumer) Count(_ context.Context, _ int64) (int32, error) {
	return 0, nil
}

type mockStockItemsRepoForConsumer struct {
	quantities map[string]decimal.Decimal // key: "tenant:warehouse:product"
}

func newMockStockItemsRepo() *mockStockItemsRepoForConsumer {
	return &mockStockItemsRepoForConsumer{
		quantities: make(map[string]decimal.Decimal),
	}
}

func (m *mockStockItemsRepoForConsumer) key(tenantID, warehouseID, productID int64) string {
	return fmt.Sprintf("%d:%d:%d", tenantID, warehouseID, productID)
}

func (m *mockStockItemsRepoForConsumer) DecrementQuantity(_ context.Context, tenantID, warehouseID, productID int64, qty decimal.Decimal) error {
	k := m.key(tenantID, warehouseID, productID)
	m.quantities[k] = m.quantities[k].Sub(qty)
	return nil
}

func (m *mockStockItemsRepoForConsumer) IncrementQuantity(_ context.Context, tenantID, warehouseID, productID int64, qty decimal.Decimal) error {
	k := m.key(tenantID, warehouseID, productID)
	m.quantities[k] = m.quantities[k].Add(qty)
	return nil
}

func (m *mockStockItemsRepoForConsumer) GetByWarehouseAndProduct(_ context.Context, _, _, _ int64) (*ent.StockItem, error) {
	return &ent.StockItem{Quantity: decimal.Zero, MinQuantity: decimal.Zero}, nil
}

// Unused interface methods
func (m *mockStockItemsRepoForConsumer) Upsert(_ context.Context, _ data.StockItemDto) (*ent.StockItem, error) {
	return nil, nil
}
func (m *mockStockItemsRepoForConsumer) ListByWarehouse(_ context.Context, _, _ int64, _ *utils_v1.PaginateRequest) ([]*ent.StockItem, error) {
	return nil, nil
}
func (m *mockStockItemsRepoForConsumer) CountByWarehouse(_ context.Context, _, _ int64) (int32, error) {
	return 0, nil
}
func (m *mockStockItemsRepoForConsumer) ListByProduct(_ context.Context, _, _ int64) ([]*ent.StockItem, error) {
	return nil, nil
}
func (m *mockStockItemsRepoForConsumer) SetMinQuantity(_ context.Context, _, _, _ int64, _ decimal.Decimal) (*ent.StockItem, error) {
	return nil, nil
}
func (m *mockStockItemsRepoForConsumer) ListLowStock(_ context.Context, _, _ int64) ([]*ent.StockItem, error) {
	return nil, nil
}

type mockStockMovementsRepoForConsumer struct{}

func (m *mockStockMovementsRepoForConsumer) CreateReceipt(_ context.Context, _ data.ReceiptDto) ([]*ent.StockMovement, error) {
	return nil, nil
}
func (m *mockStockMovementsRepoForConsumer) CreateTransfer(_ context.Context, _ data.TransferDto) ([]*ent.StockMovement, error) {
	return nil, nil
}
func (m *mockStockMovementsRepoForConsumer) CreateWriteOff(_ context.Context, _ data.WriteOffDto) ([]*ent.StockMovement, error) {
	return nil, nil
}
func (m *mockStockMovementsRepoForConsumer) CreateGift(_ context.Context, _ data.GiftDto) ([]*ent.StockMovement, error) {
	return nil, nil
}
func (m *mockStockMovementsRepoForConsumer) List(_ context.Context, _, _ int64, _ *string, _ *utils_v1.PaginateRequest) ([]*ent.StockMovement, error) {
	return nil, nil
}
func (m *mockStockMovementsRepoForConsumer) Count(_ context.Context, _, _ int64, _ *string) (int32, error) {
	return 0, nil
}

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ data.LowStockEvent) {}

func newTestUsecase(warehouseRepo data.WarehousesRepo, stockItemsRepo data.StockItemsRepo) *WarehouseUsecase {
	return &WarehouseUsecase{
		log:                log.NewHelper(log.DefaultLogger),
		warehouseRepo:      warehouseRepo,
		stockItemsRepo:     stockItemsRepo,
		stockMovementsRepo: &mockStockMovementsRepoForConsumer{},
		publisher:          &mockPublisher{},
	}
}

func TestHandleSaleCompleted(t *testing.T) {
	hallWarehouse := &ent.Warehouse{ID: 10, TenantID: 1, Type: enum.Hall}
	warehouseRepo := &mockWarehousesRepoForConsumer{hall: hallWarehouse}
	stockItems := newMockStockItemsRepo()
	// Set initial stock
	stockItems.quantities[stockItems.key(1, 10, 100)] = decimal.NewFromInt(50)
	stockItems.quantities[stockItems.key(1, 10, 200)] = decimal.NewFromInt(30)

	uc := newTestUsecase(warehouseRepo, stockItems)

	event := messages.SaleCompletedEvent{
		TenantID: 1,
		SaleID:   555,
		Items: []messages.SaleCompletedEventItem{
			{ProductID: 100, Quantity: "3"},
			{ProductID: 200, Quantity: "5"},
		},
	}

	err := uc.HandleSaleCompleted(context.Background(), event)
	require.NoError(t, err)

	// Stock should be decremented
	assert.True(t, stockItems.quantities[stockItems.key(1, 10, 100)].Equal(decimal.NewFromInt(47)))
	assert.True(t, stockItems.quantities[stockItems.key(1, 10, 200)].Equal(decimal.NewFromInt(25)))
}

func TestHandleReturnCompleted(t *testing.T) {
	hallWarehouse := &ent.Warehouse{ID: 10, TenantID: 1, Type: enum.Hall}
	warehouseRepo := &mockWarehousesRepoForConsumer{hall: hallWarehouse}
	stockItems := newMockStockItemsRepo()
	stockItems.quantities[stockItems.key(1, 10, 100)] = decimal.NewFromInt(47)

	uc := newTestUsecase(warehouseRepo, stockItems)

	event := messages.ReturnCompletedEvent{
		TenantID: 1,
		ReturnID: 10,
		Items: []messages.ReturnCompletedEventItem{
			{ProductID: 100, Quantity: "3"},
		},
	}

	err := uc.HandleReturnCompleted(context.Background(), event)
	require.NoError(t, err)

	// Stock should be incremented back
	assert.True(t, stockItems.quantities[stockItems.key(1, 10, 100)].Equal(decimal.NewFromInt(50)))
}

func TestHandleSaleCompleted_NoHallWarehouse(t *testing.T) {
	warehouseRepo := &mockWarehousesRepoForConsumer{hall: nil}
	stockItems := newMockStockItemsRepo()

	uc := newTestUsecase(warehouseRepo, stockItems)

	event := messages.SaleCompletedEvent{
		TenantID: 1,
		SaleID:   1,
		Items: []messages.SaleCompletedEventItem{
			{ProductID: 100, Quantity: "1"},
		},
	}

	err := uc.HandleSaleCompleted(context.Background(), event)
	assert.Error(t, err)
}

func TestHandleOrderConfirmed_DefaultWarehouses(t *testing.T) {
	mainWarehouse := &ent.Warehouse{ID: 5, TenantID: 1, Type: enum.Main}
	hallWarehouse := &ent.Warehouse{ID: 10, TenantID: 1, Type: enum.Hall}
	warehouseRepo := &mockWarehousesRepoForConsumer{hall: hallWarehouse, main: mainWarehouse}
	stockItems := newMockStockItemsRepo()
	// Pre-fill stock in MAIN warehouse so transfer succeeds
	stockItems.quantities[stockItems.key(1, 5, 100)] = decimal.NewFromInt(100)

	uc := newTestUsecase(warehouseRepo, stockItems)

	event := messages.OrderConfirmedEvent{
		TenantID: 1,
		OrderID:  99,
		// No warehouse IDs → defaults to MAIN -> HALL
		Items: []messages.OrderConfirmedEventItem{
			{ProductID: 100, Quantity: "10"},
		},
	}

	err := uc.HandleOrderConfirmed(context.Background(), event)
	// Transfer calls CreateTransfer which uses stockMovementsRepo (mock returns nil, nil)
	// This verifies the path is taken without error
	assert.NoError(t, err)
}

func TestHandleOrderConfirmed_NoItems(t *testing.T) {
	hallWarehouse := &ent.Warehouse{ID: 10, TenantID: 1, Type: enum.Hall}
	warehouseRepo := &mockWarehousesRepoForConsumer{hall: hallWarehouse}
	stockItems := newMockStockItemsRepo()

	uc := newTestUsecase(warehouseRepo, stockItems)

	event := messages.OrderConfirmedEvent{
		TenantID: 1,
		OrderID:  99,
		// No items → should skip
	}

	err := uc.HandleOrderConfirmed(context.Background(), event)
	assert.NoError(t, err)
}
