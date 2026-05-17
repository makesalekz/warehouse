package messages

// SaleCompletedEventItem mirrors the sales service event item structure.
type SaleCompletedEventItem struct {
	ProductID   int64  `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    string `json:"quantity"`
	UnitPrice   string `json:"unit_price"`
	Discount    string `json:"discount"`
	Total       string `json:"total"`
}

// SaleCompletedEvent is the payload of "sales.sale.completed" NATS subject.
type SaleCompletedEvent struct {
	TenantID  int64                    `json:"tenant_id"`
	SaleID    int64                    `json:"sale_id"`
	Total     string                   `json:"total"`
	Items     []SaleCompletedEventItem `json:"items"`
	Timestamp string                   `json:"timestamp"`
}

// ReturnCompletedEventItem mirrors the sales service return event item.
type ReturnCompletedEventItem struct {
	ProductID  int64  `json:"product_id"`
	SaleItemID int64  `json:"sale_item_id"`
	Quantity   string `json:"quantity"`
	UnitPrice  string `json:"unit_price"`
	Total      string `json:"total"`
}

// ReturnCompletedEvent is the payload of "sales.return.completed" NATS subject.
type ReturnCompletedEvent struct {
	TenantID  int64                      `json:"tenant_id"`
	SaleID    int64                      `json:"sale_id"`
	ReturnID  int64                      `json:"return_id"`
	Total     string                     `json:"total"`
	Items     []ReturnCompletedEventItem `json:"items"`
	Timestamp string                     `json:"timestamp"`
}

// OrderConfirmedEventItem represents an item in the order confirmed event.
type OrderConfirmedEventItem struct {
	ProductID int64  `json:"product_id"`
	Quantity  string `json:"quantity"`
}

// OrderConfirmedEvent is the payload of "orders.order.confirmed" NATS subject.
// NOTE: The current orders publisher only sends {tenant_id, order_id, counterparty_id}.
// For Story 10.3, the orders service must be extended to include items and warehouse IDs.
// Until then, the consumer fetches order details via gRPC fallback or uses enriched events.
type OrderConfirmedEvent struct {
	TenantID          int64                     `json:"tenant_id"`
	OrderID           int64                     `json:"order_id"`
	Items             []OrderConfirmedEventItem `json:"items"`
	SourceWarehouseID int64                     `json:"source_warehouse_id"`
	TargetWarehouseID int64                     `json:"target_warehouse_id"`
}
