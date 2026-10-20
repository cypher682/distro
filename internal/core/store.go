package core

import "context"

type Store interface {
	ListProducts(ctx context.Context, tenantID string) ([]Product, error)
	CreateProduct(ctx context.Context, tenantID string, input ProductInput) (Product, error)
	UpdateProduct(ctx context.Context, tenantID, productID string, input ProductInput) (Product, error)
	ListWarehouses(ctx context.Context, tenantID string) ([]Warehouse, error)
	CreateWarehouse(ctx context.Context, tenantID string, input WarehouseInput) (Warehouse, error)
	UpdateWarehouse(ctx context.Context, tenantID, warehouseID string, input WarehouseInput) (Warehouse, error)
	ListRetailers(ctx context.Context, tenantID string) ([]Retailer, error)
	CreateRetailer(ctx context.Context, tenantID string, input RetailerInput) (Retailer, error)
	UpdateRetailer(ctx context.Context, tenantID, retailerID string, input RetailerInput) (Retailer, error)
	ListSuppliers(ctx context.Context, tenantID string) ([]Supplier, error)
	CreateSupplier(ctx context.Context, tenantID string, input SupplierInput) (Supplier, error)
	UpdateSupplier(ctx context.Context, tenantID, supplierID string, input SupplierInput) (Supplier, error)
	CreatePurchaseOrder(ctx context.Context, tenantID string, input CreatePurchaseOrderInput) (PurchaseOrder, error)
	ListPurchaseOrders(ctx context.Context, tenantID string) ([]PurchaseOrder, error)
	GetPurchaseOrder(ctx context.Context, tenantID, purchaseOrderID string) (PurchaseOrder, error)
	ReceivePurchaseOrder(ctx context.Context, tenantID, purchaseOrderID string) (PurchaseOrder, error)
	CancelPurchaseOrder(ctx context.Context, tenantID, purchaseOrderID string) (PurchaseOrder, error)
	ListStock(ctx context.Context, tenantID string) ([]StockLevel, error)
	ReceiveStock(ctx context.Context, tenantID, productID, warehouseID string, quantity, unitCostCents int64, reason string) error
	CreateOrder(ctx context.Context, tenantID, retailerID string, lines []CreateOrderLine) (Order, error)
	ListOrders(ctx context.Context, tenantID string) ([]Order, error)
	GetOrder(ctx context.Context, tenantID, orderID string) (Order, error)
	ReserveOrder(ctx context.Context, tenantID, orderID string) (Order, error)
	CancelOrder(ctx context.Context, tenantID, orderID string) (Order, error)
	InvoiceOrder(ctx context.Context, tenantID, orderID string) (Invoice, error)
	ListInvoices(ctx context.Context, tenantID string) ([]Invoice, error)
	GetInvoice(ctx context.Context, tenantID, invoiceID string) (InvoiceDetail, error)
	GetCredit(ctx context.Context, tenantID, retailerID string) (CreditView, error)
	RecordPayment(ctx context.Context, tenantID, invoiceID string, amountCents int64) (Payment, error)
	ReturnInvoice(ctx context.Context, tenantID, invoiceID string, lines []CreateReturnLine) (SalesReturn, error)
	Valuation(ctx context.Context, tenantID string) (InventoryValuation, error)
	TrialBalance(ctx context.Context, tenantID string) ([]AccountBalance, error)
}
