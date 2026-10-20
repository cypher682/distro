package core

import "context"

type Store interface {
	ListProducts(ctx context.Context, tenantID string) ([]Product, error)
	ListRetailers(ctx context.Context, tenantID string) ([]Retailer, error)
	ListStock(ctx context.Context, tenantID string) ([]StockLevel, error)
	ReceiveStock(ctx context.Context, tenantID, productID, warehouseID string, quantity, unitCostCents int64, reason string) error
	CreateOrder(ctx context.Context, tenantID, retailerID string, lines []CreateOrderLine) (Order, error)
	GetOrder(ctx context.Context, tenantID, orderID string) (Order, error)
	ReserveOrder(ctx context.Context, tenantID, orderID string) (Order, error)
	CancelOrder(ctx context.Context, tenantID, orderID string) (Order, error)
	InvoiceOrder(ctx context.Context, tenantID, orderID string) (Invoice, error)
	GetCredit(ctx context.Context, tenantID, retailerID string) (CreditView, error)
	RecordPayment(ctx context.Context, tenantID, invoiceID string, amountCents int64) (Payment, error)
	Valuation(ctx context.Context, tenantID string) (InventoryValuation, error)
	TrialBalance(ctx context.Context, tenantID string) ([]AccountBalance, error)
}
