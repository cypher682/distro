package core

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrInvalid  = errors.New("invalid input")
)

type Product struct {
	ID         string `json:"id"`
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	Unit       string `json:"unit"`
	PriceCents int64  `json:"price_cents"`
}

type Retailer struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	CreditLimitCents int64  `json:"credit_limit_cents"`
}

type StockLevel struct {
	ProductID           string `json:"product_id"`
	WarehouseID         string `json:"warehouse_id"`
	OnHand              int64  `json:"on_hand"`
	Reserved            int64  `json:"reserved"`
	Available           int64  `json:"available"`
	InventoryValueCents int64  `json:"inventory_value_cents"`
	AverageCostCents    int64  `json:"average_cost_cents"`
}

type InventoryValuation struct {
	Items                []StockLevel `json:"items"`
	TotalValueCents      int64        `json:"total_value_cents"`
	LedgerInventoryCents int64        `json:"ledger_inventory_cents"`
	CogsCents            int64        `json:"cogs_cents"`
	Balanced             bool         `json:"balanced"`
}

type OrderLine struct {
	ID             string `json:"id"`
	ProductID      string `json:"product_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	LineTotalCents int64  `json:"line_total_cents"`
}

type Order struct {
	ID         string      `json:"id"`
	RetailerID string      `json:"retailer_id"`
	Status     string      `json:"status"`
	TotalCents int64       `json:"total_cents"`
	Lines      []OrderLine `json:"lines"`
	CreatedAt  time.Time   `json:"created_at"`
}

type Invoice struct {
	ID            string        `json:"id"`
	OrderID       string        `json:"order_id"`
	RetailerID    string        `json:"retailer_id"`
	Status        string        `json:"status"`
	TotalCents    int64         `json:"total_cents"`
	PaidCents     int64         `json:"paid_cents"`
	ReturnedCents int64         `json:"returned_cents"`
	Lines         []InvoiceLine `json:"lines"`
	CreatedAt     time.Time     `json:"created_at"`
}

type InvoiceLine struct {
	ID             string `json:"id"`
	InvoiceID      string `json:"invoice_id"`
	ProductID      string `json:"product_id"`
	WarehouseID    string `json:"warehouse_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	LineTotalCents int64  `json:"line_total_cents"`
	TotalCostCents int64  `json:"total_cost_cents"`
}

type SalesReturn struct {
	ID               string            `json:"id"`
	InvoiceID        string            `json:"invoice_id"`
	RetailerID       string            `json:"retailer_id"`
	TotalRefundCents int64             `json:"total_refund_cents"`
	TotalCostCents   int64             `json:"total_cost_cents"`
	Lines            []SalesReturnLine `json:"lines"`
	CreatedAt        time.Time         `json:"created_at"`
}

type SalesReturnLine struct {
	ID            string `json:"id"`
	SalesReturnID string `json:"sales_return_id"`
	InvoiceLineID string `json:"invoice_line_id"`
	ProductID     string `json:"product_id"`
	Quantity      int    `json:"quantity"`
	RefundCents   int64  `json:"refund_cents"`
	CostCents     int64  `json:"cost_cents"`
	Restock       bool   `json:"restock"`
}

type CreateReturnLine struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Restock   bool   `json:"restock"`
}

type Payment struct {
	ID          string    `json:"id"`
	InvoiceID   string    `json:"invoice_id"`
	AmountCents int64     `json:"amount_cents"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreditView struct {
	RetailerID         string `json:"retailer_id"`
	CreditLimitCents   int64  `json:"credit_limit_cents"`
	OpenInvoiceCents   int64  `json:"open_invoice_cents"`
	ReservedOrderCents int64  `json:"reserved_order_cents"`
	ExposureCents      int64  `json:"exposure_cents"`
	AvailableCents     int64  `json:"available_cents"`
}

type AccountBalance struct {
	Account      string `json:"account"`
	DebitCents   int64  `json:"debit_cents"`
	CreditCents  int64  `json:"credit_cents"`
	BalanceCents int64  `json:"balance_cents"`
}

type CreateOrderLine struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}
