package biz

import (
	"context"
	"encoding/json"

	"github.com/makesalekz/warehouse/internal/data"
	"github.com/makesalekz/warehouse/messages"

	"github.com/go-kratos/kratos/v2/log"
	nats "github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
)

// Consumer subscribes to NATS events for stock adjustments.
type Consumer struct {
	log  *log.Helper
	uc   *WarehouseUsecase
	subs []*nats.Subscription
}

// NewConsumer creates subscriptions for sale, return, and order events.
// Returns a cleanup function that unsubscribes.
func NewConsumer(nc *nats.Conn, uc *WarehouseUsecase, logger log.Logger) (*Consumer, func(), error) {
	l := log.NewHelper(logger)
	c := &Consumer{log: l, uc: uc}

	if nc == nil {
		l.Info("NATS not configured, skipping warehouse consumer subscriptions")
		return c, func() {}, nil
	}

	saleSub, err := nc.Subscribe("sales.sale.completed", c.handleSaleCompleted)
	if err != nil {
		return nil, nil, err
	}

	returnSub, err := nc.Subscribe("sales.return.completed", c.handleReturnCompleted)
	if err != nil {
		saleSub.Unsubscribe()
		return nil, nil, err
	}

	orderSub, err := nc.Subscribe("orders.order.confirmed", c.handleOrderConfirmed)
	if err != nil {
		saleSub.Unsubscribe()
		returnSub.Unsubscribe()
		return nil, nil, err
	}

	c.subs = []*nats.Subscription{saleSub, returnSub, orderSub}
	l.Info("Subscribed to sales.sale.completed, sales.return.completed, orders.order.confirmed")

	cleanup := func() {
		for _, sub := range c.subs {
			sub.Unsubscribe()
		}
	}

	return c, cleanup, nil
}

// handleSaleCompleted decreases stock for each sold item in the tenant's HALL warehouse.
func (c *Consumer) handleSaleCompleted(msg *nats.Msg) {
	var event messages.SaleCompletedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		c.log.Errorf("failed to unmarshal sale completed event: %v", err)
		return
	}

	if err := c.uc.HandleSaleCompleted(context.Background(), event); err != nil {
		c.log.Errorf("failed to handle sale completed for sale %d: %v", event.SaleID, err)
	}
}

// handleReturnCompleted increases stock for each returned item.
func (c *Consumer) handleReturnCompleted(msg *nats.Msg) {
	var event messages.ReturnCompletedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		c.log.Errorf("failed to unmarshal return completed event: %v", err)
		return
	}

	if err := c.uc.HandleReturnCompleted(context.Background(), event); err != nil {
		c.log.Errorf("failed to handle return completed for return %d: %v", event.ReturnID, err)
	}
}

// handleOrderConfirmed creates a TRANSFER movement from the order details.
func (c *Consumer) handleOrderConfirmed(msg *nats.Msg) {
	var event messages.OrderConfirmedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		c.log.Errorf("failed to unmarshal order confirmed event: %v", err)
		return
	}

	if err := c.uc.HandleOrderConfirmed(context.Background(), event); err != nil {
		c.log.Errorf("failed to handle order confirmed for order %d: %v", event.OrderID, err)
	}
}

// HandleSaleCompleted processes a sale completed event by decreasing stock.
// Uses the first active HALL warehouse for the tenant.
func (uc *WarehouseUsecase) HandleSaleCompleted(ctx context.Context, event messages.SaleCompletedEvent) error {
	// Find the HALL warehouse for this tenant
	hallWarehouse, err := uc.warehouseRepo.FindHallByTenant(ctx, event.TenantID)
	if err != nil {
		return err
	}

	for _, item := range event.Items {
		qty, err := decimal.NewFromString(item.Quantity)
		if err != nil {
			uc.log.Errorf("invalid quantity for product %d: %v", item.ProductID, err)
			continue
		}

		// Decrease stock (allow negative — don't block sales)
		if err := uc.stockItemsRepo.DecrementQuantity(ctx, event.TenantID, hallWarehouse.ID, item.ProductID, qty); err != nil {
			uc.log.Errorf("failed to decrement stock for product %d: %v", item.ProductID, err)
			continue
		}
	}

	// Check low stock for affected products
	productIDs := make([]int64, len(event.Items))
	for i, item := range event.Items {
		productIDs[i] = item.ProductID
	}
	uc.checkLowStock(ctx, event.TenantID, hallWarehouse.ID, productIDs)

	return nil
}

// HandleReturnCompleted processes a return completed event by increasing stock.
func (uc *WarehouseUsecase) HandleReturnCompleted(ctx context.Context, event messages.ReturnCompletedEvent) error {
	// Find the HALL warehouse for this tenant
	hallWarehouse, err := uc.warehouseRepo.FindHallByTenant(ctx, event.TenantID)
	if err != nil {
		return err
	}

	for _, item := range event.Items {
		qty, err := decimal.NewFromString(item.Quantity)
		if err != nil {
			uc.log.Errorf("invalid quantity for product %d: %v", item.ProductID, err)
			continue
		}

		// Increase stock back
		if err := uc.stockItemsRepo.IncrementQuantity(ctx, event.TenantID, hallWarehouse.ID, item.ProductID, qty); err != nil {
			uc.log.Errorf("failed to increment stock for product %d: %v", item.ProductID, err)
			continue
		}
	}

	return nil
}

// HandleOrderConfirmed creates a transfer movement based on order confirmation.
// If warehouse IDs are not in the event, defaults to MAIN -> HALL for the tenant.
func (uc *WarehouseUsecase) HandleOrderConfirmed(ctx context.Context, event messages.OrderConfirmedEvent) error {
	if len(event.Items) == 0 {
		uc.log.Warnf("order %d has no items, skipping transfer", event.OrderID)
		return nil
	}

	sourceID := event.SourceWarehouseID
	targetID := event.TargetWarehouseID

	// Default: MAIN -> HALL within same tenant
	if sourceID == 0 {
		mainWh, err := uc.warehouseRepo.FindMainByTenant(ctx, event.TenantID)
		if err != nil {
			return err
		}
		sourceID = mainWh.ID
	}
	if targetID == 0 {
		hallWh, err := uc.warehouseRepo.FindHallByTenant(ctx, event.TenantID)
		if err != nil {
			return err
		}
		targetID = hallWh.ID
	}

	items := make([]data.TransferItemDto, 0, len(event.Items))
	for _, item := range event.Items {
		qty, err := decimal.NewFromString(item.Quantity)
		if err != nil {
			uc.log.Errorf("invalid quantity for product %d in order %d: %v", item.ProductID, event.OrderID, err)
			continue
		}
		items = append(items, data.TransferItemDto{
			ProductID: item.ProductID,
			Quantity:  qty,
		})
	}

	if len(items) == 0 {
		return nil
	}

	dto := data.TransferDto{
		TenantID:          event.TenantID,
		SourceWarehouseID: sourceID,
		TargetWarehouseID: targetID,
		CreatedBy:         0, // system-initiated
		Items:             items,
	}

	_, err := uc.CreateTransfer(ctx, dto)
	return err
}
