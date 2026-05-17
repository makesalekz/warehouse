# Story 2.6: Critical Stock Level and Notifications

## Status: DONE

## Summary

Implemented critical stock level monitoring with NATS event publishing for the warehouse service.

## What was implemented

### New RPCs

1. **SetMinQuantity(warehouse_id, product_id, min_quantity)** - Sets the critical stock threshold for a product in a warehouse. Creates the stock item if it doesn't exist (with qty=0). Accepts zero to disable threshold.

2. **GetLowStockItems(warehouse_id)** - Returns all stock items where `quantity <= min_quantity` AND `min_quantity > 0`. Uses SQL column comparison for efficiency.

### Low Stock Event Publishing

After any stock-decreasing operation (CreateTransfer, CreateWriteOff), the system checks affected products and publishes a NATS event if quantity has dropped to or below min_quantity.

- **Subject:** `warehouse.stock.low`
- **Payload:** `{"tenant_id", "warehouse_id", "product_id", "quantity", "min_quantity"}` (quantities as strings)
- **Behavior:** State-check (fires on every operation that results in low state, not edge-triggered)
- **Guard:** Only fires when `min_quantity > 0` (unset thresholds don't trigger)
- **Failure mode:** Publish errors are logged, never fail the operation

### NATS Connection

- Added `nats` field to Bootstrap config proto
- Simple `*nats.Conn` following billing service pattern
- Optional: if `nats` config is empty, publisher is a no-op

## Design Decisions

- **State-check vs edge-triggered:** Chose state-check (publish whenever `qty <= min_qty` after operation). Simpler, no pre-state tracking needed. May be noisy for repeated decreases while already-low.
- **LowStockPublisher interface in data package:** Allows clean mocking in tests without NATS dependency.
- **No JetStream:** Uses simple `nc.Publish` (fire-and-forget) matching billing pattern. Consumer services can use JetStream if they need guaranteed delivery.

## Files Changed

- `api/warehouse/v1/warehouse.proto` - Added SetMinQuantity, GetLowStockItems RPCs + messages
- `internal/conf/conf.proto` - Added nats field to Bootstrap
- `internal/data/nats.go` - NEW: NATS client constructor
- `internal/data/publisher.go` - NEW: LowStockPublisher interface + NATS implementation
- `internal/data/stock_items.go` - Added SetMinQuantity, ListLowStock, GetByWarehouseAndProduct methods
- `internal/data/data.go` - Added NewNatsClient, NewLowStockPublisher to ProviderSet
- `internal/biz/warehouse.go` - Added publisher dep, checkLowStock helper, new use cases
- `internal/service/warehouse.go` - Added SetMinQuantity, GetLowStockItems handlers
- `internal/service/warehouse_test.go` - 16 new tests (mock publisher captures events)
- `cmd/app/wire_gen.go` - Regenerated with NATS + publisher wiring
- `go.mod` / `go.sum` - Added github.com/nats-io/nats.go dependency

## Tests Added (16)

- TestSetMinQuantity_HappyPath
- TestSetMinQuantity_ZeroValue
- TestSetMinQuantity_NoTenant
- TestSetMinQuantity_EmptyWarehouseId
- TestSetMinQuantity_EmptyProductId
- TestSetMinQuantity_NegativeValue
- TestSetMinQuantity_InvalidValue
- TestGetLowStockItems_ReturnsOnlyLow
- TestGetLowStockItems_NoTenant
- TestGetLowStockItems_EmptyWarehouseId
- TestGetLowStockItems_TenantIsolation
- TestTransfer_PublishesLowStockEvent
- TestTransfer_NoEventWhenAboveThreshold
- TestTransfer_NoEventWhenMinQuantityZero
- TestTransfer_MultipleItems_OnlyLowOneFires
- TestWriteOff_PublishesLowStockEvent
