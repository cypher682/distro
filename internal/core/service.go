package core

import (
	"context"
	"fmt"
	"strings"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListProducts(ctx context.Context, tenantID string) ([]Product, error) {
	if err := requireTenant(tenantID); err != nil {
		return nil, err
	}
	return s.store.ListProducts(ctx, tenantID)
}

func (s *Service) CreateProduct(ctx context.Context, tenantID string, input ProductInput) (Product, error) {
	if err := requireTenant(tenantID); err != nil {
		return Product{}, err
	}
	clean, err := cleanProductInput(input)
	if err != nil {
		return Product{}, err
	}
	return s.store.CreateProduct(ctx, tenantID, clean)
}

func (s *Service) UpdateProduct(ctx context.Context, tenantID, productID string, input ProductInput) (Product, error) {
	if err := requireTenant(tenantID); err != nil {
		return Product{}, err
	}
	if productID == "" {
		return Product{}, fmt.Errorf("%w: product_id is required", ErrInvalid)
	}
	clean, err := cleanProductInput(input)
	if err != nil {
		return Product{}, err
	}
	return s.store.UpdateProduct(ctx, tenantID, productID, clean)
}

func (s *Service) ListWarehouses(ctx context.Context, tenantID string) ([]Warehouse, error) {
	if err := requireTenant(tenantID); err != nil {
		return nil, err
	}
	return s.store.ListWarehouses(ctx, tenantID)
}

func (s *Service) CreateWarehouse(ctx context.Context, tenantID string, input WarehouseInput) (Warehouse, error) {
	if err := requireTenant(tenantID); err != nil {
		return Warehouse{}, err
	}
	clean, err := cleanWarehouseInput(input)
	if err != nil {
		return Warehouse{}, err
	}
	return s.store.CreateWarehouse(ctx, tenantID, clean)
}

func (s *Service) UpdateWarehouse(ctx context.Context, tenantID, warehouseID string, input WarehouseInput) (Warehouse, error) {
	if err := requireTenant(tenantID); err != nil {
		return Warehouse{}, err
	}
	if warehouseID == "" {
		return Warehouse{}, fmt.Errorf("%w: warehouse_id is required", ErrInvalid)
	}
	clean, err := cleanWarehouseInput(input)
	if err != nil {
		return Warehouse{}, err
	}
	return s.store.UpdateWarehouse(ctx, tenantID, warehouseID, clean)
}

func (s *Service) ListRetailers(ctx context.Context, tenantID string) ([]Retailer, error) {
	if err := requireTenant(tenantID); err != nil {
		return nil, err
	}
	return s.store.ListRetailers(ctx, tenantID)
}

func (s *Service) CreateRetailer(ctx context.Context, tenantID string, input RetailerInput) (Retailer, error) {
	if err := requireTenant(tenantID); err != nil {
		return Retailer{}, err
	}
	clean, err := cleanRetailerInput(input)
	if err != nil {
		return Retailer{}, err
	}
	return s.store.CreateRetailer(ctx, tenantID, clean)
}

func (s *Service) UpdateRetailer(ctx context.Context, tenantID, retailerID string, input RetailerInput) (Retailer, error) {
	if err := requireTenant(tenantID); err != nil {
		return Retailer{}, err
	}
	if retailerID == "" {
		return Retailer{}, fmt.Errorf("%w: retailer_id is required", ErrInvalid)
	}
	clean, err := cleanRetailerInput(input)
	if err != nil {
		return Retailer{}, err
	}
	return s.store.UpdateRetailer(ctx, tenantID, retailerID, clean)
}

func (s *Service) ListSuppliers(ctx context.Context, tenantID string) ([]Supplier, error) {
	if err := requireTenant(tenantID); err != nil {
		return nil, err
	}
	return s.store.ListSuppliers(ctx, tenantID)
}

func (s *Service) CreateSupplier(ctx context.Context, tenantID string, input SupplierInput) (Supplier, error) {
	if err := requireTenant(tenantID); err != nil {
		return Supplier{}, err
	}
	clean, err := cleanSupplierInput(input)
	if err != nil {
		return Supplier{}, err
	}
	return s.store.CreateSupplier(ctx, tenantID, clean)
}

func (s *Service) UpdateSupplier(ctx context.Context, tenantID, supplierID string, input SupplierInput) (Supplier, error) {
	if err := requireTenant(tenantID); err != nil {
		return Supplier{}, err
	}
	if supplierID == "" {
		return Supplier{}, fmt.Errorf("%w: supplier_id is required", ErrInvalid)
	}
	clean, err := cleanSupplierInput(input)
	if err != nil {
		return Supplier{}, err
	}
	return s.store.UpdateSupplier(ctx, tenantID, supplierID, clean)
}

func (s *Service) CreatePurchaseOrder(ctx context.Context, tenantID string, input CreatePurchaseOrderInput) (PurchaseOrder, error) {
	if err := requireTenant(tenantID); err != nil {
		return PurchaseOrder{}, err
	}
	if input.SupplierID == "" {
		return PurchaseOrder{}, fmt.Errorf("%w: supplier_id is required", ErrInvalid)
	}
	if input.WarehouseID == "" {
		return PurchaseOrder{}, fmt.Errorf("%w: warehouse_id is required", ErrInvalid)
	}
	if len(input.Lines) == 0 {
		return PurchaseOrder{}, fmt.Errorf("%w: at least one purchase order line is required", ErrInvalid)
	}

	seen := make(map[string]struct{}, len(input.Lines))
	lines := make([]CreatePurchaseOrderLine, 0, len(input.Lines))
	for _, line := range input.Lines {
		if line.ProductID == "" {
			return PurchaseOrder{}, fmt.Errorf("%w: product_id is required", ErrInvalid)
		}
		if line.Quantity <= 0 {
			return PurchaseOrder{}, fmt.Errorf("%w: quantity must be positive", ErrInvalid)
		}
		if line.UnitCostCents < 0 {
			return PurchaseOrder{}, fmt.Errorf("%w: unit_cost_cents must not be negative", ErrInvalid)
		}
		if _, ok := seen[line.ProductID]; ok {
			return PurchaseOrder{}, fmt.Errorf("%w: duplicate product_id %s", ErrInvalid, line.ProductID)
		}
		seen[line.ProductID] = struct{}{}
		lines = append(lines, line)
	}
	input.Lines = lines
	return s.store.CreatePurchaseOrder(ctx, tenantID, input)
}

func (s *Service) ListPurchaseOrders(ctx context.Context, tenantID string) ([]PurchaseOrder, error) {
	if err := requireTenant(tenantID); err != nil {
		return nil, err
	}
	return s.store.ListPurchaseOrders(ctx, tenantID)
}

func (s *Service) GetPurchaseOrder(ctx context.Context, tenantID, purchaseOrderID string) (PurchaseOrder, error) {
	if err := requireTenant(tenantID); err != nil {
		return PurchaseOrder{}, err
	}
	if purchaseOrderID == "" {
		return PurchaseOrder{}, fmt.Errorf("%w: purchase_order_id is required", ErrInvalid)
	}
	return s.store.GetPurchaseOrder(ctx, tenantID, purchaseOrderID)
}

func (s *Service) ReceivePurchaseOrder(ctx context.Context, tenantID, purchaseOrderID string) (PurchaseOrder, error) {
	if err := requireTenant(tenantID); err != nil {
		return PurchaseOrder{}, err
	}
	if purchaseOrderID == "" {
		return PurchaseOrder{}, fmt.Errorf("%w: purchase_order_id is required", ErrInvalid)
	}
	return s.store.ReceivePurchaseOrder(ctx, tenantID, purchaseOrderID)
}

func (s *Service) CancelPurchaseOrder(ctx context.Context, tenantID, purchaseOrderID string) (PurchaseOrder, error) {
	if err := requireTenant(tenantID); err != nil {
		return PurchaseOrder{}, err
	}
	if purchaseOrderID == "" {
		return PurchaseOrder{}, fmt.Errorf("%w: purchase_order_id is required", ErrInvalid)
	}
	return s.store.CancelPurchaseOrder(ctx, tenantID, purchaseOrderID)
}

func (s *Service) ListStock(ctx context.Context, tenantID string) ([]StockLevel, error) {
	if err := requireTenant(tenantID); err != nil {
		return nil, err
	}
	return s.store.ListStock(ctx, tenantID)
}

func (s *Service) ReceiveStock(ctx context.Context, tenantID, productID, warehouseID string, quantity, unitCostCents int64, reason string) error {
	if err := requireTenant(tenantID); err != nil {
		return err
	}
	if productID == "" || warehouseID == "" {
		return fmt.Errorf("%w: product_id and warehouse_id are required", ErrInvalid)
	}
	if quantity == 0 {
		return fmt.Errorf("%w: quantity must not be zero", ErrInvalid)
	}
	if unitCostCents < 0 {
		return fmt.Errorf("%w: unit_cost_cents must not be negative", ErrInvalid)
	}
	if quantity < 0 && unitCostCents != 0 {
		return fmt.Errorf("%w: unit_cost_cents is only accepted for receipts", ErrInvalid)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "manual stock movement"
	}
	return s.store.ReceiveStock(ctx, tenantID, productID, warehouseID, quantity, unitCostCents, reason)
}

func (s *Service) CreateOrder(ctx context.Context, tenantID, retailerID string, lines []CreateOrderLine) (Order, error) {
	if err := requireTenant(tenantID); err != nil {
		return Order{}, err
	}
	if retailerID == "" {
		return Order{}, fmt.Errorf("%w: retailer_id is required", ErrInvalid)
	}
	if len(lines) == 0 {
		return Order{}, fmt.Errorf("%w: at least one order line is required", ErrInvalid)
	}

	seen := make(map[string]struct{}, len(lines))
	clean := make([]CreateOrderLine, 0, len(lines))
	for _, line := range lines {
		if line.ProductID == "" {
			return Order{}, fmt.Errorf("%w: product_id is required", ErrInvalid)
		}
		if line.Quantity <= 0 {
			return Order{}, fmt.Errorf("%w: quantity must be positive", ErrInvalid)
		}
		if _, ok := seen[line.ProductID]; ok {
			return Order{}, fmt.Errorf("%w: duplicate product_id %s", ErrInvalid, line.ProductID)
		}
		seen[line.ProductID] = struct{}{}
		clean = append(clean, line)
	}

	return s.store.CreateOrder(ctx, tenantID, retailerID, clean)
}

func (s *Service) ListOrders(ctx context.Context, tenantID string) ([]Order, error) {
	if err := requireTenant(tenantID); err != nil {
		return nil, err
	}
	return s.store.ListOrders(ctx, tenantID)
}

func (s *Service) GetOrder(ctx context.Context, tenantID, orderID string) (Order, error) {
	if err := requireTenant(tenantID); err != nil {
		return Order{}, err
	}
	if orderID == "" {
		return Order{}, fmt.Errorf("%w: order_id is required", ErrInvalid)
	}
	return s.store.GetOrder(ctx, tenantID, orderID)
}

func (s *Service) ReserveOrder(ctx context.Context, tenantID, orderID string) (Order, error) {
	if err := requireTenant(tenantID); err != nil {
		return Order{}, err
	}
	if orderID == "" {
		return Order{}, fmt.Errorf("%w: order_id is required", ErrInvalid)
	}
	return s.store.ReserveOrder(ctx, tenantID, orderID)
}

func (s *Service) CancelOrder(ctx context.Context, tenantID, orderID string) (Order, error) {
	if err := requireTenant(tenantID); err != nil {
		return Order{}, err
	}
	if orderID == "" {
		return Order{}, fmt.Errorf("%w: order_id is required", ErrInvalid)
	}
	return s.store.CancelOrder(ctx, tenantID, orderID)
}

func (s *Service) InvoiceOrder(ctx context.Context, tenantID, orderID string) (Invoice, error) {
	if err := requireTenant(tenantID); err != nil {
		return Invoice{}, err
	}
	if orderID == "" {
		return Invoice{}, fmt.Errorf("%w: order_id is required", ErrInvalid)
	}
	return s.store.InvoiceOrder(ctx, tenantID, orderID)
}

func (s *Service) ListInvoices(ctx context.Context, tenantID string) ([]Invoice, error) {
	if err := requireTenant(tenantID); err != nil {
		return nil, err
	}
	return s.store.ListInvoices(ctx, tenantID)
}

func (s *Service) GetInvoice(ctx context.Context, tenantID, invoiceID string) (InvoiceDetail, error) {
	if err := requireTenant(tenantID); err != nil {
		return InvoiceDetail{}, err
	}
	if invoiceID == "" {
		return InvoiceDetail{}, fmt.Errorf("%w: invoice_id is required", ErrInvalid)
	}
	return s.store.GetInvoice(ctx, tenantID, invoiceID)
}

func (s *Service) GetCredit(ctx context.Context, tenantID, retailerID string) (CreditView, error) {
	if err := requireTenant(tenantID); err != nil {
		return CreditView{}, err
	}
	if retailerID == "" {
		return CreditView{}, fmt.Errorf("%w: retailer_id is required", ErrInvalid)
	}
	return s.store.GetCredit(ctx, tenantID, retailerID)
}

func (s *Service) RecordPayment(ctx context.Context, tenantID, invoiceID string, amountCents int64) (Payment, error) {
	if err := requireTenant(tenantID); err != nil {
		return Payment{}, err
	}
	if invoiceID == "" {
		return Payment{}, fmt.Errorf("%w: invoice_id is required", ErrInvalid)
	}
	if amountCents <= 0 {
		return Payment{}, fmt.Errorf("%w: amount_cents must be positive", ErrInvalid)
	}
	return s.store.RecordPayment(ctx, tenantID, invoiceID, amountCents)
}

func (s *Service) ReturnInvoice(ctx context.Context, tenantID, invoiceID string, lines []CreateReturnLine) (SalesReturn, error) {
	if err := requireTenant(tenantID); err != nil {
		return SalesReturn{}, err
	}
	if invoiceID == "" {
		return SalesReturn{}, fmt.Errorf("%w: invoice_id is required", ErrInvalid)
	}
	if len(lines) == 0 {
		return SalesReturn{}, fmt.Errorf("%w: at least one return line is required", ErrInvalid)
	}

	seen := make(map[string]struct{}, len(lines))
	clean := make([]CreateReturnLine, 0, len(lines))
	for _, line := range lines {
		if line.ProductID == "" {
			return SalesReturn{}, fmt.Errorf("%w: product_id is required", ErrInvalid)
		}
		if line.Quantity <= 0 {
			return SalesReturn{}, fmt.Errorf("%w: quantity must be positive", ErrInvalid)
		}
		if _, ok := seen[line.ProductID]; ok {
			return SalesReturn{}, fmt.Errorf("%w: duplicate product_id %s", ErrInvalid, line.ProductID)
		}
		seen[line.ProductID] = struct{}{}
		clean = append(clean, line)
	}
	return s.store.ReturnInvoice(ctx, tenantID, invoiceID, clean)
}

func (s *Service) Valuation(ctx context.Context, tenantID string) (InventoryValuation, error) {
	if err := requireTenant(tenantID); err != nil {
		return InventoryValuation{}, err
	}
	return s.store.Valuation(ctx, tenantID)
}

func (s *Service) TrialBalance(ctx context.Context, tenantID string) ([]AccountBalance, error) {
	if err := requireTenant(tenantID); err != nil {
		return nil, err
	}
	return s.store.TrialBalance(ctx, tenantID)
}

func cleanProductInput(input ProductInput) (ProductInput, error) {
	input.SKU = strings.TrimSpace(input.SKU)
	input.Name = strings.TrimSpace(input.Name)
	input.Unit = strings.TrimSpace(input.Unit)
	if input.Unit == "" {
		input.Unit = "unit"
	}
	switch {
	case input.SKU == "":
		return ProductInput{}, fmt.Errorf("%w: sku is required", ErrInvalid)
	case len(input.SKU) > 64:
		return ProductInput{}, fmt.Errorf("%w: sku must be 64 characters or fewer", ErrInvalid)
	case input.Name == "":
		return ProductInput{}, fmt.Errorf("%w: product name is required", ErrInvalid)
	case len(input.Name) > 160:
		return ProductInput{}, fmt.Errorf("%w: product name must be 160 characters or fewer", ErrInvalid)
	case len(input.Unit) > 32:
		return ProductInput{}, fmt.Errorf("%w: unit must be 32 characters or fewer", ErrInvalid)
	case input.PriceCents < 0:
		return ProductInput{}, fmt.Errorf("%w: price_cents must not be negative", ErrInvalid)
	}
	return input, nil
}

func cleanRetailerInput(input RetailerInput) (RetailerInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	switch {
	case input.Name == "":
		return RetailerInput{}, fmt.Errorf("%w: retailer name is required", ErrInvalid)
	case len(input.Name) > 160:
		return RetailerInput{}, fmt.Errorf("%w: retailer name must be 160 characters or fewer", ErrInvalid)
	case input.CreditLimitCents < 0:
		return RetailerInput{}, fmt.Errorf("%w: credit_limit_cents must not be negative", ErrInvalid)
	}
	return input, nil
}

func cleanWarehouseInput(input WarehouseInput) (WarehouseInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	switch {
	case input.Name == "":
		return WarehouseInput{}, fmt.Errorf("%w: warehouse name is required", ErrInvalid)
	case len(input.Name) > 160:
		return WarehouseInput{}, fmt.Errorf("%w: warehouse name must be 160 characters or fewer", ErrInvalid)
	}
	return input, nil
}

func cleanSupplierInput(input SupplierInput) (SupplierInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.ContactName = strings.TrimSpace(input.ContactName)
	input.Phone = strings.TrimSpace(input.Phone)
	switch {
	case input.Name == "":
		return SupplierInput{}, fmt.Errorf("%w: supplier name is required", ErrInvalid)
	case len(input.Name) > 160:
		return SupplierInput{}, fmt.Errorf("%w: supplier name must be 160 characters or fewer", ErrInvalid)
	case len(input.ContactName) > 160:
		return SupplierInput{}, fmt.Errorf("%w: contact name must be 160 characters or fewer", ErrInvalid)
	case len(input.Phone) > 40:
		return SupplierInput{}, fmt.Errorf("%w: phone must be 40 characters or fewer", ErrInvalid)
	}
	return input, nil
}

func requireTenant(tenantID string) error {
	if strings.TrimSpace(tenantID) == "" {
		return fmt.Errorf("%w: tenant is required", ErrInvalid)
	}
	return nil
}
