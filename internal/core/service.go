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

func (s *Service) ListWarehouses(ctx context.Context, tenantID string) ([]Warehouse, error) {
	if err := requireTenant(tenantID); err != nil {
		return nil, err
	}
	return s.store.ListWarehouses(ctx, tenantID)
}

func (s *Service) ListRetailers(ctx context.Context, tenantID string) ([]Retailer, error) {
	if err := requireTenant(tenantID); err != nil {
		return nil, err
	}
	return s.store.ListRetailers(ctx, tenantID)
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

func requireTenant(tenantID string) error {
	if strings.TrimSpace(tenantID) == "" {
		return fmt.Errorf("%w: tenant is required", ErrInvalid)
	}
	return nil
}
