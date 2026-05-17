package data

import (
	"encoding/json"

	"github.com/go-kratos/kratos/v2/log"
	nats "github.com/nats-io/nats.go"
)

const lowStockSubject = "warehouse.stock.low"

type LowStockEvent struct {
	TenantID    int64  `json:"tenant_id"`
	WarehouseID int64  `json:"warehouse_id"`
	ProductID   int64  `json:"product_id"`
	Quantity    string `json:"quantity"`
	MinQuantity string `json:"min_quantity"`
}

type LowStockPublisher interface {
	Publish(event LowStockEvent)
}

type natsLowStockPublisher struct {
	nc  *nats.Conn
	log *log.Helper
}

func NewLowStockPublisher(nc *nats.Conn, logger log.Logger) LowStockPublisher {
	return &natsLowStockPublisher{
		nc:  nc,
		log: log.NewHelper(logger),
	}
}

func (p *natsLowStockPublisher) Publish(event LowStockEvent) {
	if p.nc == nil {
		return
	}

	payload, err := json.Marshal(event)
	if err != nil {
		p.log.Errorf("failed to marshal low stock event: %v", err)
		return
	}

	if err := p.nc.Publish(lowStockSubject, payload); err != nil {
		p.log.Errorf("failed to publish low stock event: %v", err)
	}
}
