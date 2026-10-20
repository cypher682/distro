package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListProducts(ctx context.Context, tenantID string) ([]Product, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, sku, name, unit, price_cents
		FROM products
		WHERE tenant_id = $1::uuid
		ORDER BY sku
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Product, 0)
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.SKU, &product.Name, &product.Unit, &product.PriceCents); err != nil {
			return nil, err
		}
		out = append(out, product)
	}
	return out, rows.Err()
}

func (s *PostgresStore) CreateProduct(ctx context.Context, tenantID string, input ProductInput) (Product, error) {
	var product Product
	err := s.pool.QueryRow(ctx, `
		INSERT INTO products (tenant_id, sku, name, unit, price_cents)
		VALUES ($1::uuid, $2, $3, $4, $5)
		RETURNING id::text, sku, name, unit, price_cents
	`, tenantID, input.SKU, input.Name, input.Unit, input.PriceCents).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Unit,
		&product.PriceCents,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Product{}, fmt.Errorf("%w: sku %s already exists", ErrConflict, input.SKU)
		}
		return Product{}, err
	}
	return product, nil
}

func (s *PostgresStore) UpdateProduct(ctx context.Context, tenantID, productID string, input ProductInput) (Product, error) {
	var product Product
	err := s.pool.QueryRow(ctx, `
		UPDATE products
		SET sku = $3, name = $4, unit = $5, price_cents = $6
		WHERE tenant_id = $1::uuid AND id = $2::uuid
		RETURNING id::text, sku, name, unit, price_cents
	`, tenantID, productID, input.SKU, input.Name, input.Unit, input.PriceCents).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Unit,
		&product.PriceCents,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Product{}, fmt.Errorf("%w: product %s", ErrNotFound, productID)
		}
		if isUniqueViolation(err) {
			return Product{}, fmt.Errorf("%w: sku %s already exists", ErrConflict, input.SKU)
		}
		return Product{}, err
	}
	return product, nil
}

func (s *PostgresStore) ListWarehouses(ctx context.Context, tenantID string) ([]Warehouse, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name
		FROM warehouses
		WHERE tenant_id = $1::uuid
		ORDER BY name
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Warehouse, 0)
	for rows.Next() {
		var warehouse Warehouse
		if err := rows.Scan(&warehouse.ID, &warehouse.Name); err != nil {
			return nil, err
		}
		out = append(out, warehouse)
	}
	return out, rows.Err()
}

func (s *PostgresStore) CreateWarehouse(ctx context.Context, tenantID string, input WarehouseInput) (Warehouse, error) {
	var warehouse Warehouse
	err := s.pool.QueryRow(ctx, `
		INSERT INTO warehouses (tenant_id, name)
		VALUES ($1::uuid, $2)
		RETURNING id::text, name
	`, tenantID, input.Name).Scan(&warehouse.ID, &warehouse.Name)
	return warehouse, err
}

func (s *PostgresStore) UpdateWarehouse(ctx context.Context, tenantID, warehouseID string, input WarehouseInput) (Warehouse, error) {
	var warehouse Warehouse
	err := s.pool.QueryRow(ctx, `
		UPDATE warehouses
		SET name = $3
		WHERE tenant_id = $1::uuid AND id = $2::uuid
		RETURNING id::text, name
	`, tenantID, warehouseID, input.Name).Scan(&warehouse.ID, &warehouse.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return Warehouse{}, fmt.Errorf("%w: warehouse %s", ErrNotFound, warehouseID)
	}
	return warehouse, err
}

func (s *PostgresStore) ListRetailers(ctx context.Context, tenantID string) ([]Retailer, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, credit_limit_cents
		FROM retailers
		WHERE tenant_id = $1::uuid
		ORDER BY name
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Retailer, 0)
	for rows.Next() {
		var retailer Retailer
		if err := rows.Scan(&retailer.ID, &retailer.Name, &retailer.CreditLimitCents); err != nil {
			return nil, err
		}
		out = append(out, retailer)
	}
	return out, rows.Err()
}

func (s *PostgresStore) CreateRetailer(ctx context.Context, tenantID string, input RetailerInput) (Retailer, error) {
	var retailer Retailer
	err := s.pool.QueryRow(ctx, `
		INSERT INTO retailers (tenant_id, name, credit_limit_cents)
		VALUES ($1::uuid, $2, $3)
		RETURNING id::text, name, credit_limit_cents
	`, tenantID, input.Name, input.CreditLimitCents).Scan(
		&retailer.ID,
		&retailer.Name,
		&retailer.CreditLimitCents,
	)
	return retailer, err
}

func (s *PostgresStore) UpdateRetailer(ctx context.Context, tenantID, retailerID string, input RetailerInput) (Retailer, error) {
	var retailer Retailer
	err := s.pool.QueryRow(ctx, `
		UPDATE retailers
		SET name = $3, credit_limit_cents = $4
		WHERE tenant_id = $1::uuid AND id = $2::uuid
		RETURNING id::text, name, credit_limit_cents
	`, tenantID, retailerID, input.Name, input.CreditLimitCents).Scan(
		&retailer.ID,
		&retailer.Name,
		&retailer.CreditLimitCents,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Retailer{}, fmt.Errorf("%w: retailer %s", ErrNotFound, retailerID)
	}
	return retailer, err
}

func (s *PostgresStore) ListStock(ctx context.Context, tenantID string) ([]StockLevel, error) {
	return listStock(ctx, s.pool, tenantID)
}

func listStock(ctx context.Context, q querier, tenantID string) ([]StockLevel, error) {
	rows, err := q.Query(ctx, `
		SELECT p.id::text,
		       w.id::text,
		       COALESCE(SUM(sm.quantity_delta), 0)::bigint AS on_hand,
		       COALESCE((
		           SELECT SUM(r.quantity)
		           FROM reservations r
		           WHERE r.tenant_id = p.tenant_id
		             AND r.product_id = p.id
		             AND r.warehouse_id = w.id
		             AND r.status = 'active'
		       ), 0)::bigint AS reserved,
		       COALESCE(SUM(sm.total_cost_cents), 0)::bigint AS inventory_value
		FROM products p
		JOIN warehouses w ON w.tenant_id = p.tenant_id
		LEFT JOIN stock_movements sm
		  ON sm.tenant_id = p.tenant_id
		 AND sm.product_id = p.id
		 AND sm.warehouse_id = w.id
		WHERE p.tenant_id = $1::uuid
		GROUP BY p.id, w.id, p.sku, w.name
		ORDER BY p.sku, w.name
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]StockLevel, 0)
	for rows.Next() {
		var level StockLevel
		if err := rows.Scan(&level.ProductID, &level.WarehouseID, &level.OnHand, &level.Reserved, &level.InventoryValueCents); err != nil {
			return nil, err
		}
		level.Available = level.OnHand - level.Reserved
		if level.OnHand > 0 {
			level.AverageCostCents = level.InventoryValueCents / level.OnHand
		}
		out = append(out, level)
	}
	return out, rows.Err()
}

func (s *PostgresStore) ReceiveStock(ctx context.Context, tenantID, productID, warehouseID string, quantity, unitCostCents int64, reason string) error {
	if unitCostCents < 0 {
		return fmt.Errorf("%w: unit_cost_cents must not be negative", ErrInvalid)
	}
	if quantity < 0 && unitCostCents != 0 {
		return fmt.Errorf("%w: unit_cost_cents is only accepted for receipts", ErrInvalid)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var id string
	if err := tx.QueryRow(ctx, `
		SELECT id::text FROM products
		WHERE tenant_id = $1::uuid AND id = $2::uuid
		FOR UPDATE
	`, tenantID, productID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: product %s", ErrNotFound, productID)
		}
		return err
	}
	if err := tx.QueryRow(ctx, `
		SELECT id::text FROM warehouses
		WHERE tenant_id = $1::uuid AND id = $2::uuid
	`, tenantID, warehouseID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: warehouse %s", ErrNotFound, warehouseID)
		}
		return err
	}

	var onHand, value, reserved int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(quantity_delta), 0)::bigint,
		       COALESCE(SUM(total_cost_cents), 0)::bigint
		FROM stock_movements
		WHERE tenant_id = $1::uuid AND product_id = $2::uuid AND warehouse_id = $3::uuid
	`, tenantID, productID, warehouseID).Scan(&onHand, &value); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(quantity), 0)::bigint
		FROM reservations
		WHERE tenant_id = $1::uuid AND product_id = $2::uuid AND warehouse_id = $3::uuid AND status = 'active'
	`, tenantID, productID, warehouseID).Scan(&reserved); err != nil {
		return err
	}

	totalCost := quantity * unitCostCents
	if quantity < 0 {
		if onHand+quantity < reserved {
			return fmt.Errorf("%w: stock adjustment would make reserved stock unavailable", ErrConflict)
		}
		cost, err := issueCost(value, -quantity, onHand)
		if err != nil {
			return err
		}
		totalCost = -cost
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO stock_movements
			(tenant_id, product_id, warehouse_id, quantity_delta, unit_cost_cents, total_cost_cents, reason)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)
	`, tenantID, productID, warehouseID, quantity, unitCostCents, totalCost, reason); err != nil {
		return err
	}

	if totalCost > 0 {
		if _, err := tx.Exec(ctx, `
			WITH g AS (SELECT gen_random_uuid() AS id)
			INSERT INTO ledger_entries
				(tenant_id, entry_group, account, debit_cents, credit_cents, reference_type, reference_id)
			SELECT $1::uuid, g.id, 'inventory', $2, 0, 'stock_receipt', $3::uuid FROM g
			UNION ALL
			SELECT $1::uuid, g.id, 'accounts_payable', 0, $2, 'stock_receipt', $3::uuid FROM g
		`, tenantID, totalCost, productID); err != nil {
			return err
		}
	}
	if totalCost < 0 {
		if _, err := tx.Exec(ctx, `
			WITH g AS (SELECT gen_random_uuid() AS id)
			INSERT INTO ledger_entries
				(tenant_id, entry_group, account, debit_cents, credit_cents, reference_type, reference_id)
			SELECT $1::uuid, g.id, 'inventory_shrinkage', $2, 0, 'stock_adjustment', $3::uuid FROM g
			UNION ALL
			SELECT $1::uuid, g.id, 'inventory', 0, $2, 'stock_adjustment', $3::uuid FROM g
		`, tenantID, -totalCost, productID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) CreateOrder(ctx context.Context, tenantID, retailerID string, lines []CreateOrderLine) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	var id string
	if err := tx.QueryRow(ctx, `
		SELECT id::text FROM retailers
		WHERE tenant_id = $1::uuid AND id = $2::uuid
	`, tenantID, retailerID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, fmt.Errorf("%w: retailer %s", ErrNotFound, retailerID)
		}
		return Order{}, err
	}

	order := Order{
		RetailerID: retailerID,
		Status:     "draft",
	}
	for _, line := range lines {
		var price int64
		if err := tx.QueryRow(ctx, `
			SELECT price_cents FROM products
			WHERE tenant_id = $1::uuid AND id = $2::uuid
		`, tenantID, line.ProductID).Scan(&price); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Order{}, fmt.Errorf("%w: product %s", ErrNotFound, line.ProductID)
			}
			return Order{}, err
		}
		lineTotal := int64(line.Quantity) * price
		order.Lines = append(order.Lines, OrderLine{
			ProductID:      line.ProductID,
			Quantity:       line.Quantity,
			UnitPriceCents: price,
			LineTotalCents: lineTotal,
		})
		order.TotalCents += lineTotal
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO orders (tenant_id, retailer_id, status, total_cents)
		VALUES ($1::uuid, $2::uuid, 'draft', $3)
		RETURNING id::text, created_at
	`, tenantID, retailerID, order.TotalCents).Scan(&order.ID, &order.CreatedAt); err != nil {
		return Order{}, err
	}

	for i := range order.Lines {
		line := &order.Lines[i]
		if err := tx.QueryRow(ctx, `
			INSERT INTO order_lines (order_id, product_id, quantity, unit_price_cents)
			VALUES ($1::uuid, $2::uuid, $3, $4)
			RETURNING id::text
		`, order.ID, line.ProductID, line.Quantity, line.UnitPriceCents).Scan(&line.ID); err != nil {
			return Order{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}
	return order, nil
}

func (s *PostgresStore) ListOrders(ctx context.Context, tenantID string) ([]Order, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, retailer_id::text, status, total_cents, created_at
		FROM orders
		WHERE tenant_id = $1::uuid
		ORDER BY created_at DESC, id DESC
		LIMIT 100
	`, tenantID)
	if err != nil {
		return nil, err
	}
	orders := make([]Order, 0)
	index := make(map[string]int)
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.RetailerID, &order.Status, &order.TotalCents, &order.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		index[order.ID] = len(orders)
		orders = append(orders, order)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return orders, nil
	}

	ids := make([]string, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.ID)
	}
	lineRows, err := s.pool.Query(ctx, `
		SELECT order_id::text, id::text, product_id::text, quantity, unit_price_cents
		FROM order_lines
		WHERE order_id::text = ANY($1::text[])
		ORDER BY id
	`, ids)
	if err != nil {
		return nil, err
	}
	defer lineRows.Close()
	for lineRows.Next() {
		var orderID string
		var line OrderLine
		if err := lineRows.Scan(&orderID, &line.ID, &line.ProductID, &line.Quantity, &line.UnitPriceCents); err != nil {
			return nil, err
		}
		line.LineTotalCents = int64(line.Quantity) * line.UnitPriceCents
		i, ok := index[orderID]
		if ok {
			orders[i].Lines = append(orders[i].Lines, line)
		}
	}
	return orders, lineRows.Err()
}

func (s *PostgresStore) GetOrder(ctx context.Context, tenantID, orderID string) (Order, error) {
	return loadOrder(ctx, s.pool, tenantID, orderID)
}

func (s *PostgresStore) ReserveOrder(ctx context.Context, tenantID, orderID string) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	var status, retailerID string
	var totalCents int64
	if err := tx.QueryRow(ctx, `
		SELECT status, retailer_id::text, total_cents
		FROM orders
		WHERE tenant_id = $1::uuid AND id = $2::uuid
		FOR UPDATE
	`, tenantID, orderID).Scan(&status, &retailerID, &totalCents); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
		}
		return Order{}, err
	}
	if status != "draft" {
		return Order{}, fmt.Errorf("%w: order %s is %s, expected draft", ErrConflict, orderID, status)
	}

	var creditLimit int64
	if err := tx.QueryRow(ctx, `
		SELECT credit_limit_cents FROM retailers
		WHERE tenant_id = $1::uuid AND id = $2::uuid
		FOR UPDATE
	`, tenantID, retailerID).Scan(&creditLimit); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, fmt.Errorf("%w: retailer %s", ErrNotFound, retailerID)
		}
		return Order{}, err
	}

	var exposure int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE((
		           SELECT SUM(total_cents - paid_cents)
		           FROM invoices
		           WHERE tenant_id = $1::uuid AND retailer_id = $2::uuid AND status = 'open'
		       ), 0)::bigint
		     + COALESCE((
		           SELECT SUM(total_cents)
		           FROM orders
		           WHERE tenant_id = $1::uuid AND retailer_id = $2::uuid AND status = 'reserved'
		       ), 0)::bigint
	`, tenantID, retailerID).Scan(&exposure); err != nil {
		return Order{}, err
	}
	if exposure+totalCents > creditLimit {
		return Order{}, fmt.Errorf("%w: credit limit exceeded for retailer %s", ErrConflict, retailerID)
	}

	var warehouseID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text FROM warehouses
		WHERE tenant_id = $1::uuid
		ORDER BY created_at, id
		LIMIT 1
	`, tenantID).Scan(&warehouseID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, fmt.Errorf("%w: no warehouse for tenant", ErrNotFound)
		}
		return Order{}, err
	}

	type reserveLine struct {
		productID string
		quantity  int64
	}
	rows, err := tx.Query(ctx, `
		SELECT product_id::text, quantity
		FROM order_lines
		WHERE order_id = $1::uuid
		ORDER BY product_id
	`, orderID)
	if err != nil {
		return Order{}, err
	}
	lines := make([]reserveLine, 0)
	for rows.Next() {
		var line reserveLine
		if err := rows.Scan(&line.productID, &line.quantity); err != nil {
			rows.Close()
			return Order{}, err
		}
		lines = append(lines, line)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Order{}, err
	}
	if len(lines) == 0 {
		return Order{}, fmt.Errorf("%w: order %s has no lines", ErrConflict, orderID)
	}

	for _, line := range lines {
		var id string
		if err := tx.QueryRow(ctx, `
			SELECT id::text FROM products
			WHERE tenant_id = $1::uuid AND id = $2::uuid
			FOR UPDATE
		`, tenantID, line.productID).Scan(&id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Order{}, fmt.Errorf("%w: product %s", ErrNotFound, line.productID)
			}
			return Order{}, err
		}

		var onHand, reserved int64
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(quantity_delta), 0)::bigint
			FROM stock_movements
			WHERE tenant_id = $1::uuid AND product_id = $2::uuid AND warehouse_id = $3::uuid
		`, tenantID, line.productID, warehouseID).Scan(&onHand); err != nil {
			return Order{}, err
		}
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(quantity), 0)::bigint
			FROM reservations
			WHERE tenant_id = $1::uuid AND product_id = $2::uuid AND warehouse_id = $3::uuid AND status = 'active'
		`, tenantID, line.productID, warehouseID).Scan(&reserved); err != nil {
			return Order{}, err
		}
		if onHand-reserved < line.quantity {
			return Order{}, fmt.Errorf("%w: insufficient stock for product %s", ErrConflict, line.productID)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO reservations
				(tenant_id, order_id, product_id, warehouse_id, quantity, status)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, 'active')
		`, tenantID, orderID, line.productID, warehouseID, line.quantity); err != nil {
			return Order{}, err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE orders SET status = 'reserved'
		WHERE tenant_id = $1::uuid AND id = $2::uuid
	`, tenantID, orderID); err != nil {
		return Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}
	return s.GetOrder(ctx, tenantID, orderID)
}

func (s *PostgresStore) CancelOrder(ctx context.Context, tenantID, orderID string) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	var status string
	if err := tx.QueryRow(ctx, `
		SELECT status FROM orders
		WHERE tenant_id = $1::uuid AND id = $2::uuid
		FOR UPDATE
	`, tenantID, orderID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
		}
		return Order{}, err
	}

	switch status {
	case "cancelled":
		if err := tx.Commit(ctx); err != nil {
			return Order{}, err
		}
	case "draft":
		if _, err := tx.Exec(ctx, `
			UPDATE orders SET status = 'cancelled'
			WHERE tenant_id = $1::uuid AND id = $2::uuid
		`, tenantID, orderID); err != nil {
			return Order{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Order{}, err
		}
	case "reserved":
		if _, err := tx.Exec(ctx, `
			UPDATE reservations SET status = 'released'
			WHERE tenant_id = $1::uuid AND order_id = $2::uuid AND status = 'active'
		`, tenantID, orderID); err != nil {
			return Order{}, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE orders SET status = 'cancelled'
			WHERE tenant_id = $1::uuid AND id = $2::uuid
		`, tenantID, orderID); err != nil {
			return Order{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Order{}, err
		}
	default:
		return Order{}, fmt.Errorf("%w: order %s is %s and cannot be cancelled", ErrConflict, orderID, status)
	}
	return s.GetOrder(ctx, tenantID, orderID)
}

func (s *PostgresStore) InvoiceOrder(ctx context.Context, tenantID, orderID string) (Invoice, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Invoice{}, err
	}
	defer tx.Rollback(ctx)

	var status, retailerID string
	var totalCents int64
	if err := tx.QueryRow(ctx, `
		SELECT status, retailer_id::text, total_cents
		FROM orders
		WHERE tenant_id = $1::uuid AND id = $2::uuid
		FOR UPDATE
	`, tenantID, orderID).Scan(&status, &retailerID, &totalCents); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Invoice{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
		}
		return Invoice{}, err
	}
	if status != "reserved" {
		return Invoice{}, fmt.Errorf("%w: order %s is %s, expected reserved", ErrConflict, orderID, status)
	}

	invoice := Invoice{
		OrderID:    orderID,
		RetailerID: retailerID,
		Status:     "open",
		TotalCents: totalCents,
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO invoices (tenant_id, order_id, retailer_id, status, total_cents, paid_cents)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'open', $4, 0)
		RETURNING id::text, created_at
	`, tenantID, orderID, retailerID, totalCents).Scan(&invoice.ID, &invoice.CreatedAt); err != nil {
		return Invoice{}, err
	}

	type fulfillment struct {
		productID   string
		warehouseID string
		quantity    int64
	}
	rows, err := tx.Query(ctx, `
		SELECT product_id::text, warehouse_id::text, quantity
		FROM reservations
		WHERE tenant_id = $1::uuid AND order_id = $2::uuid AND status = 'active'
		ORDER BY product_id, warehouse_id
		FOR UPDATE
	`, tenantID, orderID)
	if err != nil {
		return Invoice{}, err
	}
	fulfillments := make([]fulfillment, 0)
	for rows.Next() {
		var item fulfillment
		if err := rows.Scan(&item.productID, &item.warehouseID, &item.quantity); err != nil {
			rows.Close()
			return Invoice{}, err
		}
		fulfillments = append(fulfillments, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Invoice{}, err
	}
	if len(fulfillments) == 0 {
		return Invoice{}, fmt.Errorf("%w: order %s has no active reservations", ErrConflict, orderID)
	}

	type orderLineSnapshot struct {
		id             string
		productID      string
		quantity       int
		unitPriceCents int64
	}
	lineRows, err := tx.Query(ctx, `
		SELECT id::text, product_id::text, quantity, unit_price_cents
		FROM order_lines
		WHERE order_id = $1::uuid
		ORDER BY product_id
	`, orderID)
	if err != nil {
		return Invoice{}, err
	}
	orderLines := make(map[string]orderLineSnapshot, len(fulfillments))
	for lineRows.Next() {
		var line orderLineSnapshot
		if err := lineRows.Scan(&line.id, &line.productID, &line.quantity, &line.unitPriceCents); err != nil {
			lineRows.Close()
			return Invoice{}, err
		}
		orderLines[line.productID] = line
	}
	lineRows.Close()
	if err := lineRows.Err(); err != nil {
		return Invoice{}, err
	}

	lockedProducts := make(map[string]struct{}, len(fulfillments))
	var cogs int64
	for _, item := range fulfillments {
		if _, locked := lockedProducts[item.productID]; !locked {
			var id string
			if err := tx.QueryRow(ctx, `
				SELECT id::text FROM products
				WHERE tenant_id = $1::uuid AND id = $2::uuid
				FOR UPDATE
			`, tenantID, item.productID).Scan(&id); err != nil {
				return Invoice{}, err
			}
			lockedProducts[item.productID] = struct{}{}
		}

		var onHand, value int64
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(quantity_delta), 0)::bigint,
			       COALESCE(SUM(total_cost_cents), 0)::bigint
			FROM stock_movements
			WHERE tenant_id = $1::uuid AND product_id = $2::uuid AND warehouse_id = $3::uuid
		`, tenantID, item.productID, item.warehouseID).Scan(&onHand, &value); err != nil {
			return Invoice{}, err
		}
		cost, err := issueCost(value, item.quantity, onHand)
		if err != nil {
			return Invoice{}, err
		}
		cogs += cost
		if _, err := tx.Exec(ctx, `
			INSERT INTO stock_movements
				(tenant_id, product_id, warehouse_id, quantity_delta, unit_cost_cents, total_cost_cents, reason, reference_type, reference_id)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, 0, $5, 'invoice fulfilled', 'invoice', $6::uuid)
		`, tenantID, item.productID, item.warehouseID, -item.quantity, -cost, invoice.ID); err != nil {
			return Invoice{}, err
		}

		orderLine, ok := orderLines[item.productID]
		if !ok {
			return Invoice{}, fmt.Errorf("%w: missing order line for product %s", ErrConflict, item.productID)
		}
		if int64(orderLine.quantity) != item.quantity {
			return Invoice{}, fmt.Errorf("%w: reservation quantity does not match order line for product %s", ErrConflict, item.productID)
		}
		invoiceLine := InvoiceLine{
			InvoiceID:      invoice.ID,
			ProductID:      item.productID,
			WarehouseID:    item.warehouseID,
			Quantity:       orderLine.quantity,
			UnitPriceCents: orderLine.unitPriceCents,
			LineTotalCents: int64(orderLine.quantity) * orderLine.unitPriceCents,
			TotalCostCents: cost,
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO invoice_lines
				(tenant_id, invoice_id, order_line_id, product_id, warehouse_id, quantity, unit_price_cents, line_total_cents, total_cost_cents)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid, $6, $7, $8, $9)
			RETURNING id::text
		`, tenantID, invoice.ID, orderLine.id, item.productID, item.warehouseID, orderLine.quantity, orderLine.unitPriceCents, invoiceLine.LineTotalCents, cost).Scan(&invoiceLine.ID); err != nil {
			return Invoice{}, err
		}
		invoice.Lines = append(invoice.Lines, invoiceLine)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE reservations SET status = 'fulfilled'
		WHERE tenant_id = $1::uuid AND order_id = $2::uuid AND status = 'active'
	`, tenantID, orderID); err != nil {
		return Invoice{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE orders SET status = 'invoiced'
		WHERE tenant_id = $1::uuid AND id = $2::uuid
	`, tenantID, orderID); err != nil {
		return Invoice{}, err
	}

	if _, err := tx.Exec(ctx, `
		WITH g AS (SELECT gen_random_uuid() AS id)
		INSERT INTO ledger_entries
			(tenant_id, entry_group, account, debit_cents, credit_cents, reference_type, reference_id)
		SELECT $1::uuid, g.id, 'accounts_receivable', $2, 0, 'invoice', $3::uuid FROM g
		UNION ALL
		SELECT $1::uuid, g.id, 'revenue', 0, $2, 'invoice', $3::uuid FROM g
	`, tenantID, totalCents, invoice.ID); err != nil {
		return Invoice{}, err
	}

	if cogs > 0 {
		if _, err := tx.Exec(ctx, `
			WITH g AS (SELECT gen_random_uuid() AS id)
			INSERT INTO ledger_entries
				(tenant_id, entry_group, account, debit_cents, credit_cents, reference_type, reference_id)
			SELECT $1::uuid, g.id, 'cost_of_goods_sold', $2, 0, 'invoice', $3::uuid FROM g
			UNION ALL
			SELECT $1::uuid, g.id, 'inventory', 0, $2, 'invoice', $3::uuid FROM g
		`, tenantID, cogs, invoice.ID); err != nil {
			return Invoice{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Invoice{}, err
	}
	return invoice, nil
}

func (s *PostgresStore) ListInvoices(ctx context.Context, tenantID string) ([]Invoice, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id::text,
		       i.order_id::text,
		       i.retailer_id::text,
		       i.status,
		       i.total_cents,
		       i.paid_cents,
		       COALESCE((
		           SELECT SUM(sr.total_refund_cents)
		           FROM sales_returns sr
		           WHERE sr.tenant_id = i.tenant_id AND sr.invoice_id = i.id
		       ), 0)::bigint,
		       i.created_at
		FROM invoices i
		WHERE i.tenant_id = $1::uuid
		ORDER BY i.created_at DESC, i.id DESC
		LIMIT 100
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Invoice, 0)
	for rows.Next() {
		var invoice Invoice
		if err := rows.Scan(
			&invoice.ID,
			&invoice.OrderID,
			&invoice.RetailerID,
			&invoice.Status,
			&invoice.TotalCents,
			&invoice.PaidCents,
			&invoice.ReturnedCents,
			&invoice.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, invoice)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetInvoice(ctx context.Context, tenantID, invoiceID string) (InvoiceDetail, error) {
	var detail InvoiceDetail
	if err := s.pool.QueryRow(ctx, `
		SELECT i.id::text,
		       i.order_id::text,
		       i.retailer_id::text,
		       i.status,
		       i.total_cents,
		       i.paid_cents,
		       COALESCE((
		           SELECT SUM(sr.total_refund_cents)
		           FROM sales_returns sr
		           WHERE sr.tenant_id = i.tenant_id AND sr.invoice_id = i.id
		       ), 0)::bigint,
		       i.created_at
		FROM invoices i
		WHERE i.tenant_id = $1::uuid AND i.id = $2::uuid
	`, tenantID, invoiceID).Scan(
		&detail.ID,
		&detail.OrderID,
		&detail.RetailerID,
		&detail.Status,
		&detail.TotalCents,
		&detail.PaidCents,
		&detail.ReturnedCents,
		&detail.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InvoiceDetail{}, fmt.Errorf("%w: invoice %s", ErrNotFound, invoiceID)
		}
		return InvoiceDetail{}, err
	}

	lineRows, err := s.pool.Query(ctx, `
		SELECT id::text, invoice_id::text, product_id::text, warehouse_id::text,
		       quantity, unit_price_cents, line_total_cents, total_cost_cents
		FROM invoice_lines
		WHERE tenant_id = $1::uuid AND invoice_id = $2::uuid
		ORDER BY product_id
	`, tenantID, invoiceID)
	if err != nil {
		return InvoiceDetail{}, err
	}
	for lineRows.Next() {
		var line InvoiceLine
		if err := lineRows.Scan(
			&line.ID,
			&line.InvoiceID,
			&line.ProductID,
			&line.WarehouseID,
			&line.Quantity,
			&line.UnitPriceCents,
			&line.LineTotalCents,
			&line.TotalCostCents,
		); err != nil {
			lineRows.Close()
			return InvoiceDetail{}, err
		}
		detail.Lines = append(detail.Lines, line)
	}
	lineRows.Close()
	if err := lineRows.Err(); err != nil {
		return InvoiceDetail{}, err
	}

	returnRows, err := s.pool.Query(ctx, `
		SELECT id::text, retailer_id::text, total_refund_cents, total_cost_cents, created_at
		FROM sales_returns
		WHERE tenant_id = $1::uuid AND invoice_id = $2::uuid
		ORDER BY created_at DESC, id DESC
	`, tenantID, invoiceID)
	if err != nil {
		return InvoiceDetail{}, err
	}
	for returnRows.Next() {
		var salesReturn SalesReturn
		salesReturn.InvoiceID = invoiceID
		if err := returnRows.Scan(
			&salesReturn.ID,
			&salesReturn.RetailerID,
			&salesReturn.TotalRefundCents,
			&salesReturn.TotalCostCents,
			&salesReturn.CreatedAt,
		); err != nil {
			returnRows.Close()
			return InvoiceDetail{}, err
		}
		detail.Returns = append(detail.Returns, salesReturn)
	}
	returnRows.Close()
	if err := returnRows.Err(); err != nil {
		return InvoiceDetail{}, err
	}

	if len(detail.Returns) > 0 {
		returnIDs := make([]string, 0, len(detail.Returns))
		returnIndex := make(map[string]int, len(detail.Returns))
		for i, salesReturn := range detail.Returns {
			returnIDs = append(returnIDs, salesReturn.ID)
			returnIndex[salesReturn.ID] = i
		}
		returnLineRows, err := s.pool.Query(ctx, `
			SELECT id::text, sales_return_id::text, invoice_line_id::text, product_id::text,
			       quantity, refund_cents, cost_cents, restock
			FROM sales_return_lines
			WHERE tenant_id = $1::uuid AND sales_return_id::text = ANY($2::text[])
			ORDER BY id
		`, tenantID, returnIDs)
		if err != nil {
			return InvoiceDetail{}, err
		}
		for returnLineRows.Next() {
			var line SalesReturnLine
			if err := returnLineRows.Scan(
				&line.ID,
				&line.SalesReturnID,
				&line.InvoiceLineID,
				&line.ProductID,
				&line.Quantity,
				&line.RefundCents,
				&line.CostCents,
				&line.Restock,
			); err != nil {
				returnLineRows.Close()
				return InvoiceDetail{}, err
			}
			if i, ok := returnIndex[line.SalesReturnID]; ok {
				detail.Returns[i].Lines = append(detail.Returns[i].Lines, line)
			}
		}
		returnLineRows.Close()
		if err := returnLineRows.Err(); err != nil {
			return InvoiceDetail{}, err
		}
	}

	paymentRows, err := s.pool.Query(ctx, `
		SELECT id::text, invoice_id::text, amount_cents, created_at
		FROM payments
		WHERE tenant_id = $1::uuid AND invoice_id = $2::uuid
		ORDER BY created_at DESC, id DESC
	`, tenantID, invoiceID)
	if err != nil {
		return InvoiceDetail{}, err
	}
	for paymentRows.Next() {
		var payment Payment
		if err := paymentRows.Scan(&payment.ID, &payment.InvoiceID, &payment.AmountCents, &payment.CreatedAt); err != nil {
			paymentRows.Close()
			return InvoiceDetail{}, err
		}
		detail.Payments = append(detail.Payments, payment)
	}
	paymentRows.Close()
	if err := paymentRows.Err(); err != nil {
		return InvoiceDetail{}, err
	}

	return detail, nil
}

func (s *PostgresStore) GetCredit(ctx context.Context, tenantID, retailerID string) (CreditView, error) {
	var view CreditView
	view.RetailerID = retailerID
	if err := s.pool.QueryRow(ctx, `
		SELECT credit_limit_cents FROM retailers
		WHERE tenant_id = $1::uuid AND id = $2::uuid
	`, tenantID, retailerID).Scan(&view.CreditLimitCents); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CreditView{}, fmt.Errorf("%w: retailer %s", ErrNotFound, retailerID)
		}
		return CreditView{}, err
	}

	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE((
		           SELECT SUM(i.total_cents - i.paid_cents - COALESCE(r.refund_cents, 0))
		           FROM invoices i
		           LEFT JOIN (
		               SELECT invoice_id, SUM(total_refund_cents)::bigint AS refund_cents
		               FROM sales_returns
		               WHERE tenant_id = $1::uuid
		               GROUP BY invoice_id
		           ) r ON r.invoice_id = i.id
		           WHERE i.tenant_id = $1::uuid
		             AND i.retailer_id = $2::uuid
		             AND i.status <> 'void'
		       ), 0)::bigint,
		       COALESCE((
		           SELECT SUM(total_cents)
		           FROM orders
		           WHERE tenant_id = $1::uuid AND retailer_id = $2::uuid AND status = 'reserved'
		       ), 0)::bigint
	`, tenantID, retailerID).Scan(&view.OpenInvoiceCents, &view.ReservedOrderCents); err != nil {
		return CreditView{}, err
	}

	view.ExposureCents = view.OpenInvoiceCents + view.ReservedOrderCents
	view.AvailableCents = view.CreditLimitCents - view.ExposureCents
	return view, nil
}

func (s *PostgresStore) RecordPayment(ctx context.Context, tenantID, invoiceID string, amountCents int64) (Payment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Payment{}, err
	}
	defer tx.Rollback(ctx)

	var status string
	var totalCents, paidCents, returnedCents int64
	if err := tx.QueryRow(ctx, `
		SELECT status, total_cents, paid_cents,
		       COALESCE((SELECT SUM(total_refund_cents) FROM sales_returns WHERE invoice_id = invoices.id), 0)::bigint
		FROM invoices
		WHERE tenant_id = $1::uuid AND id = $2::uuid
		FOR UPDATE
	`, tenantID, invoiceID).Scan(&status, &totalCents, &paidCents, &returnedCents); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Payment{}, fmt.Errorf("%w: invoice %s", ErrNotFound, invoiceID)
		}
		return Payment{}, err
	}
	if status != "open" {
		return Payment{}, fmt.Errorf("%w: invoice %s is %s", ErrConflict, invoiceID, status)
	}
	if amountCents > totalCents-paidCents-returnedCents {
		return Payment{}, fmt.Errorf("%w: payment exceeds remaining invoice balance", ErrConflict)
	}

	payment := Payment{
		InvoiceID:   invoiceID,
		AmountCents: amountCents,
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO payments (tenant_id, invoice_id, amount_cents)
		VALUES ($1::uuid, $2::uuid, $3)
		RETURNING id::text, created_at
	`, tenantID, invoiceID, amountCents).Scan(&payment.ID, &payment.CreatedAt); err != nil {
		return Payment{}, err
	}

	newPaid := paidCents + amountCents
	newStatus := "open"
	if newPaid+returnedCents >= totalCents {
		newStatus = "paid"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE invoices SET paid_cents = $3, status = $4
		WHERE tenant_id = $1::uuid AND id = $2::uuid
	`, tenantID, invoiceID, newPaid, newStatus); err != nil {
		return Payment{}, err
	}

	if _, err := tx.Exec(ctx, `
		WITH g AS (SELECT gen_random_uuid() AS id)
		INSERT INTO ledger_entries
			(tenant_id, entry_group, account, debit_cents, credit_cents, reference_type, reference_id)
		SELECT $1::uuid, g.id, 'cash', $2, 0, 'payment', $3::uuid FROM g
		UNION ALL
		SELECT $1::uuid, g.id, 'accounts_receivable', 0, $2, 'payment', $3::uuid FROM g
	`, tenantID, amountCents, payment.ID); err != nil {
		return Payment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Payment{}, err
	}
	return payment, nil
}

func (s *PostgresStore) ReturnInvoice(ctx context.Context, tenantID, invoiceID string, lines []CreateReturnLine) (SalesReturn, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SalesReturn{}, err
	}
	defer tx.Rollback(ctx)

	var status, retailerID string
	var totalCents, paidCents, returnedCents int64
	if err := tx.QueryRow(ctx, `
		SELECT status, retailer_id::text, total_cents, paid_cents,
		       COALESCE((SELECT SUM(total_refund_cents) FROM sales_returns WHERE invoice_id = invoices.id), 0)::bigint
		FROM invoices
		WHERE tenant_id = $1::uuid AND id = $2::uuid
		FOR UPDATE
	`, tenantID, invoiceID).Scan(&status, &retailerID, &totalCents, &paidCents, &returnedCents); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SalesReturn{}, fmt.Errorf("%w: invoice %s", ErrNotFound, invoiceID)
		}
		return SalesReturn{}, err
	}
	if status == "void" {
		return SalesReturn{}, fmt.Errorf("%w: invoice %s is void", ErrConflict, invoiceID)
	}

	type lineState struct {
		id               string
		productID        string
		warehouseID      string
		quantity         int64
		unitPriceCents   int64
		totalCostCents   int64
		returnedQuantity int64
		returnedCost     int64
	}
	rows, err := tx.Query(ctx, `
		SELECT il.id::text,
		       il.product_id::text,
		       il.warehouse_id::text,
		       il.quantity,
		       il.unit_price_cents,
		       il.total_cost_cents,
		       COALESCE(SUM(srl.quantity), 0)::bigint,
		       COALESCE(SUM(srl.cost_cents), 0)::bigint
		FROM invoice_lines il
		LEFT JOIN sales_return_lines srl
		       ON srl.invoice_line_id = il.id
		      AND srl.tenant_id = il.tenant_id
		WHERE il.tenant_id = $1::uuid AND il.invoice_id = $2::uuid
		GROUP BY il.id, il.product_id, il.warehouse_id, il.quantity,
		         il.unit_price_cents, il.total_cost_cents
		ORDER BY il.product_id
	`, tenantID, invoiceID)
	if err != nil {
		return SalesReturn{}, err
	}
	byProduct := make(map[string]lineState)
	for rows.Next() {
		var line lineState
		if err := rows.Scan(&line.id, &line.productID, &line.warehouseID, &line.quantity,
			&line.unitPriceCents, &line.totalCostCents, &line.returnedQuantity, &line.returnedCost); err != nil {
			rows.Close()
			return SalesReturn{}, err
		}
		byProduct[line.productID] = line
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return SalesReturn{}, err
	}
	if len(byProduct) == 0 {
		return SalesReturn{}, fmt.Errorf("%w: invoice %s has no lines", ErrConflict, invoiceID)
	}

	salesReturn := SalesReturn{
		InvoiceID:  invoiceID,
		RetailerID: retailerID,
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO sales_returns
			(tenant_id, invoice_id, retailer_id, total_refund_cents, total_cost_cents)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 0, 0)
		RETURNING id::text, created_at
	`, tenantID, invoiceID, retailerID).Scan(&salesReturn.ID, &salesReturn.CreatedAt); err != nil {
		return SalesReturn{}, err
	}

	seen := make(map[string]struct{}, len(lines))
	var restockCost int64
	for _, request := range lines {
		if request.ProductID == "" || request.Quantity <= 0 {
			return SalesReturn{}, fmt.Errorf("%w: product_id and positive quantity are required", ErrInvalid)
		}
		if _, duplicate := seen[request.ProductID]; duplicate {
			return SalesReturn{}, fmt.Errorf("%w: duplicate product_id %s", ErrInvalid, request.ProductID)
		}
		seen[request.ProductID] = struct{}{}

		line, ok := byProduct[request.ProductID]
		if !ok {
			return SalesReturn{}, fmt.Errorf("%w: product %s was not invoiced", ErrInvalid, request.ProductID)
		}
		remainingQuantity := line.quantity - line.returnedQuantity
		if int64(request.Quantity) > remainingQuantity {
			return SalesReturn{}, fmt.Errorf("%w: return quantity exceeds remaining invoice quantity for product %s", ErrConflict, request.ProductID)
		}
		remainingCost := line.totalCostCents - line.returnedCost
		cost, err := issueCost(remainingCost, int64(request.Quantity), remainingQuantity)
		if err != nil {
			return SalesReturn{}, err
		}

		returnLine := SalesReturnLine{
			SalesReturnID: salesReturn.ID,
			InvoiceLineID: line.id,
			ProductID:     request.ProductID,
			Quantity:      request.Quantity,
			RefundCents:   int64(request.Quantity) * line.unitPriceCents,
			CostCents:     cost,
			Restock:       request.Restock,
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO sales_return_lines
				(tenant_id, sales_return_id, invoice_line_id, product_id, quantity, refund_cents, cost_cents, restock)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8)
			RETURNING id::text
		`, tenantID, salesReturn.ID, line.id, request.ProductID, request.Quantity,
			returnLine.RefundCents, returnLine.CostCents, request.Restock).Scan(&returnLine.ID); err != nil {
			return SalesReturn{}, err
		}
		salesReturn.Lines = append(salesReturn.Lines, returnLine)
		salesReturn.TotalRefundCents += returnLine.RefundCents
		salesReturn.TotalCostCents += returnLine.CostCents

		if request.Restock {
			restockCost += returnLine.CostCents
			if _, err := tx.Exec(ctx, `
				INSERT INTO stock_movements
					(tenant_id, product_id, warehouse_id, quantity_delta, unit_cost_cents, total_cost_cents, reason, reference_type, reference_id)
				VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, 'sales return restock', 'sales_return', $7::uuid)
			`, tenantID, request.ProductID, line.warehouseID, request.Quantity,
				returnLine.CostCents/int64(request.Quantity), returnLine.CostCents, salesReturn.ID); err != nil {
				return SalesReturn{}, err
			}
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sales_returns
		SET total_refund_cents = $3, total_cost_cents = $4
		WHERE tenant_id = $1::uuid AND id = $2::uuid
	`, tenantID, salesReturn.ID, salesReturn.TotalRefundCents, salesReturn.TotalCostCents); err != nil {
		return SalesReturn{}, err
	}

	if salesReturn.TotalRefundCents > 0 {
		if _, err := tx.Exec(ctx, `
			WITH g AS (SELECT gen_random_uuid() AS id)
			INSERT INTO ledger_entries
				(tenant_id, entry_group, account, debit_cents, credit_cents, reference_type, reference_id)
			SELECT $1::uuid, g.id, 'sales_returns', $2, 0, 'sales_return', $3::uuid FROM g
			UNION ALL
			SELECT $1::uuid, g.id, 'accounts_receivable', 0, $2, 'sales_return', $3::uuid FROM g
		`, tenantID, salesReturn.TotalRefundCents, salesReturn.ID); err != nil {
			return SalesReturn{}, err
		}
	}
	if restockCost > 0 {
		if _, err := tx.Exec(ctx, `
			WITH g AS (SELECT gen_random_uuid() AS id)
			INSERT INTO ledger_entries
				(tenant_id, entry_group, account, debit_cents, credit_cents, reference_type, reference_id)
			SELECT $1::uuid, g.id, 'inventory', $2, 0, 'sales_return', $3::uuid FROM g
			UNION ALL
			SELECT $1::uuid, g.id, 'cost_of_goods_sold', 0, $2, 'sales_return', $3::uuid FROM g
		`, tenantID, restockCost, salesReturn.ID); err != nil {
			return SalesReturn{}, err
		}
	}

	newStatus := status
	if status != "paid" && paidCents+returnedCents+salesReturn.TotalRefundCents >= totalCents {
		newStatus = "paid"
	}
	if newStatus != status {
		if _, err := tx.Exec(ctx, `
			UPDATE invoices SET status = $3
			WHERE tenant_id = $1::uuid AND id = $2::uuid
		`, tenantID, invoiceID, newStatus); err != nil {
			return SalesReturn{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SalesReturn{}, err
	}
	return salesReturn, nil
}

func (s *PostgresStore) Valuation(ctx context.Context, tenantID string) (InventoryValuation, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return InventoryValuation{}, err
	}
	defer tx.Rollback(ctx)

	items, err := listStock(ctx, tx, tenantID)
	if err != nil {
		return InventoryValuation{}, err
	}
	valuation := InventoryValuation{Items: items}
	for _, item := range items {
		valuation.TotalValueCents += item.InventoryValueCents
	}
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(debit_cents - credit_cents) FILTER (WHERE account = 'inventory'), 0)::bigint,
		       COALESCE(SUM(debit_cents - credit_cents) FILTER (WHERE account = 'cost_of_goods_sold'), 0)::bigint
		FROM ledger_entries
		WHERE tenant_id = $1::uuid
	`, tenantID).Scan(&valuation.LedgerInventoryCents, &valuation.CogsCents); err != nil {
		return InventoryValuation{}, err
	}
	valuation.Balanced = valuation.TotalValueCents == valuation.LedgerInventoryCents
	if err := tx.Commit(ctx); err != nil {
		return InventoryValuation{}, err
	}
	return valuation, nil
}

func (s *PostgresStore) TrialBalance(ctx context.Context, tenantID string) ([]AccountBalance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT account,
		       COALESCE(SUM(debit_cents), 0)::bigint,
		       COALESCE(SUM(credit_cents), 0)::bigint
		FROM ledger_entries
		WHERE tenant_id = $1::uuid
		GROUP BY account
		ORDER BY account
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AccountBalance, 0)
	for rows.Next() {
		var balance AccountBalance
		if err := rows.Scan(&balance.Account, &balance.DebitCents, &balance.CreditCents); err != nil {
			return nil, err
		}
		balance.BalanceCents = balance.DebitCents - balance.CreditCents
		out = append(out, balance)
	}
	return out, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func loadOrder(ctx context.Context, q querier, tenantID, orderID string) (Order, error) {
	var order Order
	if err := q.QueryRow(ctx, `
		SELECT id::text, retailer_id::text, status, total_cents, created_at
		FROM orders
		WHERE tenant_id = $1::uuid AND id = $2::uuid
	`, tenantID, orderID).Scan(&order.ID, &order.RetailerID, &order.Status, &order.TotalCents, &order.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
		}
		return Order{}, err
	}

	rows, err := q.Query(ctx, `
		SELECT id::text, product_id::text, quantity, unit_price_cents
		FROM order_lines
		WHERE order_id = $1::uuid
		ORDER BY id
	`, orderID)
	if err != nil {
		return Order{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var line OrderLine
		if err := rows.Scan(&line.ID, &line.ProductID, &line.Quantity, &line.UnitPriceCents); err != nil {
			return Order{}, err
		}
		line.LineTotalCents = int64(line.Quantity) * line.UnitPriceCents
		order.Lines = append(order.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return Order{}, err
	}
	return order, nil
}
