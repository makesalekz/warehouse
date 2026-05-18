package service

import (
	"context"
	"strings"

	v1 "github.com/makesalekz/warehouse/api/warehouse/v1"
	"github.com/makesalekz/warehouse/ent"
	"github.com/makesalekz/warehouse/ent/enum"
	"github.com/makesalekz/warehouse/internal/biz"
	"github.com/makesalekz/warehouse/internal/data"
	utils_v1 "github.com/makesalekz/utils/api/utils/v1"
	"github.com/makesalekz/utils/v2/auth"

	"github.com/shopspring/decimal"
)

type WarehouseService struct {
	v1.UnimplementedWarehouseServiceServer

	uc *biz.WarehouseUsecase
}

func NewWarehouseService(uc *biz.WarehouseUsecase) *WarehouseService {
	return &WarehouseService{uc: uc}
}

func (s *WarehouseService) CreateWarehouse(ctx context.Context, req *v1.CreateWarehouseRequest) (*v1.CreateWarehouseReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	wType := enum.WarehouseType(req.GetType())
	if !wType.IsValid() {
		wType = enum.Main
	}

	dto := data.WarehouseDto{
		TenantID:    tenantID,
		Name:        req.GetName(),
		Type:        wType,
		Description: req.GetDescription(),
		Address:     req.GetAddress(),
		IsActive:    true,
	}
	if req.StoreId != nil {
		sid := *req.StoreId
		dto.StoreID = &sid
	}

	w, err := s.uc.Create(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &v1.CreateWarehouseReply{Warehouse: replyWarehouse(w)}, nil
}

func (s *WarehouseService) GetWarehouse(ctx context.Context, req *v1.GetWarehouseRequest) (*v1.GetWarehouseReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	w, err := s.uc.Get(ctx, tenantID, req.GetId())
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, v1.ErrorNotFound("warehouse not found")
		}
		return nil, err
	}

	return &v1.GetWarehouseReply{Warehouse: replyWarehouse(w)}, nil
}

func (s *WarehouseService) ListWarehouses(ctx context.Context, req *v1.ListWarehousesRequest) (*v1.ListWarehousesReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	paginate := req.GetPaginate()
	if paginate == nil {
		paginate = &utils_v1.PaginateRequest{}
	}

	var storeID *int64
	if req.StoreId != nil {
		sid := *req.StoreId
		storeID = &sid
	}

	items, total, err := s.uc.List(ctx, tenantID, storeID, paginate)
	if err != nil {
		return nil, err
	}

	warehouses := make([]*v1.Warehouse, 0, len(items))
	for _, item := range items {
		warehouses = append(warehouses, replyWarehouse(item))
	}

	var fromID, toID *int64
	if len(items) > 0 {
		f := items[0].ID
		t := items[len(items)-1].ID
		fromID = &f
		toID = &t
	}

	return &v1.ListWarehousesReply{
		Items: warehouses,
		Paginate: &utils_v1.PaginateReply{
			Total:  &total,
			FromId: fromID,
			ToId:   toID,
		},
	}, nil
}

func (s *WarehouseService) UpdateWarehouse(ctx context.Context, req *v1.UpdateWarehouseRequest) (*v1.UpdateWarehouseReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	wType := enum.WarehouseType(req.GetType())
	if !wType.IsValid() {
		wType = enum.Main
	}

	dto := data.WarehouseDto{
		ID:          req.GetId(),
		TenantID:    tenantID,
		Name:        req.GetName(),
		Type:        wType,
		Description: req.GetDescription(),
		Address:     req.GetAddress(),
		IsActive:    req.GetIsActive(),
	}
	if req.StoreId != nil {
		sid := *req.StoreId
		dto.StoreID = &sid
	}

	w, err := s.uc.Update(ctx, dto)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, v1.ErrorNotFound("warehouse not found")
		}
		return nil, err
	}

	return &v1.UpdateWarehouseReply{Warehouse: replyWarehouse(w)}, nil
}

func (s *WarehouseService) DeleteWarehouse(ctx context.Context, req *v1.DeleteWarehouseRequest) (*v1.DeleteWarehouseReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	err := s.uc.Delete(ctx, tenantID, req.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.DeleteWarehouseReply{}, nil
}

func (s *WarehouseService) GetStockItems(ctx context.Context, req *v1.GetStockItemsRequest) (*v1.GetStockItemsReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	paginate := req.GetPaginate()
	if paginate == nil {
		paginate = &utils_v1.PaginateRequest{}
	}

	items, total, err := s.uc.GetStockItems(ctx, tenantID, req.GetWarehouseId(), paginate)
	if err != nil {
		return nil, err
	}

	stockItems := make([]*v1.StockItem, 0, len(items))
	for _, item := range items {
		stockItems = append(stockItems, replyStockItem(item))
	}

	var fromID, toID *int64
	if len(items) > 0 {
		f := items[0].ID
		t := items[len(items)-1].ID
		fromID = &f
		toID = &t
	}

	return &v1.GetStockItemsReply{
		Items: stockItems,
		Paginate: &utils_v1.PaginateReply{
			Total:  &total,
			FromId: fromID,
			ToId:   toID,
		},
	}, nil
}

func (s *WarehouseService) GetStockByProduct(ctx context.Context, req *v1.GetStockByProductRequest) (*v1.GetStockByProductReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	items, err := s.uc.GetStockByProduct(ctx, tenantID, req.GetProductId())
	if err != nil {
		return nil, err
	}

	stockItems := make([]*v1.StockItem, 0, len(items))
	for _, item := range items {
		stockItems = append(stockItems, replyStockItem(item))
	}

	return &v1.GetStockByProductReply{Items: stockItems}, nil
}

func (s *WarehouseService) CreateReceipt(ctx context.Context, req *v1.CreateReceiptRequest) (*v1.CreateReceiptReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorInvalidRequest("empty actor id")
	}

	if req.GetWarehouseId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty warehouse id")
	}

	if len(req.GetItems()) == 0 {
		return nil, v1.ErrorInvalidRequest("empty items")
	}

	items := make([]data.ReceiptItemDto, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		qty, err := decimal.NewFromString(item.GetQuantity())
		if err != nil || !qty.IsPositive() {
			return nil, v1.ErrorInvalidRequest("invalid quantity")
		}
		items = append(items, data.ReceiptItemDto{
			ProductID: item.GetProductId(),
			Quantity:  qty,
		})
	}

	var counterpartyID *int64
	if req.CounterpartyId != nil {
		cid := req.GetCounterpartyId()
		counterpartyID = &cid
	}

	dto := data.ReceiptDto{
		TenantID:       tenantID,
		WarehouseID:    req.GetWarehouseId(),
		CounterpartyID: counterpartyID,
		CreatedBy:      actorID,
		Items:          items,
	}

	movements, err := s.uc.CreateReceipt(ctx, dto)
	if err != nil {
		return nil, err
	}

	result := make([]*v1.StockMovement, 0, len(movements))
	for _, m := range movements {
		result = append(result, replyStockMovement(m))
	}

	return &v1.CreateReceiptReply{Movements: result}, nil
}

func (s *WarehouseService) CreateTransfer(ctx context.Context, req *v1.CreateTransferRequest) (*v1.CreateTransferReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorInvalidRequest("empty actor id")
	}

	if req.GetSourceWarehouseId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty source warehouse id")
	}

	if req.GetTargetWarehouseId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty target warehouse id")
	}

	if req.GetSourceWarehouseId() == req.GetTargetWarehouseId() {
		return nil, v1.ErrorInvalidRequest("source and target warehouse must be different")
	}

	if len(req.GetItems()) == 0 {
		return nil, v1.ErrorInvalidRequest("empty items")
	}

	items := make([]data.TransferItemDto, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		qty, err := decimal.NewFromString(item.GetQuantity())
		if err != nil || !qty.IsPositive() {
			return nil, v1.ErrorInvalidRequest("invalid quantity")
		}
		items = append(items, data.TransferItemDto{
			ProductID: item.GetProductId(),
			Quantity:  qty,
		})
	}

	dto := data.TransferDto{
		TenantID:          tenantID,
		SourceWarehouseID: req.GetSourceWarehouseId(),
		TargetWarehouseID: req.GetTargetWarehouseId(),
		CreatedBy:         actorID,
		Items:             items,
	}

	movements, err := s.uc.CreateTransfer(ctx, dto)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "warehouse not found") {
			return nil, v1.ErrorNotFound("%s", errMsg)
		}
		if strings.Contains(errMsg, "insufficient stock") {
			return nil, v1.ErrorInsufficientStock("%s", errMsg)
		}
		return nil, err
	}

	result := make([]*v1.StockMovement, 0, len(movements))
	for _, m := range movements {
		result = append(result, replyStockMovement(m))
	}

	return &v1.CreateTransferReply{Movements: result}, nil
}

func (s *WarehouseService) CreateWriteOff(ctx context.Context, req *v1.CreateWriteOffRequest) (*v1.CreateWriteOffReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorInvalidRequest("empty actor id")
	}

	if req.GetWarehouseId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty warehouse id")
	}

	if len(req.GetItems()) == 0 {
		return nil, v1.ErrorInvalidRequest("empty items")
	}

	items := make([]data.WriteOffItemDto, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		qty, err := decimal.NewFromString(item.GetQuantity())
		if err != nil || !qty.IsPositive() {
			return nil, v1.ErrorInvalidRequest("invalid quantity")
		}
		reason := strings.TrimSpace(item.GetReason())
		if reason == "" {
			return nil, v1.ErrorInvalidRequest("empty reason")
		}
		items = append(items, data.WriteOffItemDto{
			ProductID: item.GetProductId(),
			Quantity:  qty,
			Reason:    reason,
		})
	}

	dto := data.WriteOffDto{
		TenantID:    tenantID,
		WarehouseID: req.GetWarehouseId(),
		CreatedBy:   actorID,
		Items:       items,
	}

	movements, err := s.uc.CreateWriteOff(ctx, dto)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "insufficient stock") {
			return nil, v1.ErrorInsufficientStock("%s", errMsg)
		}
		return nil, err
	}

	result := make([]*v1.StockMovement, 0, len(movements))
	for _, m := range movements {
		result = append(result, replyStockMovement(m))
	}

	return &v1.CreateWriteOffReply{Movements: result}, nil
}

func (s *WarehouseService) CreateGift(ctx context.Context, req *v1.CreateGiftRequest) (*v1.CreateGiftReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorInvalidRequest("empty actor id")
	}

	if req.GetWarehouseId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty warehouse id")
	}

	if req.GetCounterpartyId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty counterparty id")
	}

	if len(req.GetItems()) == 0 {
		return nil, v1.ErrorInvalidRequest("empty items")
	}

	items := make([]data.GiftItemDto, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		qty, err := decimal.NewFromString(item.GetQuantity())
		if err != nil || !qty.IsPositive() {
			return nil, v1.ErrorInvalidRequest("invalid quantity")
		}
		items = append(items, data.GiftItemDto{
			ProductID: item.GetProductId(),
			Quantity:  qty,
		})
	}

	dto := data.GiftDto{
		TenantID:       tenantID,
		WarehouseID:    req.GetWarehouseId(),
		CounterpartyID: req.GetCounterpartyId(),
		CreatedBy:      actorID,
		Items:          items,
	}

	movements, err := s.uc.CreateGift(ctx, dto)
	if err != nil {
		return nil, err
	}

	result := make([]*v1.StockMovement, 0, len(movements))
	for _, m := range movements {
		result = append(result, replyStockMovement(m))
	}

	return &v1.CreateGiftReply{Movements: result}, nil
}

func (s *WarehouseService) ListMovements(ctx context.Context, req *v1.ListMovementsRequest) (*v1.ListMovementsReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	paginate := req.GetPaginate()
	if paginate == nil {
		paginate = &utils_v1.PaginateRequest{}
	}

	var movementType *string
	if req.Type != nil {
		t := req.GetType()
		movementType = &t
	}

	items, total, err := s.uc.ListMovements(ctx, tenantID, req.GetWarehouseId(), movementType, paginate)
	if err != nil {
		return nil, err
	}

	movements := make([]*v1.StockMovement, 0, len(items))
	for _, item := range items {
		movements = append(movements, replyStockMovement(item))
	}

	var fromID, toID *int64
	if len(items) > 0 {
		f := items[0].ID
		t := items[len(items)-1].ID
		fromID = &f
		toID = &t
	}

	return &v1.ListMovementsReply{
		Items: movements,
		Paginate: &utils_v1.PaginateReply{
			Total:  &total,
			FromId: fromID,
			ToId:   toID,
		},
	}, nil
}

func (s *WarehouseService) SetMinQuantity(ctx context.Context, req *v1.SetMinQuantityRequest) (*v1.SetMinQuantityReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	if req.GetWarehouseId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty warehouse id")
	}

	if req.GetProductId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty product id")
	}

	minQty, err := decimal.NewFromString(req.GetMinQuantity())
	if err != nil || minQty.IsNegative() {
		return nil, v1.ErrorInvalidRequest("invalid min_quantity")
	}

	item, err := s.uc.SetMinQuantity(ctx, tenantID, req.GetWarehouseId(), req.GetProductId(), minQty)
	if err != nil {
		return nil, err
	}

	return &v1.SetMinQuantityReply{Item: replyStockItem(item)}, nil
}

func (s *WarehouseService) GetLowStockItems(ctx context.Context, req *v1.GetLowStockItemsRequest) (*v1.GetLowStockItemsReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	if req.GetWarehouseId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty warehouse id")
	}

	items, err := s.uc.GetLowStockItems(ctx, tenantID, req.GetWarehouseId())
	if err != nil {
		return nil, err
	}

	stockItems := make([]*v1.StockItem, 0, len(items))
	for _, item := range items {
		stockItems = append(stockItems, replyStockItem(item))
	}

	return &v1.GetLowStockItemsReply{Items: stockItems}, nil
}

func (s *WarehouseService) StartInventory(ctx context.Context, req *v1.StartInventoryRequest) (*v1.StartInventoryReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorInvalidRequest("empty actor id")
	}

	if req.GetWarehouseId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty warehouse id")
	}

	dto := data.StartInventoryDto{
		TenantID:    tenantID,
		WarehouseID: req.GetWarehouseId(),
		CreatedBy:   actorID,
	}

	inv, items, err := s.uc.StartInventory(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &v1.StartInventoryReply{Inventory: replyInventory(inv, items)}, nil
}

func (s *WarehouseService) SetInventoryItem(ctx context.Context, req *v1.SetInventoryItemRequest) (*v1.SetInventoryItemReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	if req.GetInventoryId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty inventory id")
	}

	if req.GetProductId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty product id")
	}

	actualQty, err := decimal.NewFromString(req.GetActualQty())
	if err != nil || actualQty.IsNegative() {
		return nil, v1.ErrorInvalidRequest("invalid actual_qty")
	}

	dto := data.SetInventoryItemDto{
		TenantID:    tenantID,
		InventoryID: req.GetInventoryId(),
		ProductID:   req.GetProductId(),
		ActualQty:   actualQty,
	}

	item, err := s.uc.SetInventoryItem(ctx, dto)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return nil, v1.ErrorNotFound("%s", errMsg)
		}
		if strings.Contains(errMsg, "already completed") {
			return nil, v1.ErrorInvalidRequest("%s", errMsg)
		}
		return nil, err
	}

	return &v1.SetInventoryItemReply{Item: replyInventoryItem(item)}, nil
}

func (s *WarehouseService) CompleteInventory(ctx context.Context, req *v1.CompleteInventoryRequest) (*v1.CompleteInventoryReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	if req.GetInventoryId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty inventory id")
	}

	dto := data.CompleteInventoryDto{
		TenantID:    tenantID,
		InventoryID: req.GetInventoryId(),
	}

	inv, items, err := s.uc.CompleteInventory(ctx, dto)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return nil, v1.ErrorNotFound("%s", errMsg)
		}
		if strings.Contains(errMsg, "already completed") {
			return nil, v1.ErrorInvalidRequest("%s", errMsg)
		}
		return nil, err
	}

	return &v1.CompleteInventoryReply{Inventory: replyInventory(inv, items)}, nil
}

func (s *WarehouseService) GetInventory(ctx context.Context, req *v1.GetInventoryRequest) (*v1.GetInventoryReply, error) {
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorInvalidRequest("empty tenant id")
	}

	if req.GetInventoryId() == 0 {
		return nil, v1.ErrorInvalidRequest("empty inventory id")
	}

	inv, items, err := s.uc.GetInventory(ctx, tenantID, req.GetInventoryId())
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return nil, v1.ErrorNotFound("%s", errMsg)
		}
		return nil, err
	}

	return &v1.GetInventoryReply{Inventory: replyInventory(inv, items)}, nil
}

func replyInventory(inv *ent.Inventory, items []*ent.InventoryItem) *v1.Inventory {
	result := &v1.Inventory{
		Id:          inv.ID,
		TenantId:    inv.TenantID,
		WarehouseId: inv.WarehouseID,
		Status:      string(inv.Status),
		CreatedBy:   inv.CreatedBy,
		CreatedAt:   inv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if inv.CompletedAt != nil {
		t := inv.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		result.CompletedAt = &t
	}
	result.Items = make([]*v1.InventoryItem, 0, len(items))
	for _, item := range items {
		result.Items = append(result.Items, replyInventoryItem(item))
	}
	return result
}

func replyInventoryItem(item *ent.InventoryItem) *v1.InventoryItem {
	result := &v1.InventoryItem{
		Id:          item.ID,
		InventoryId: item.InventoryID,
		ProductId:   item.ProductID,
		ExpectedQty: item.ExpectedQty.String(),
	}
	if item.ActualQty != nil {
		s := item.ActualQty.String()
		result.ActualQty = &s
	}
	if item.Difference != nil {
		s := item.Difference.String()
		result.Difference = &s
	}
	return result
}

func replyStockMovement(m *ent.StockMovement) *v1.StockMovement {
	sm := &v1.StockMovement{
		Id:          m.ID,
		TenantId:    m.TenantID,
		WarehouseId: m.WarehouseID,
		Type:        string(m.Type),
		ProductId:   m.ProductID,
		Quantity:    m.Quantity.String(),
		CreatedBy:   m.CreatedBy,
		CreatedAt:   m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if m.CounterpartyID != nil {
		sm.CounterpartyId = m.CounterpartyID
	}
	if m.Reason != nil {
		sm.Reason = m.Reason
	}
	return sm
}

func replyWarehouse(w *ent.Warehouse) *v1.Warehouse {
	result := &v1.Warehouse{
		Id:          w.ID,
		TenantId:    w.TenantID,
		Name:        w.Name,
		Type:        string(w.Type),
		Description: w.Description,
		Address:     w.Address,
		IsActive:    w.IsActive,
		CreatedAt:   w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   w.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if w.StoreID != nil {
		result.StoreId = w.StoreID
	}
	return result
}

func replyStockItem(s *ent.StockItem) *v1.StockItem {
	return &v1.StockItem{
		Id:          s.ID,
		TenantId:    s.TenantID,
		WarehouseId: s.WarehouseID,
		ProductId:   s.ProductID,
		Quantity:    s.Quantity.String(),
		MinQuantity: s.MinQuantity.String(),
		UpdatedAt:   s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
