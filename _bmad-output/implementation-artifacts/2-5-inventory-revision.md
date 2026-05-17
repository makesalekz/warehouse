# Story 2.5: Inventory Revision (Ревизия / инвентаризация)

## Status: COMPLETE

## What was implemented

### Ent Schemas
- `ent/schema/inventory.go` — Inventory entity (id, tenant_id, warehouse_id FK, status enum DRAFT/IN_PROGRESS/COMPLETED, created_by, created_at, completed_at)
- `ent/schema/inventory_item.go` — InventoryItem entity (id, inventory_id FK, product_id, expected_qty, actual_qty nullable, difference nullable)
- `ent/enum/inventory_status.go` — InventoryStatus enum (DRAFT, IN_PROGRESS, COMPLETED)
- Added `inventories` edge to Warehouse schema

### Proto (api/warehouse/v1/)
- Added `Inventory` and `InventoryItem` messages to `models.proto`
- Added 4 RPCs to `warehouse.proto`: StartInventory, SetInventoryItem, CompleteInventory, GetInventory

### Data Layer
- `internal/data/inventories.go` — InventoriesRepo interface + implementation
- DTOs in `internal/data/models.go`: StartInventoryDto, SetInventoryItemDto, CompleteInventoryDto
- Added `NewInventoriesRepo` to data ProviderSet

### Biz Layer
- Added `inventoriesRepo` field to WarehouseUsecase
- Methods: StartInventory, SetInventoryItem, CompleteInventory, GetInventory

### Service Layer
- 4 gRPC handlers with validation (tenant, actor, IDs, quantities)
- Helper functions: replyInventory, replyInventoryItem

### Tests (23 new tests)
- StartInventory: happy path, empty warehouse, no tenant, no actor, empty warehouse_id, snapshots correct quantities
- SetInventoryItem: happy path, surplus, new product (not in snapshot), zero qty, invalid qty, no tenant, empty inventory_id, empty product_id, completed inventory rejection, not found
- CompleteInventory: happy path, does NOT adjust stock, already completed rejection, not found, no tenant, empty inventory_id
- GetInventory: happy path, not found, no tenant, empty inventory_id, tenant isolation
- Full flow integration test

## Design Decisions
- StartInventory creates inventory in IN_PROGRESS status (skips DRAFT — snapshot exists immediately)
- SetInventoryItem for a product NOT in the snapshot creates a new InventoryItem with expected_qty=0
- SetInventoryItem rejects if inventory is COMPLETED
- CompleteInventory rejects if already COMPLETED
- Completing inventory does NOT auto-adjust stock (report-only, per requirements)
- Difference = actual_qty - expected_qty (computed on each SetInventoryItem call)
- actual_qty allows zero (product counted but none found)

## Indexes
- Inventory: (tenant_id, warehouse_id, status)
- InventoryItem: (inventory_id, product_id) UNIQUE
