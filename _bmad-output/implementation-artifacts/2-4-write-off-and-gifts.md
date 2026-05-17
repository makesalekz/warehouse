# Story 2.4: Write-Off and Gifts

## Status: DONE

## Summary

Implemented two new RPCs for the warehouse service:

1. **CreateWriteOff** - Decreases stock via write-off (spoilage, loss, damage)
2. **CreateGift** - Increases stock via gift receipt from a counterparty (price=0)

## Changes

### Proto (`api/warehouse/v1/warehouse.proto`)
- Added `WriteOffItem` message (product_id, quantity, reason)
- Added `CreateWriteOffRequest/Reply` messages
- Added `GiftItem` message (product_id, quantity)
- Added `CreateGiftRequest/Reply` messages (with required counterparty_id)
- Added `CreateWriteOff` and `CreateGift` RPCs to `WarehouseService`

### Data Layer (`internal/data/`)
- `models.go`: Added `WriteOffDto`, `WriteOffItemDto`, `GiftDto`, `GiftItemDto`
- `stock_movements.go`: Added `CreateWriteOff` and `CreateGift` to `StockMovementsRepo` interface + implementations

### Business Logic (`internal/biz/warehouse.go`)
- Added `CreateWriteOff` and `CreateGift` pass-through methods

### Service Layer (`internal/service/warehouse.go`)
- Added `CreateWriteOff` handler with validations:
  - tenant/actor required
  - warehouse_id required
  - items non-empty
  - quantity positive
  - reason non-empty (trimmed whitespace)
  - maps insufficient stock errors to `ErrorInsufficientStock`
- Added `CreateGift` handler with validations:
  - tenant/actor required
  - warehouse_id required
  - counterparty_id required (non-zero)
  - items non-empty
  - quantity positive

### Tests (`internal/service/warehouse_test.go`)
Added 25 new tests:
- WriteOff: happy path, multiple items, insufficient stock, no stock, atomic rollback, write-off all, empty reason, whitespace reason, no tenant, no actor, empty items, empty warehouse, invalid quantity, movements visible in list
- Gift: happy path, multiple items, adds to existing stock, no counterparty, no tenant, no actor, empty items, empty warehouse, invalid quantity, movements visible in list, tenant isolation

## Design Decisions

- **Reason required on WriteOff**: Every write-off must have a non-empty reason (audit trail)
- **counterparty_id required on Gift**: A gift without a giver is incoherent
- **WriteOff stores negative quantity** in StockMovement (consistent with Transfer outgoing)
- **Gift stores positive quantity** in StockMovement (consistent with Receipt)
- **Atomic transactions**: Both operations validate all items before applying any changes; rollback on failure

## Test Count
- Before: 51 tests
- After: 78 tests (26 new for WriteOff/Gift + mock updates for parallel stories)

## Out-of-Scope Fixes (required for build to pass)

Parallel stories (Inventory, Low Stock Alerts) introduced interface changes without complete implementations. Fixed to unblock this story:

- `internal/data/stock_items.go`: Added `SetMinQuantity`, `ListLowStock`, `GetByWarehouseAndProduct` method implementations (interface declared by parallel story). Fixed `ListLowStock` type error (`sql.ColumnsLTE` instead of passing column name as decimal value).
- `internal/biz/warehouse.go`: Removed unused `decimal` import (leftover from linter).
- `internal/service/warehouse_test.go`: Added `mockInventoriesRepo`, `mockLowStockPublisher`, and `mockStockItemsRepo` methods (`SetMinQuantity`, `ListLowStock`, `GetByWarehouseAndProduct`) to satisfy updated interfaces.
- `cmd/app/wire_gen.go`: Regenerated via `make generate` (was stale after parallel story added `InventoriesRepo` + `LowStockPublisher`).
