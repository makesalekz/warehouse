package data

import (
	"context"
	"fmt"

	"gitlab.calendaria.team/services/warehouse/ent"
	"gitlab.calendaria.team/services/warehouse/ent/enum"
	"gitlab.calendaria.team/services/warehouse/ent/stockitem"
	"gitlab.calendaria.team/services/warehouse/ent/stockmovement"
	"gitlab.calendaria.team/services/warehouse/ent/warehouse"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type StockMovementsRepo interface {
	CreateReceipt(ctx context.Context, dto ReceiptDto) ([]*ent.StockMovement, error)
	CreateTransfer(ctx context.Context, dto TransferDto) ([]*ent.StockMovement, error)
	CreateWriteOff(ctx context.Context, dto WriteOffDto) ([]*ent.StockMovement, error)
	CreateGift(ctx context.Context, dto GiftDto) ([]*ent.StockMovement, error)
	List(ctx context.Context, tenantID, warehouseID int64, movementType *string, paginate *utils_v1.PaginateRequest) ([]*ent.StockMovement, error)
	Count(ctx context.Context, tenantID, warehouseID int64, movementType *string) (int32, error)
}

type stockMovementsRepo struct {
	db *ent.Client
}

func NewStockMovementsRepo(d *Data) StockMovementsRepo {
	return &stockMovementsRepo{db: d.db}
}

func (r *stockMovementsRepo) CreateReceipt(ctx context.Context, dto ReceiptDto) (movements []*ent.StockMovement, err error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	movements = make([]*ent.StockMovement, 0, len(dto.Items))
	for _, item := range dto.Items {
		builder := tx.StockMovement.Create().
			SetTenantID(dto.TenantID).
			SetWarehouseID(dto.WarehouseID).
			SetType(enum.Receipt).
			SetProductID(item.ProductID).
			SetQuantity(item.Quantity).
			SetCreatedBy(dto.CreatedBy)

		if dto.CounterpartyID != nil {
			builder.SetCounterpartyID(*dto.CounterpartyID)
		}

		var m *ent.StockMovement
		m, err = builder.Save(ctx)
		if err != nil {
			return nil, err
		}
		movements = append(movements, m)

		// Upsert stock item: create or increment quantity
		err = tx.StockItem.Create().
			SetTenantID(dto.TenantID).
			SetWarehouseID(dto.WarehouseID).
			SetProductID(item.ProductID).
			SetQuantity(item.Quantity).
			OnConflictColumns(stockitem.FieldTenantID, stockitem.FieldWarehouseID, stockitem.FieldProductID).
			AddQuantity(item.Quantity).
			Exec(ctx)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return movements, nil
}

func (r *stockMovementsRepo) CreateTransfer(ctx context.Context, dto TransferDto) (movements []*ent.StockMovement, err error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Verify both warehouses belong to the same tenant
	sourceExists, err := tx.Warehouse.Query().
		Where(warehouse.ID(dto.SourceWarehouseID), warehouse.TenantID(dto.TenantID)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !sourceExists {
		err = fmt.Errorf("source warehouse not found")
		return nil, err
	}

	targetExists, err := tx.Warehouse.Query().
		Where(warehouse.ID(dto.TargetWarehouseID), warehouse.TenantID(dto.TenantID)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !targetExists {
		err = fmt.Errorf("target warehouse not found")
		return nil, err
	}

	movements = make([]*ent.StockMovement, 0, len(dto.Items)*2)
	for _, item := range dto.Items {
		// Check source stock
		sourceStock, qErr := tx.StockItem.Query().
			Where(
				stockitem.TenantID(dto.TenantID),
				stockitem.WarehouseID(dto.SourceWarehouseID),
				stockitem.ProductID(item.ProductID),
			).
			Only(ctx)
		if qErr != nil {
			if ent.IsNotFound(qErr) {
				err = fmt.Errorf("insufficient stock for product %d", item.ProductID)
				return nil, err
			}
			err = qErr
			return nil, err
		}

		if sourceStock.Quantity.LessThan(item.Quantity) {
			err = fmt.Errorf("insufficient stock for product %d", item.ProductID)
			return nil, err
		}

		// Decrease source stock
		_, err = tx.StockItem.UpdateOne(sourceStock).
			SetQuantity(sourceStock.Quantity.Sub(item.Quantity)).
			Save(ctx)
		if err != nil {
			return nil, err
		}

		// Create outgoing movement (negative qty on source)
		negQty := item.Quantity.Neg()
		var outMov *ent.StockMovement
		outMov, err = tx.StockMovement.Create().
			SetTenantID(dto.TenantID).
			SetWarehouseID(dto.SourceWarehouseID).
			SetType(enum.Transfer).
			SetProductID(item.ProductID).
			SetQuantity(negQty).
			SetCreatedBy(dto.CreatedBy).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		movements = append(movements, outMov)

		// Upsert target stock (create or increment)
		err = tx.StockItem.Create().
			SetTenantID(dto.TenantID).
			SetWarehouseID(dto.TargetWarehouseID).
			SetProductID(item.ProductID).
			SetQuantity(item.Quantity).
			OnConflictColumns(stockitem.FieldTenantID, stockitem.FieldWarehouseID, stockitem.FieldProductID).
			AddQuantity(item.Quantity).
			Exec(ctx)
		if err != nil {
			return nil, err
		}

		// Create incoming movement (positive qty on target)
		var inMov *ent.StockMovement
		inMov, err = tx.StockMovement.Create().
			SetTenantID(dto.TenantID).
			SetWarehouseID(dto.TargetWarehouseID).
			SetType(enum.Transfer).
			SetProductID(item.ProductID).
			SetQuantity(item.Quantity).
			SetCreatedBy(dto.CreatedBy).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		movements = append(movements, inMov)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return movements, nil
}

func (r *stockMovementsRepo) CreateWriteOff(ctx context.Context, dto WriteOffDto) (movements []*ent.StockMovement, err error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	movements = make([]*ent.StockMovement, 0, len(dto.Items))
	for _, item := range dto.Items {
		// Check stock
		sourceStock, qErr := tx.StockItem.Query().
			Where(
				stockitem.TenantID(dto.TenantID),
				stockitem.WarehouseID(dto.WarehouseID),
				stockitem.ProductID(item.ProductID),
			).
			Only(ctx)
		if qErr != nil {
			if ent.IsNotFound(qErr) {
				err = fmt.Errorf("insufficient stock for product %d", item.ProductID)
				return nil, err
			}
			err = qErr
			return nil, err
		}

		if sourceStock.Quantity.LessThan(item.Quantity) {
			err = fmt.Errorf("insufficient stock for product %d", item.ProductID)
			return nil, err
		}

		// Decrease stock
		_, err = tx.StockItem.UpdateOne(sourceStock).
			SetQuantity(sourceStock.Quantity.Sub(item.Quantity)).
			Save(ctx)
		if err != nil {
			return nil, err
		}

		// Create movement (negative quantity)
		negQty := item.Quantity.Neg()
		reason := item.Reason
		var m *ent.StockMovement
		m, err = tx.StockMovement.Create().
			SetTenantID(dto.TenantID).
			SetWarehouseID(dto.WarehouseID).
			SetType(enum.WriteOff).
			SetProductID(item.ProductID).
			SetQuantity(negQty).
			SetReason(reason).
			SetCreatedBy(dto.CreatedBy).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		movements = append(movements, m)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return movements, nil
}

func (r *stockMovementsRepo) CreateGift(ctx context.Context, dto GiftDto) (movements []*ent.StockMovement, err error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	movements = make([]*ent.StockMovement, 0, len(dto.Items))
	for _, item := range dto.Items {
		var m *ent.StockMovement
		m, err = tx.StockMovement.Create().
			SetTenantID(dto.TenantID).
			SetWarehouseID(dto.WarehouseID).
			SetType(enum.Gift).
			SetProductID(item.ProductID).
			SetQuantity(item.Quantity).
			SetCounterpartyID(dto.CounterpartyID).
			SetCreatedBy(dto.CreatedBy).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		movements = append(movements, m)

		// Upsert stock item: create or increment quantity
		err = tx.StockItem.Create().
			SetTenantID(dto.TenantID).
			SetWarehouseID(dto.WarehouseID).
			SetProductID(item.ProductID).
			SetQuantity(item.Quantity).
			OnConflictColumns(stockitem.FieldTenantID, stockitem.FieldWarehouseID, stockitem.FieldProductID).
			AddQuantity(item.Quantity).
			Exec(ctx)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return movements, nil
}

func (r *stockMovementsRepo) List(ctx context.Context, tenantID, warehouseID int64, movementType *string, paginate *utils_v1.PaginateRequest) ([]*ent.StockMovement, error) {
	query := r.db.StockMovement.Query().
		Where(stockmovement.TenantID(tenantID), stockmovement.WarehouseID(warehouseID))

	if movementType != nil && *movementType != "" {
		query.Where(stockmovement.TypeEQ(enum.MovementType(*movementType)))
	}

	if paginate.GetFromId() != 0 {
		query.Where(stockmovement.IDGT(paginate.GetFromId()))
	}

	limit := int(paginate.GetLimit())
	if limit == 0 {
		limit = 100
	}

	return query.Limit(limit).Order(ent.Asc(stockmovement.FieldID)).All(ctx)
}

func (r *stockMovementsRepo) Count(ctx context.Context, tenantID, warehouseID int64, movementType *string) (int32, error) {
	query := r.db.StockMovement.Query().
		Where(stockmovement.TenantID(tenantID), stockmovement.WarehouseID(warehouseID))

	if movementType != nil && *movementType != "" {
		query.Where(stockmovement.TypeEQ(enum.MovementType(*movementType)))
	}

	count, err := query.Count(ctx)
	return int32(count), err
}
