package core

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type MemoryStore struct {
	mu             sync.Mutex
	products       map[string]map[string]Product
	retailers      map[string]map[string]Retailer
	suppliers      map[string]map[string]Supplier
	warehouses     map[string]map[string]string
	purchaseOrders map[string]map[string]*PurchaseOrder
	onHand         map[string]int64
	inventoryCost  map[string]int64
	reservations   map[string]*memoryReservation
	orders         map[string]map[string]*Order
	deliveryNotes  map[string]map[string]*DeliveryNote
	invoices       map[string]map[string]*Invoice
	invoiceLines   map[string][]InvoiceLine
	payments       map[string]map[string]*Payment
	returns        map[string]map[string]*SalesReturn
	ledger         []memoryLedgerEntry
	nextID         uint64
}

type memoryReservation struct {
	ID          string
	TenantID    string
	OrderID     string
	ProductID   string
	WarehouseID string
	Quantity    int64
	Status      string
}

type memoryLedgerEntry struct {
	TenantID      string
	Account       string
	DebitCents    int64
	CreditCents   int64
	ReferenceType string
	ReferenceID   string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		products:       make(map[string]map[string]Product),
		retailers:      make(map[string]map[string]Retailer),
		suppliers:      make(map[string]map[string]Supplier),
		warehouses:     make(map[string]map[string]string),
		purchaseOrders: make(map[string]map[string]*PurchaseOrder),
		onHand:         make(map[string]int64),
		inventoryCost:  make(map[string]int64),
		reservations:   make(map[string]*memoryReservation),
		orders:         make(map[string]map[string]*Order),
		deliveryNotes:  make(map[string]map[string]*DeliveryNote),
		invoices:       make(map[string]map[string]*Invoice),
		invoiceLines:   make(map[string][]InvoiceLine),
		payments:       make(map[string]map[string]*Payment),
		returns:        make(map[string]map[string]*SalesReturn),
	}
}

func (m *MemoryStore) AddProduct(tenantID string, product Product) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenantProducts(tenantID)[product.ID] = product
}

func (m *MemoryStore) AddRetailer(tenantID string, retailer Retailer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenantRetailers(tenantID)[retailer.ID] = retailer
}

func (m *MemoryStore) AddSupplier(tenantID string, supplier Supplier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenantSuppliers(tenantID)[supplier.ID] = supplier
}

func (m *MemoryStore) AddWarehouse(tenantID, warehouseID, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenantWarehouses(tenantID)[warehouseID] = name
}

func (m *MemoryStore) SetStock(tenantID, productID, warehouseID string, quantity int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onHand[stockKey(tenantID, productID, warehouseID)] = quantity
}

func (m *MemoryStore) SetStockValue(tenantID, productID, warehouseID string, quantity, valueCents int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onHand[stockKey(tenantID, productID, warehouseID)] = quantity
	m.inventoryCost[stockKey(tenantID, productID, warehouseID)] = valueCents
	if valueCents > 0 {
		m.ledger = append(m.ledger,
			memoryLedgerEntry{TenantID: tenantID, Account: "inventory", DebitCents: valueCents, ReferenceType: "opening_balance", ReferenceID: productID},
			memoryLedgerEntry{TenantID: tenantID, Account: "opening_equity", CreditCents: valueCents, ReferenceType: "opening_balance", ReferenceID: productID},
		)
	}
}

func (m *MemoryStore) ListProducts(_ context.Context, tenantID string) ([]Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Product, 0, len(m.products[tenantID]))
	for _, product := range m.products[tenantID] {
		out = append(out, product)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SKU < out[j].SKU })
	return out, nil
}

func (m *MemoryStore) CreateProduct(_ context.Context, tenantID string, input ProductInput) (Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	products := m.tenantProducts(tenantID)
	for _, product := range products {
		if product.SKU == input.SKU {
			return Product{}, fmt.Errorf("%w: sku %s already exists", ErrConflict, input.SKU)
		}
	}
	product := Product{
		ID:         m.newID(),
		SKU:        input.SKU,
		Name:       input.Name,
		Unit:       input.Unit,
		PriceCents: input.PriceCents,
	}
	products[product.ID] = product
	return product, nil
}

func (m *MemoryStore) UpdateProduct(_ context.Context, tenantID, productID string, input ProductInput) (Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	products := m.tenantProducts(tenantID)
	product, ok := products[productID]
	if !ok {
		return Product{}, fmt.Errorf("%w: product %s", ErrNotFound, productID)
	}
	for id, other := range products {
		if id != productID && other.SKU == input.SKU {
			return Product{}, fmt.Errorf("%w: sku %s already exists", ErrConflict, input.SKU)
		}
	}
	product.SKU = input.SKU
	product.Name = input.Name
	product.Unit = input.Unit
	product.PriceCents = input.PriceCents
	products[productID] = product
	return product, nil
}

func (m *MemoryStore) ListWarehouses(_ context.Context, tenantID string) ([]Warehouse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Warehouse, 0, len(m.warehouses[tenantID]))
	for id, name := range m.warehouses[tenantID] {
		out = append(out, Warehouse{ID: id, Name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *MemoryStore) CreateWarehouse(_ context.Context, tenantID string, input WarehouseInput) (Warehouse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	warehouse := Warehouse{ID: m.newID(), Name: input.Name}
	m.tenantWarehouses(tenantID)[warehouse.ID] = warehouse.Name
	return warehouse, nil
}

func (m *MemoryStore) UpdateWarehouse(_ context.Context, tenantID, warehouseID string, input WarehouseInput) (Warehouse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	warehouses := m.tenantWarehouses(tenantID)
	if _, ok := warehouses[warehouseID]; !ok {
		return Warehouse{}, fmt.Errorf("%w: warehouse %s", ErrNotFound, warehouseID)
	}
	warehouses[warehouseID] = input.Name
	return Warehouse{ID: warehouseID, Name: input.Name}, nil
}

func (m *MemoryStore) ListRetailers(_ context.Context, tenantID string) ([]Retailer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Retailer, 0, len(m.retailers[tenantID]))
	for _, retailer := range m.retailers[tenantID] {
		out = append(out, retailer)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *MemoryStore) CreateRetailer(_ context.Context, tenantID string, input RetailerInput) (Retailer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	retailer := Retailer{
		ID:               m.newID(),
		Name:             input.Name,
		CreditLimitCents: input.CreditLimitCents,
	}
	m.tenantRetailers(tenantID)[retailer.ID] = retailer
	return retailer, nil
}

func (m *MemoryStore) UpdateRetailer(_ context.Context, tenantID, retailerID string, input RetailerInput) (Retailer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	retailers := m.tenantRetailers(tenantID)
	retailer, ok := retailers[retailerID]
	if !ok {
		return Retailer{}, fmt.Errorf("%w: retailer %s", ErrNotFound, retailerID)
	}
	retailer.Name = input.Name
	retailer.CreditLimitCents = input.CreditLimitCents
	retailers[retailerID] = retailer
	return retailer, nil
}

func (m *MemoryStore) ListSuppliers(_ context.Context, tenantID string) ([]Supplier, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	suppliers := make([]Supplier, 0, len(m.tenantSuppliers(tenantID)))
	for _, supplier := range m.tenantSuppliers(tenantID) {
		suppliers = append(suppliers, supplier)
	}
	sort.Slice(suppliers, func(i, j int) bool {
		return suppliers[i].Name < suppliers[j].Name
	})
	return suppliers, nil
}

func (m *MemoryStore) CreateSupplier(_ context.Context, tenantID string, input SupplierInput) (Supplier, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	supplier := Supplier{
		ID:          m.newID(),
		Name:        input.Name,
		ContactName: input.ContactName,
		Phone:       input.Phone,
	}
	m.tenantSuppliers(tenantID)[supplier.ID] = supplier
	return supplier, nil
}

func (m *MemoryStore) UpdateSupplier(_ context.Context, tenantID, supplierID string, input SupplierInput) (Supplier, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	suppliers := m.tenantSuppliers(tenantID)
	supplier, ok := suppliers[supplierID]
	if !ok {
		return Supplier{}, fmt.Errorf("%w: supplier %s", ErrNotFound, supplierID)
	}
	supplier.Name = input.Name
	supplier.ContactName = input.ContactName
	supplier.Phone = input.Phone
	suppliers[supplierID] = supplier
	return supplier, nil
}

func (m *MemoryStore) CreatePurchaseOrder(_ context.Context, tenantID string, input CreatePurchaseOrderInput) (PurchaseOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tenantSuppliers(tenantID)[input.SupplierID]; !ok {
		return PurchaseOrder{}, fmt.Errorf("%w: supplier %s", ErrNotFound, input.SupplierID)
	}
	if _, ok := m.tenantWarehouses(tenantID)[input.WarehouseID]; !ok {
		return PurchaseOrder{}, fmt.Errorf("%w: warehouse %s", ErrNotFound, input.WarehouseID)
	}

	purchaseOrder := &PurchaseOrder{
		ID:          m.newID(),
		SupplierID:  input.SupplierID,
		WarehouseID: input.WarehouseID,
		Status:      "ordered",
		CreatedAt:   time.Now().UTC(),
	}
	for _, requestLine := range input.Lines {
		if _, ok := m.tenantProducts(tenantID)[requestLine.ProductID]; !ok {
			return PurchaseOrder{}, fmt.Errorf("%w: product %s", ErrNotFound, requestLine.ProductID)
		}
		lineTotal := int64(requestLine.Quantity) * requestLine.UnitCostCents
		purchaseOrder.Lines = append(purchaseOrder.Lines, PurchaseOrderLine{
			ID:             m.newID(),
			ProductID:      requestLine.ProductID,
			Quantity:       requestLine.Quantity,
			UnitCostCents:  requestLine.UnitCostCents,
			LineTotalCents: lineTotal,
		})
		purchaseOrder.TotalCents += lineTotal
	}
	m.tenantPurchaseOrders(tenantID)[purchaseOrder.ID] = purchaseOrder
	return clonePurchaseOrder(*purchaseOrder), nil
}

func (m *MemoryStore) ListPurchaseOrders(_ context.Context, tenantID string) ([]PurchaseOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]PurchaseOrder, 0, len(m.purchaseOrders[tenantID]))
	for _, purchaseOrder := range m.purchaseOrders[tenantID] {
		out = append(out, clonePurchaseOrder(*purchaseOrder))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (m *MemoryStore) GetPurchaseOrder(_ context.Context, tenantID, purchaseOrderID string) (PurchaseOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	purchaseOrder, ok := m.purchaseOrders[tenantID][purchaseOrderID]
	if !ok {
		return PurchaseOrder{}, fmt.Errorf("%w: purchase order %s", ErrNotFound, purchaseOrderID)
	}
	return clonePurchaseOrder(*purchaseOrder), nil
}

func (m *MemoryStore) ReceivePurchaseOrder(_ context.Context, tenantID, purchaseOrderID string) (PurchaseOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	purchaseOrder, ok := m.purchaseOrders[tenantID][purchaseOrderID]
	if !ok {
		return PurchaseOrder{}, fmt.Errorf("%w: purchase order %s", ErrNotFound, purchaseOrderID)
	}
	if purchaseOrder.Status != "ordered" {
		return PurchaseOrder{}, fmt.Errorf("%w: purchase order %s is %s, expected ordered", ErrConflict, purchaseOrderID, purchaseOrder.Status)
	}

	for _, line := range purchaseOrder.Lines {
		key := stockKey(tenantID, line.ProductID, purchaseOrder.WarehouseID)
		m.onHand[key] += int64(line.Quantity)
		m.inventoryCost[key] += line.LineTotalCents
	}
	if purchaseOrder.TotalCents > 0 {
		m.ledger = append(m.ledger,
			memoryLedgerEntry{TenantID: tenantID, Account: "inventory", DebitCents: purchaseOrder.TotalCents, ReferenceType: "purchase_receipt", ReferenceID: purchaseOrder.ID},
			memoryLedgerEntry{TenantID: tenantID, Account: "accounts_payable", CreditCents: purchaseOrder.TotalCents, ReferenceType: "purchase_receipt", ReferenceID: purchaseOrder.ID},
		)
	}
	now := time.Now().UTC()
	purchaseOrder.Status = "received"
	purchaseOrder.ReceivedAt = &now
	return clonePurchaseOrder(*purchaseOrder), nil
}

func (m *MemoryStore) CancelPurchaseOrder(_ context.Context, tenantID, purchaseOrderID string) (PurchaseOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	purchaseOrder, ok := m.purchaseOrders[tenantID][purchaseOrderID]
	if !ok {
		return PurchaseOrder{}, fmt.Errorf("%w: purchase order %s", ErrNotFound, purchaseOrderID)
	}
	if purchaseOrder.Status != "ordered" {
		return PurchaseOrder{}, fmt.Errorf("%w: purchase order %s is %s, expected ordered", ErrConflict, purchaseOrderID, purchaseOrder.Status)
	}
	purchaseOrder.Status = "cancelled"
	return clonePurchaseOrder(*purchaseOrder), nil
}

func (m *MemoryStore) ListStock(_ context.Context, tenantID string) ([]StockLevel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]StockLevel, 0)
	for productID := range m.products[tenantID] {
		for warehouseID := range m.warehouses[tenantID] {
			key := stockKey(tenantID, productID, warehouseID)
			onHand := m.onHand[key]
			reserved := m.reservedLocked(tenantID, productID, warehouseID)
			value := m.inventoryCost[key]
			level := StockLevel{
				ProductID:           productID,
				WarehouseID:         warehouseID,
				OnHand:              onHand,
				Reserved:            reserved,
				Available:           onHand - reserved,
				InventoryValueCents: value,
			}
			if onHand > 0 {
				level.AverageCostCents = value / onHand
			}
			out = append(out, level)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ProductID == out[j].ProductID {
			return out[i].WarehouseID < out[j].WarehouseID
		}
		return out[i].ProductID < out[j].ProductID
	})
	return out, nil
}

func (m *MemoryStore) ReceiveStock(_ context.Context, tenantID, productID, warehouseID string, quantity, unitCostCents int64, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.products[tenantID][productID]; !ok {
		return fmt.Errorf("%w: product %s", ErrNotFound, productID)
	}
	if _, ok := m.warehouses[tenantID][warehouseID]; !ok {
		return fmt.Errorf("%w: warehouse %s", ErrNotFound, warehouseID)
	}

	key := stockKey(tenantID, productID, warehouseID)
	if quantity > 0 {
		m.onHand[key] += quantity
		m.inventoryCost[key] += quantity * unitCostCents
		if quantity*unitCostCents > 0 {
			m.ledger = append(m.ledger,
				memoryLedgerEntry{TenantID: tenantID, Account: "inventory", DebitCents: quantity * unitCostCents, ReferenceType: "stock_receipt", ReferenceID: productID},
				memoryLedgerEntry{TenantID: tenantID, Account: "accounts_payable", CreditCents: quantity * unitCostCents, ReferenceType: "stock_receipt", ReferenceID: productID},
			)
		}
		return nil
	}

	available := m.onHand[key] - m.reservedLocked(tenantID, productID, warehouseID)
	if available+quantity < 0 {
		return fmt.Errorf("%w: stock adjustment would make reserved stock unavailable", ErrConflict)
	}
	cost, err := issueCost(m.inventoryCost[key], -quantity, m.onHand[key])
	if err != nil {
		return err
	}
	m.onHand[key] += quantity
	m.inventoryCost[key] -= cost
	if cost > 0 {
		m.ledger = append(m.ledger,
			memoryLedgerEntry{TenantID: tenantID, Account: "inventory_shrinkage", DebitCents: cost, ReferenceType: "stock_adjustment", ReferenceID: productID},
			memoryLedgerEntry{TenantID: tenantID, Account: "inventory", CreditCents: cost, ReferenceType: "stock_adjustment", ReferenceID: productID},
		)
	}
	_ = reason
	return nil
}

func (m *MemoryStore) CreateOrder(_ context.Context, tenantID, retailerID string, lines []CreateOrderLine) (Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.retailers[tenantID][retailerID]; !ok {
		return Order{}, fmt.Errorf("%w: retailer %s", ErrNotFound, retailerID)
	}

	order := Order{
		ID:         newID(),
		RetailerID: retailerID,
		Status:     "draft",
		CreatedAt:  time.Now().UTC(),
	}
	for _, line := range lines {
		product, ok := m.products[tenantID][line.ProductID]
		if !ok {
			return Order{}, fmt.Errorf("%w: product %s", ErrNotFound, line.ProductID)
		}
		lineTotal := int64(line.Quantity) * product.PriceCents
		order.Lines = append(order.Lines, OrderLine{
			ID:             newID(),
			ProductID:      line.ProductID,
			Quantity:       line.Quantity,
			UnitPriceCents: product.PriceCents,
			LineTotalCents: lineTotal,
		})
		order.TotalCents += lineTotal
	}

	m.tenantOrders(tenantID)[order.ID] = &order
	return cloneOrder(order), nil
}

func (m *MemoryStore) ListOrders(_ context.Context, tenantID string) ([]Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Order, 0, len(m.orders[tenantID]))
	for _, order := range m.orders[tenantID] {
		out = append(out, cloneOrder(*order))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (m *MemoryStore) GetOrder(_ context.Context, tenantID, orderID string) (Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, ok := m.orders[tenantID][orderID]
	if !ok {
		return Order{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
	}
	return cloneOrder(*order), nil
}

func (m *MemoryStore) ReserveOrder(_ context.Context, tenantID, orderID string) (Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, ok := m.orders[tenantID][orderID]
	if !ok {
		return Order{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
	}
	if order.Status != "draft" {
		return Order{}, fmt.Errorf("%w: order %s is %s, expected draft", ErrConflict, orderID, order.Status)
	}

	retailer, ok := m.retailers[tenantID][order.RetailerID]
	if !ok {
		return Order{}, fmt.Errorf("%w: retailer %s", ErrNotFound, order.RetailerID)
	}
	if m.exposureLocked(tenantID, order.RetailerID)+order.TotalCents > retailer.CreditLimitCents {
		return Order{}, fmt.Errorf("%w: credit limit exceeded for retailer %s", ErrConflict, order.RetailerID)
	}

	warehouseID, ok := m.defaultWarehouseLocked(tenantID)
	if !ok {
		return Order{}, fmt.Errorf("%w: no warehouse for tenant", ErrNotFound)
	}

	for _, line := range order.Lines {
		available := m.onHand[stockKey(tenantID, line.ProductID, warehouseID)] -
			m.reservedLocked(tenantID, line.ProductID, warehouseID)
		if available < int64(line.Quantity) {
			return Order{}, fmt.Errorf("%w: insufficient stock for product %s", ErrConflict, line.ProductID)
		}
	}

	for _, line := range order.Lines {
		reservation := &memoryReservation{
			ID:          newID(),
			TenantID:    tenantID,
			OrderID:     order.ID,
			ProductID:   line.ProductID,
			WarehouseID: warehouseID,
			Quantity:    int64(line.Quantity),
			Status:      "active",
		}
		m.reservations[reservation.ID] = reservation
	}
	order.Status = "reserved"
	return cloneOrder(*order), nil
}

func (m *MemoryStore) CancelOrder(_ context.Context, tenantID, orderID string) (Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, ok := m.orders[tenantID][orderID]
	if !ok {
		return Order{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
	}

	switch order.Status {
	case "cancelled":
		return cloneOrder(*order), nil
	case "draft":
		order.Status = "cancelled"
	case "reserved":
		for _, reservation := range m.reservations {
			if reservation.TenantID == tenantID && reservation.OrderID == order.ID && reservation.Status == "active" {
				reservation.Status = "released"
			}
		}
		for _, note := range m.deliveryNotes[tenantID] {
			if note.OrderID == order.ID && note.Status == "picking" {
				note.Status = "cancelled"
			}
		}
		order.Status = "cancelled"
	default:
		return Order{}, fmt.Errorf("%w: order %s is %s and cannot be cancelled", ErrConflict, orderID, order.Status)
	}
	return cloneOrder(*order), nil
}

func (m *MemoryStore) CreateDeliveryNote(_ context.Context, tenantID, orderID string) (DeliveryNote, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, ok := m.orders[tenantID][orderID]
	if !ok {
		return DeliveryNote{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
	}
	if order.Status != "reserved" {
		return DeliveryNote{}, fmt.Errorf("%w: order %s is %s, expected reserved", ErrConflict, orderID, order.Status)
	}
	for _, note := range m.deliveryNotes[tenantID] {
		if note.OrderID == orderID && note.Status != "cancelled" {
			return DeliveryNote{}, fmt.Errorf("%w: order %s already has a delivery note", ErrConflict, orderID)
		}
	}

	reservations := make([]*memoryReservation, 0)
	for _, reservation := range m.reservations {
		if reservation.TenantID == tenantID && reservation.OrderID == orderID && reservation.Status == "active" {
			reservations = append(reservations, reservation)
		}
	}
	if len(reservations) == 0 {
		return DeliveryNote{}, fmt.Errorf("%w: order %s has no active reservations", ErrConflict, orderID)
	}
	sort.Slice(reservations, func(i, j int) bool {
		if reservations[i].ProductID == reservations[j].ProductID {
			return reservations[i].WarehouseID < reservations[j].WarehouseID
		}
		return reservations[i].ProductID < reservations[j].ProductID
	})

	orderLines := make(map[string]OrderLine, len(order.Lines))
	for _, line := range order.Lines {
		orderLines[line.ProductID] = line
	}

	note := &DeliveryNote{
		ID:         m.newID(),
		OrderID:    order.ID,
		RetailerID: order.RetailerID,
		Status:     "picking",
		CreatedAt:  time.Now().UTC(),
	}
	for _, reservation := range reservations {
		line, ok := orderLines[reservation.ProductID]
		if !ok || int64(line.Quantity) != reservation.Quantity {
			return DeliveryNote{}, fmt.Errorf("%w: reservation does not match order line for product %s", ErrConflict, reservation.ProductID)
		}
		note.Lines = append(note.Lines, DeliveryNoteLine{
			ID:             m.newID(),
			DeliveryNoteID: note.ID,
			OrderLineID:    line.ID,
			ProductID:      reservation.ProductID,
			WarehouseID:    reservation.WarehouseID,
			Quantity:       int(reservation.Quantity),
			UnitPriceCents: line.UnitPriceCents,
			LineTotalCents: int64(reservation.Quantity) * line.UnitPriceCents,
		})
	}
	m.tenantDeliveryNotes(tenantID)[note.ID] = note
	return cloneDeliveryNote(*note), nil
}

func (m *MemoryStore) ListDeliveryNotes(_ context.Context, tenantID string) ([]DeliveryNote, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]DeliveryNote, 0, len(m.deliveryNotes[tenantID]))
	for _, note := range m.deliveryNotes[tenantID] {
		out = append(out, cloneDeliveryNote(*note))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (m *MemoryStore) GetDeliveryNote(_ context.Context, tenantID, deliveryNoteID string) (DeliveryNote, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	note, ok := m.deliveryNotes[tenantID][deliveryNoteID]
	if !ok {
		return DeliveryNote{}, fmt.Errorf("%w: delivery note %s", ErrNotFound, deliveryNoteID)
	}
	return cloneDeliveryNote(*note), nil
}

func (m *MemoryStore) GetDeliveryNoteForOrder(_ context.Context, tenantID, orderID string) (DeliveryNote, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var found *DeliveryNote
	for _, note := range m.deliveryNotes[tenantID] {
		if note.OrderID != orderID || note.Status == "cancelled" {
			continue
		}
		if found == nil || note.CreatedAt.After(found.CreatedAt) || (note.CreatedAt.Equal(found.CreatedAt) && note.ID > found.ID) {
			found = note
		}
	}
	if found == nil {
		return DeliveryNote{}, fmt.Errorf("%w: delivery note for order %s", ErrNotFound, orderID)
	}
	return cloneDeliveryNote(*found), nil
}

func (m *MemoryStore) DispatchDeliveryNote(_ context.Context, tenantID, deliveryNoteID string) (DeliveryNote, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	note, ok := m.deliveryNotes[tenantID][deliveryNoteID]
	if !ok {
		return DeliveryNote{}, fmt.Errorf("%w: delivery note %s", ErrNotFound, deliveryNoteID)
	}
	if note.Status != "picking" {
		return DeliveryNote{}, fmt.Errorf("%w: delivery note %s is %s, expected picking", ErrConflict, deliveryNoteID, note.Status)
	}
	order, ok := m.orders[tenantID][note.OrderID]
	if !ok {
		return DeliveryNote{}, fmt.Errorf("%w: order %s", ErrNotFound, note.OrderID)
	}
	if order.Status != "reserved" {
		return DeliveryNote{}, fmt.Errorf("%w: order %s is %s, expected reserved", ErrConflict, note.OrderID, order.Status)
	}

	var cogs int64
	for i := range note.Lines {
		line := &note.Lines[i]
		key := stockKey(tenantID, line.ProductID, line.WarehouseID)
		cost, err := issueCost(m.inventoryCost[key], int64(line.Quantity), m.onHand[key])
		if err != nil {
			return DeliveryNote{}, err
		}
		var reservation *memoryReservation
		for _, candidate := range m.reservations {
			if candidate.TenantID == tenantID &&
				candidate.OrderID == order.ID &&
				candidate.ProductID == line.ProductID &&
				candidate.WarehouseID == line.WarehouseID &&
				candidate.Status == "active" {
				reservation = candidate
				break
			}
		}
		if reservation == nil || reservation.Quantity != int64(line.Quantity) {
			return DeliveryNote{}, fmt.Errorf("%w: missing reservation for product %s", ErrConflict, line.ProductID)
		}
		reservation.Status = "fulfilled"
		m.onHand[key] -= int64(line.Quantity)
		m.inventoryCost[key] -= cost
		line.TotalCostCents = cost
		note.TotalCostCents += cost
		cogs += cost
	}

	now := time.Now().UTC()
	note.Status = "dispatched"
	note.DispatchedAt = &now
	order.Status = "dispatched"
	if cogs > 0 {
		m.ledger = append(m.ledger,
			memoryLedgerEntry{TenantID: tenantID, Account: "cost_of_goods_sold", DebitCents: cogs, ReferenceType: "delivery_note", ReferenceID: note.ID},
			memoryLedgerEntry{TenantID: tenantID, Account: "inventory", CreditCents: cogs, ReferenceType: "delivery_note", ReferenceID: note.ID},
		)
	}
	return cloneDeliveryNote(*note), nil
}

func (m *MemoryStore) CancelDeliveryNote(_ context.Context, tenantID, deliveryNoteID string) (DeliveryNote, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	note, ok := m.deliveryNotes[tenantID][deliveryNoteID]
	if !ok {
		return DeliveryNote{}, fmt.Errorf("%w: delivery note %s", ErrNotFound, deliveryNoteID)
	}
	if note.Status != "picking" {
		return DeliveryNote{}, fmt.Errorf("%w: delivery note %s is %s, expected picking", ErrConflict, deliveryNoteID, note.Status)
	}
	note.Status = "cancelled"
	return cloneDeliveryNote(*note), nil
}

func (m *MemoryStore) InvoiceOrder(_ context.Context, tenantID, orderID string) (Invoice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, ok := m.orders[tenantID][orderID]
	if !ok {
		return Invoice{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
	}
	if order.Status != "dispatched" {
		return Invoice{}, fmt.Errorf("%w: order %s is %s, expected dispatched", ErrConflict, orderID, order.Status)
	}

	var note *DeliveryNote
	for _, candidate := range m.deliveryNotes[tenantID] {
		if candidate.OrderID == orderID && candidate.Status == "dispatched" {
			note = candidate
			break
		}
	}
	if note == nil {
		return Invoice{}, fmt.Errorf("%w: order %s has no dispatched delivery note", ErrConflict, orderID)
	}

	invoice := Invoice{
		ID:         newID(),
		OrderID:    order.ID,
		RetailerID: order.RetailerID,
		Status:     "open",
		TotalCents: order.TotalCents,
		CreatedAt:  time.Now().UTC(),
	}
	lines := make([]InvoiceLine, 0, len(note.Lines))
	var invoiceTotal int64
	for _, noteLine := range note.Lines {
		line := InvoiceLine{
			ID:             newID(),
			InvoiceID:      invoice.ID,
			ProductID:      noteLine.ProductID,
			WarehouseID:    noteLine.WarehouseID,
			Quantity:       noteLine.Quantity,
			UnitPriceCents: noteLine.UnitPriceCents,
			LineTotalCents: noteLine.LineTotalCents,
			TotalCostCents: noteLine.TotalCostCents,
		}
		invoiceTotal += line.LineTotalCents
		lines = append(lines, line)
	}
	if invoiceTotal != order.TotalCents {
		return Invoice{}, fmt.Errorf("%w: delivery note total does not match order %s", ErrConflict, orderID)
	}
	invoice.Lines = lines
	m.invoiceLines[invoice.ID] = lines
	m.tenantInvoices(tenantID)[invoice.ID] = &invoice
	order.Status = "invoiced"

	m.ledger = append(m.ledger,
		memoryLedgerEntry{TenantID: tenantID, Account: "accounts_receivable", DebitCents: invoice.TotalCents, ReferenceType: "invoice", ReferenceID: invoice.ID},
		memoryLedgerEntry{TenantID: tenantID, Account: "revenue", CreditCents: invoice.TotalCents, ReferenceType: "invoice", ReferenceID: invoice.ID},
	)
	return invoice, nil
}

func (m *MemoryStore) ListInvoices(_ context.Context, tenantID string) ([]Invoice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Invoice, 0, len(m.invoices[tenantID]))
	for _, invoice := range m.invoices[tenantID] {
		out = append(out, cloneInvoice(*invoice))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (m *MemoryStore) GetInvoice(_ context.Context, tenantID, invoiceID string) (InvoiceDetail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	invoice, ok := m.invoices[tenantID][invoiceID]
	if !ok {
		return InvoiceDetail{}, fmt.Errorf("%w: invoice %s", ErrNotFound, invoiceID)
	}
	detail := InvoiceDetail{Invoice: cloneInvoice(*invoice)}
	for _, salesReturn := range m.returns[tenantID] {
		if salesReturn.InvoiceID == invoiceID {
			detail.Returns = append(detail.Returns, cloneSalesReturn(*salesReturn))
		}
	}
	sort.Slice(detail.Returns, func(i, j int) bool {
		if detail.Returns[i].CreatedAt.Equal(detail.Returns[j].CreatedAt) {
			return detail.Returns[i].ID > detail.Returns[j].ID
		}
		return detail.Returns[i].CreatedAt.After(detail.Returns[j].CreatedAt)
	})
	for _, payment := range m.payments[tenantID] {
		if payment.InvoiceID == invoiceID {
			detail.Payments = append(detail.Payments, *payment)
		}
	}
	sort.Slice(detail.Payments, func(i, j int) bool {
		if detail.Payments[i].CreatedAt.Equal(detail.Payments[j].CreatedAt) {
			return detail.Payments[i].ID > detail.Payments[j].ID
		}
		return detail.Payments[i].CreatedAt.After(detail.Payments[j].CreatedAt)
	})
	return detail, nil
}

func (m *MemoryStore) GetCredit(_ context.Context, tenantID, retailerID string) (CreditView, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	retailer, ok := m.retailers[tenantID][retailerID]
	if !ok {
		return CreditView{}, fmt.Errorf("%w: retailer %s", ErrNotFound, retailerID)
	}

	view := CreditView{
		RetailerID:       retailerID,
		CreditLimitCents: retailer.CreditLimitCents,
	}
	for _, invoice := range m.invoices[tenantID] {
		if invoice.RetailerID == retailerID && invoice.Status != "void" {
			view.OpenInvoiceCents += invoice.TotalCents - invoice.PaidCents - invoice.ReturnedCents
		}
	}
	for _, order := range m.orders[tenantID] {
		if order.RetailerID == retailerID && (order.Status == "reserved" || order.Status == "dispatched") {
			view.ReservedOrderCents += order.TotalCents
		}
	}
	view.ExposureCents = view.OpenInvoiceCents + view.ReservedOrderCents
	view.AvailableCents = view.CreditLimitCents - view.ExposureCents
	return view, nil
}

func (m *MemoryStore) RecordPayment(_ context.Context, tenantID, invoiceID string, amountCents int64) (Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	invoice, ok := m.invoices[tenantID][invoiceID]
	if !ok {
		return Payment{}, fmt.Errorf("%w: invoice %s", ErrNotFound, invoiceID)
	}
	if invoice.Status != "open" {
		return Payment{}, fmt.Errorf("%w: invoice %s is %s", ErrConflict, invoiceID, invoice.Status)
	}

	remaining := invoice.TotalCents - invoice.PaidCents - invoice.ReturnedCents
	if amountCents > remaining {
		return Payment{}, fmt.Errorf("%w: payment exceeds remaining invoice balance", ErrConflict)
	}

	payment := Payment{
		ID:          newID(),
		InvoiceID:   invoice.ID,
		AmountCents: amountCents,
		CreatedAt:   time.Now().UTC(),
	}
	m.tenantPayments(tenantID)[payment.ID] = &payment

	invoice.PaidCents += amountCents
	if invoice.PaidCents+invoice.ReturnedCents >= invoice.TotalCents {
		invoice.Status = "paid"
	}

	m.ledger = append(m.ledger,
		memoryLedgerEntry{TenantID: tenantID, Account: "cash", DebitCents: amountCents, ReferenceType: "payment", ReferenceID: payment.ID},
		memoryLedgerEntry{TenantID: tenantID, Account: "accounts_receivable", CreditCents: amountCents, ReferenceType: "payment", ReferenceID: payment.ID},
	)
	return payment, nil
}

func (m *MemoryStore) ReturnInvoice(_ context.Context, tenantID, invoiceID string, lines []CreateReturnLine) (SalesReturn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	invoice, ok := m.invoices[tenantID][invoiceID]
	if !ok {
		return SalesReturn{}, fmt.Errorf("%w: invoice %s", ErrNotFound, invoiceID)
	}
	if invoice.Status == "void" {
		return SalesReturn{}, fmt.Errorf("%w: invoice %s is void", ErrConflict, invoiceID)
	}
	invoiceLines := m.invoiceLines[invoiceID]
	if len(invoiceLines) == 0 {
		return SalesReturn{}, fmt.Errorf("%w: invoice %s has no lines", ErrConflict, invoiceID)
	}

	byProduct := make(map[string]InvoiceLine, len(invoiceLines))
	warehouseByLineID := make(map[string]string, len(invoiceLines))
	for _, line := range invoiceLines {
		byProduct[line.ProductID] = line
		warehouseByLineID[line.ID] = line.WarehouseID
	}
	returnedQuantity := make(map[string]int, len(invoiceLines))
	returnedCost := make(map[string]int64, len(invoiceLines))
	for _, existing := range m.returns[tenantID] {
		if existing.InvoiceID != invoiceID {
			continue
		}
		for _, line := range existing.Lines {
			returnedQuantity[line.InvoiceLineID] += line.Quantity
			returnedCost[line.InvoiceLineID] += line.CostCents
		}
	}

	created := SalesReturn{
		ID:         newID(),
		InvoiceID:  invoiceID,
		RetailerID: invoice.RetailerID,
		CreatedAt:  time.Now().UTC(),
	}
	seen := make(map[string]struct{}, len(lines))
	for _, request := range lines {
		if request.ProductID == "" || request.Quantity <= 0 {
			return SalesReturn{}, fmt.Errorf("%w: product_id and positive quantity are required", ErrInvalid)
		}
		if _, duplicate := seen[request.ProductID]; duplicate {
			return SalesReturn{}, fmt.Errorf("%w: duplicate product_id %s", ErrInvalid, request.ProductID)
		}
		seen[request.ProductID] = struct{}{}

		invoiceLine, ok := byProduct[request.ProductID]
		if !ok {
			return SalesReturn{}, fmt.Errorf("%w: product %s was not invoiced", ErrInvalid, request.ProductID)
		}
		remainingQuantity := invoiceLine.Quantity - returnedQuantity[invoiceLine.ID]
		if request.Quantity > remainingQuantity {
			return SalesReturn{}, fmt.Errorf("%w: return quantity exceeds remaining invoice quantity for product %s", ErrConflict, request.ProductID)
		}
		remainingCost := invoiceLine.TotalCostCents - returnedCost[invoiceLine.ID]
		cost, err := issueCost(remainingCost, int64(request.Quantity), int64(remainingQuantity))
		if err != nil {
			return SalesReturn{}, err
		}
		line := SalesReturnLine{
			ID:            newID(),
			SalesReturnID: created.ID,
			InvoiceLineID: invoiceLine.ID,
			ProductID:     request.ProductID,
			Quantity:      request.Quantity,
			RefundCents:   int64(request.Quantity) * invoiceLine.UnitPriceCents,
			CostCents:     cost,
			Restock:       request.Restock,
		}
		created.Lines = append(created.Lines, line)
		created.TotalRefundCents += line.RefundCents
		created.TotalCostCents += line.CostCents
	}

	var restockCost int64
	for _, line := range created.Lines {
		if !line.Restock {
			continue
		}
		key := stockKey(tenantID, line.ProductID, warehouseByLineID[line.InvoiceLineID])
		m.onHand[key] += int64(line.Quantity)
		m.inventoryCost[key] += line.CostCents
		restockCost += line.CostCents
	}

	invoice.ReturnedCents += created.TotalRefundCents
	if invoice.Status != "paid" && invoice.PaidCents+invoice.ReturnedCents >= invoice.TotalCents {
		invoice.Status = "paid"
	}
	m.tenantReturns(tenantID)[created.ID] = &created

	if created.TotalRefundCents > 0 {
		m.ledger = append(m.ledger,
			memoryLedgerEntry{TenantID: tenantID, Account: "sales_returns", DebitCents: created.TotalRefundCents, ReferenceType: "sales_return", ReferenceID: created.ID},
			memoryLedgerEntry{TenantID: tenantID, Account: "accounts_receivable", CreditCents: created.TotalRefundCents, ReferenceType: "sales_return", ReferenceID: created.ID},
		)
	}
	if restockCost > 0 {
		m.ledger = append(m.ledger,
			memoryLedgerEntry{TenantID: tenantID, Account: "inventory", DebitCents: restockCost, ReferenceType: "sales_return", ReferenceID: created.ID},
			memoryLedgerEntry{TenantID: tenantID, Account: "cost_of_goods_sold", CreditCents: restockCost, ReferenceType: "sales_return", ReferenceID: created.ID},
		)
	}
	return created, nil
}

func (m *MemoryStore) Valuation(_ context.Context, tenantID string) (InventoryValuation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	valuation := InventoryValuation{Items: make([]StockLevel, 0)}
	for productID := range m.products[tenantID] {
		for warehouseID := range m.warehouses[tenantID] {
			key := stockKey(tenantID, productID, warehouseID)
			onHand := m.onHand[key]
			reserved := m.reservedLocked(tenantID, productID, warehouseID)
			value := m.inventoryCost[key]
			level := StockLevel{
				ProductID:           productID,
				WarehouseID:         warehouseID,
				OnHand:              onHand,
				Reserved:            reserved,
				Available:           onHand - reserved,
				InventoryValueCents: value,
			}
			if onHand > 0 {
				level.AverageCostCents = value / onHand
			}
			valuation.Items = append(valuation.Items, level)
			valuation.TotalValueCents += value
		}
	}
	sort.Slice(valuation.Items, func(i, j int) bool {
		if valuation.Items[i].ProductID == valuation.Items[j].ProductID {
			return valuation.Items[i].WarehouseID < valuation.Items[j].WarehouseID
		}
		return valuation.Items[i].ProductID < valuation.Items[j].ProductID
	})

	for _, entry := range m.ledger {
		if entry.TenantID != tenantID {
			continue
		}
		switch entry.Account {
		case "inventory":
			valuation.LedgerInventoryCents += entry.DebitCents - entry.CreditCents
		case "cost_of_goods_sold":
			valuation.CogsCents += entry.DebitCents - entry.CreditCents
		}
	}
	valuation.Balanced = valuation.TotalValueCents == valuation.LedgerInventoryCents
	return valuation, nil
}

func (m *MemoryStore) TrialBalance(_ context.Context, tenantID string) ([]AccountBalance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	byAccount := make(map[string]*AccountBalance)
	for _, entry := range m.ledger {
		if entry.TenantID != tenantID {
			continue
		}
		balance, ok := byAccount[entry.Account]
		if !ok {
			balance = &AccountBalance{Account: entry.Account}
			byAccount[entry.Account] = balance
		}
		balance.DebitCents += entry.DebitCents
		balance.CreditCents += entry.CreditCents
		balance.BalanceCents = balance.DebitCents - balance.CreditCents
	}

	accounts := make([]string, 0, len(byAccount))
	for account := range byAccount {
		accounts = append(accounts, account)
	}
	sort.Strings(accounts)

	out := make([]AccountBalance, 0, len(accounts))
	for _, account := range accounts {
		out = append(out, *byAccount[account])
	}
	return out, nil
}

func (m *MemoryStore) exposureLocked(tenantID, retailerID string) int64 {
	var exposure int64
	for _, invoice := range m.invoices[tenantID] {
		if invoice.RetailerID == retailerID && invoice.Status != "void" {
			exposure += invoice.TotalCents - invoice.PaidCents - invoice.ReturnedCents
		}
	}
	for _, order := range m.orders[tenantID] {
		if order.RetailerID == retailerID && (order.Status == "reserved" || order.Status == "dispatched") {
			exposure += order.TotalCents
		}
	}
	return exposure
}

func (m *MemoryStore) reservedLocked(tenantID, productID, warehouseID string) int64 {
	var reserved int64
	for _, reservation := range m.reservations {
		if reservation.TenantID == tenantID &&
			reservation.ProductID == productID &&
			reservation.WarehouseID == warehouseID &&
			reservation.Status == "active" {
			reserved += reservation.Quantity
		}
	}
	return reserved
}

func (m *MemoryStore) defaultWarehouseLocked(tenantID string) (string, bool) {
	ids := make([]string, 0, len(m.warehouses[tenantID]))
	for id := range m.warehouses[tenantID] {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return "", false
	}
	sort.Strings(ids)
	return ids[0], true
}

func (m *MemoryStore) tenantProducts(tenantID string) map[string]Product {
	if m.products[tenantID] == nil {
		m.products[tenantID] = make(map[string]Product)
	}
	return m.products[tenantID]
}

func (m *MemoryStore) tenantRetailers(tenantID string) map[string]Retailer {
	if m.retailers[tenantID] == nil {
		m.retailers[tenantID] = make(map[string]Retailer)
	}
	return m.retailers[tenantID]
}

func (m *MemoryStore) tenantSuppliers(tenantID string) map[string]Supplier {
	if m.suppliers[tenantID] == nil {
		m.suppliers[tenantID] = make(map[string]Supplier)
	}
	return m.suppliers[tenantID]
}

func (m *MemoryStore) tenantPurchaseOrders(tenantID string) map[string]*PurchaseOrder {
	if m.purchaseOrders[tenantID] == nil {
		m.purchaseOrders[tenantID] = make(map[string]*PurchaseOrder)
	}
	return m.purchaseOrders[tenantID]
}

func (m *MemoryStore) tenantWarehouses(tenantID string) map[string]string {
	if m.warehouses[tenantID] == nil {
		m.warehouses[tenantID] = make(map[string]string)
	}
	return m.warehouses[tenantID]
}

func (m *MemoryStore) tenantOrders(tenantID string) map[string]*Order {
	if m.orders[tenantID] == nil {
		m.orders[tenantID] = make(map[string]*Order)
	}
	return m.orders[tenantID]
}

func (m *MemoryStore) tenantDeliveryNotes(tenantID string) map[string]*DeliveryNote {
	if m.deliveryNotes[tenantID] == nil {
		m.deliveryNotes[tenantID] = make(map[string]*DeliveryNote)
	}
	return m.deliveryNotes[tenantID]
}

func (m *MemoryStore) tenantInvoices(tenantID string) map[string]*Invoice {
	if m.invoices[tenantID] == nil {
		m.invoices[tenantID] = make(map[string]*Invoice)
	}
	return m.invoices[tenantID]
}

func (m *MemoryStore) tenantPayments(tenantID string) map[string]*Payment {
	if m.payments[tenantID] == nil {
		m.payments[tenantID] = make(map[string]*Payment)
	}
	return m.payments[tenantID]
}

func (m *MemoryStore) tenantReturns(tenantID string) map[string]*SalesReturn {
	if m.returns[tenantID] == nil {
		m.returns[tenantID] = make(map[string]*SalesReturn)
	}
	return m.returns[tenantID]
}

func stockKey(tenantID, productID, warehouseID string) string {
	return tenantID + "|" + productID + "|" + warehouseID
}

func cloneOrder(order Order) Order {
	order.Lines = append([]OrderLine(nil), order.Lines...)
	return order
}

func cloneDeliveryNote(note DeliveryNote) DeliveryNote {
	note.Lines = append([]DeliveryNoteLine(nil), note.Lines...)
	if note.DispatchedAt != nil {
		dispatchedAt := *note.DispatchedAt
		note.DispatchedAt = &dispatchedAt
	}
	return note
}

func cloneInvoice(invoice Invoice) Invoice {
	invoice.Lines = append([]InvoiceLine(nil), invoice.Lines...)
	return invoice
}

func cloneSalesReturn(salesReturn SalesReturn) SalesReturn {
	salesReturn.Lines = append([]SalesReturnLine(nil), salesReturn.Lines...)
	return salesReturn
}

func clonePurchaseOrder(purchaseOrder PurchaseOrder) PurchaseOrder {
	purchaseOrder.Lines = append([]PurchaseOrderLine(nil), purchaseOrder.Lines...)
	if purchaseOrder.ReceivedAt != nil {
		receivedAt := *purchaseOrder.ReceivedAt
		purchaseOrder.ReceivedAt = &receivedAt
	}
	return purchaseOrder
}

func (m *MemoryStore) newID() string {
	m.nextID++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", m.nextID)
}
