package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

const (
	testTenant    = "00000000-0000-0000-0000-000000000001"
	testWarehouse = "00000000-0000-0000-0000-000000000010"
	testRetailer  = "00000000-0000-0000-0000-000000000020"
	testProduct   = "00000000-0000-0000-0000-000000000101"
)

func testService(t *testing.T, stock int64, creditLimit int64) (*MemoryStore, *Service) {
	t.Helper()

	store := NewMemoryStore()
	store.AddWarehouse(testTenant, testWarehouse, "Main Warehouse")
	store.AddRetailer(testTenant, Retailer{
		ID:               testRetailer,
		Name:             "Demo Retailer",
		CreditLimitCents: creditLimit,
	})
	store.AddProduct(testTenant, Product{
		ID:         testProduct,
		SKU:        "RICE-25",
		Name:       "Rice 25kg",
		Unit:       "bag",
		PriceCents: 1000,
	})
	store.SetStock(testTenant, testProduct, testWarehouse, stock)
	return store, NewService(store)
}

func createOrder(t *testing.T, svc *Service, quantity int) Order {
	t.Helper()

	order, err := svc.CreateOrder(context.Background(), testTenant, testRetailer, []CreateOrderLine{
		{ProductID: testProduct, Quantity: quantity},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	return order
}

func TestReserveRejectsOversell(t *testing.T) {
	store, svc := testService(t, 5, 100000)
	order := createOrder(t, svc, 6)

	_, err := svc.ReserveOrder(context.Background(), testTenant, order.ID)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}

	level := stockLevel(t, store)
	if level.Available != 5 || level.Reserved != 0 {
		t.Fatalf("expected stock untouched, got %+v", level)
	}
}

func TestConcurrentReservationsDoNotOversell(t *testing.T) {
	store, svc := testService(t, 10, 100000)

	const attempts = 40
	var successes atomic.Int64
	var failures atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			order, err := svc.CreateOrder(context.Background(), testTenant, testRetailer, []CreateOrderLine{
				{ProductID: testProduct, Quantity: 1},
			})
			if err != nil {
				t.Errorf("create order: %v", err)
				return
			}
			if _, err := svc.ReserveOrder(context.Background(), testTenant, order.ID); err != nil {
				if !errors.Is(err, ErrConflict) {
					t.Errorf("reserve order: %v", err)
					return
				}
				failures.Add(1)
				return
			}
			successes.Add(1)
		}()
	}
	wg.Wait()

	if successes.Load() != 10 {
		t.Fatalf("expected 10 successful reservations, got %d", successes.Load())
	}
	if failures.Load() != attempts-10 {
		t.Fatalf("expected %d conflicts, got %d", attempts-10, failures.Load())
	}

	level := stockLevel(t, store)
	if level.OnHand != 10 || level.Reserved != 10 || level.Available != 0 {
		t.Fatalf("unexpected final stock: %+v", level)
	}
}

func TestInvoiceAndPaymentKeepLedgerBalanced(t *testing.T) {
	store, svc := testService(t, 10, 100000)
	order := createOrder(t, svc, 2)

	if _, err := svc.ReserveOrder(context.Background(), testTenant, order.ID); err != nil {
		t.Fatalf("reserve order: %v", err)
	}
	invoice, err := svc.InvoiceOrder(context.Background(), testTenant, order.ID)
	if err != nil {
		t.Fatalf("invoice order: %v", err)
	}
	if invoice.TotalCents != 2000 {
		t.Fatalf("expected invoice total 2000, got %d", invoice.TotalCents)
	}
	if level := stockLevel(t, store); level.OnHand != 8 || level.Reserved != 0 || level.Available != 8 {
		t.Fatalf("expected fulfilled stock to leave inventory, got %+v", level)
	}

	credit, err := svc.GetCredit(context.Background(), testTenant, testRetailer)
	if err != nil {
		t.Fatalf("get credit: %v", err)
	}
	if credit.OpenInvoiceCents != 2000 || credit.ExposureCents != 2000 {
		t.Fatalf("unexpected credit after invoice: %+v", credit)
	}
	assertLedgerBalanced(t, svc)

	if _, err := svc.RecordPayment(context.Background(), testTenant, invoice.ID, 500); err != nil {
		t.Fatalf("record partial payment: %v", err)
	}
	credit, err = svc.GetCredit(context.Background(), testTenant, testRetailer)
	if err != nil {
		t.Fatalf("get credit: %v", err)
	}
	if credit.OpenInvoiceCents != 1500 || credit.ExposureCents != 1500 {
		t.Fatalf("unexpected credit after partial payment: %+v", credit)
	}
	assertLedgerBalanced(t, svc)

	if _, err := svc.RecordPayment(context.Background(), testTenant, invoice.ID, 1500); err != nil {
		t.Fatalf("record final payment: %v", err)
	}
	credit, err = svc.GetCredit(context.Background(), testTenant, testRetailer)
	if err != nil {
		t.Fatalf("get credit: %v", err)
	}
	if credit.OpenInvoiceCents != 0 || credit.ExposureCents != 0 {
		t.Fatalf("unexpected credit after full payment: %+v", credit)
	}
	assertLedgerBalanced(t, svc)
}

func TestInvoicedStockIsNotAvailableAgain(t *testing.T) {
	store, svc := testService(t, 5, 100000)
	order := createOrder(t, svc, 5)
	if _, err := svc.ReserveOrder(context.Background(), testTenant, order.ID); err != nil {
		t.Fatalf("reserve order: %v", err)
	}
	if _, err := svc.InvoiceOrder(context.Background(), testTenant, order.ID); err != nil {
		t.Fatalf("invoice order: %v", err)
	}

	next := createOrder(t, svc, 1)
	if _, err := svc.ReserveOrder(context.Background(), testTenant, next.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected no stock conflict, got %v", err)
	}
	if level := stockLevel(t, store); level.OnHand != 0 || level.Available != 0 {
		t.Fatalf("expected stock to be consumed, got %+v", level)
	}
}

func TestPaymentOverpayRejected(t *testing.T) {
	_, svc := testService(t, 10, 100000)
	order := createOrder(t, svc, 1)
	if _, err := svc.ReserveOrder(context.Background(), testTenant, order.ID); err != nil {
		t.Fatalf("reserve order: %v", err)
	}
	invoice, err := svc.InvoiceOrder(context.Background(), testTenant, order.ID)
	if err != nil {
		t.Fatalf("invoice order: %v", err)
	}

	if _, err := svc.RecordPayment(context.Background(), testTenant, invoice.ID, invoice.TotalCents+1); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected overpay conflict, got %v", err)
	}
}

func TestCancelReleasesReservation(t *testing.T) {
	store, svc := testService(t, 5, 100000)
	order := createOrder(t, svc, 5)
	if _, err := svc.ReserveOrder(context.Background(), testTenant, order.ID); err != nil {
		t.Fatalf("reserve order: %v", err)
	}
	if level := stockLevel(t, store); level.Available != 0 {
		t.Fatalf("expected no available stock, got %+v", level)
	}

	cancelled, err := svc.CancelOrder(context.Background(), testTenant, order.ID)
	if err != nil {
		t.Fatalf("cancel order: %v", err)
	}
	if cancelled.Status != "cancelled" {
		t.Fatalf("expected cancelled order, got %s", cancelled.Status)
	}
	if level := stockLevel(t, store); level.Available != 5 || level.Reserved != 0 {
		t.Fatalf("expected released stock, got %+v", level)
	}
}

func TestCreditLimitBlocksReservation(t *testing.T) {
	_, svc := testService(t, 10, 1000)
	order := createOrder(t, svc, 2)

	if _, err := svc.ReserveOrder(context.Background(), testTenant, order.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected credit conflict, got %v", err)
	}
}

func TestStockAdjustmentCannotConsumeReservedStock(t *testing.T) {
	store, svc := testService(t, 5, 100000)
	order := createOrder(t, svc, 4)
	if _, err := svc.ReserveOrder(context.Background(), testTenant, order.ID); err != nil {
		t.Fatalf("reserve order: %v", err)
	}

	if err := svc.ReceiveStock(context.Background(), testTenant, testProduct, testWarehouse, -2, 0, "shrinkage"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected adjustment conflict, got %v", err)
	}
	if err := svc.ReceiveStock(context.Background(), testTenant, testProduct, testWarehouse, -1, 0, "shrinkage"); err != nil {
		t.Fatalf("adjust available stock: %v", err)
	}
	if level := stockLevel(t, store); level.OnHand != 4 || level.Reserved != 4 || level.Available != 0 {
		t.Fatalf("unexpected stock after adjustment: %+v", level)
	}
}

func TestDuplicateOrderLinesRejected(t *testing.T) {
	_, svc := testService(t, 10, 100000)

	_, err := svc.CreateOrder(context.Background(), testTenant, testRetailer, []CreateOrderLine{
		{ProductID: testProduct, Quantity: 1},
		{ProductID: testProduct, Quantity: 2},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestInvoicePostsWeightedAverageCogs(t *testing.T) {
	store, svc := testService(t, 10, 100000)
	store.SetStockValue(testTenant, testProduct, testWarehouse, 10, 1000)
	order := createOrder(t, svc, 4)

	if _, err := svc.ReserveOrder(context.Background(), testTenant, order.ID); err != nil {
		t.Fatalf("reserve order: %v", err)
	}
	if _, err := svc.InvoiceOrder(context.Background(), testTenant, order.ID); err != nil {
		t.Fatalf("invoice order: %v", err)
	}

	valuation, err := svc.Valuation(context.Background(), testTenant)
	if err != nil {
		t.Fatalf("valuation: %v", err)
	}
	if valuation.TotalValueCents != 600 || valuation.LedgerInventoryCents != 600 {
		t.Fatalf("unexpected inventory valuation: %+v", valuation)
	}
	if valuation.CogsCents != 400 {
		t.Fatalf("expected cogs 400, got %d", valuation.CogsCents)
	}
	if !valuation.Balanced {
		t.Fatalf("expected valuation to reconcile: %+v", valuation)
	}
	if len(valuation.Items) != 1 || valuation.Items[0].AverageCostCents != 100 {
		t.Fatalf("expected average cost 100, got %+v", valuation.Items)
	}
	assertLedgerBalanced(t, svc)
}

func TestWeightedAverageRoundingReconcilesToZero(t *testing.T) {
	store, svc := testService(t, 3, 100000)
	store.SetStockValue(testTenant, testProduct, testWarehouse, 3, 100)

	for i := 0; i < 3; i++ {
		order := createOrder(t, svc, 1)
		if _, err := svc.ReserveOrder(context.Background(), testTenant, order.ID); err != nil {
			t.Fatalf("reserve order %d: %v", i, err)
		}
		if _, err := svc.InvoiceOrder(context.Background(), testTenant, order.ID); err != nil {
			t.Fatalf("invoice order %d: %v", i, err)
		}
	}

	valuation, err := svc.Valuation(context.Background(), testTenant)
	if err != nil {
		t.Fatalf("valuation: %v", err)
	}
	if valuation.TotalValueCents != 0 || valuation.LedgerInventoryCents != 0 {
		t.Fatalf("expected inventory to be fully released: %+v", valuation)
	}
	if valuation.CogsCents != 100 {
		t.Fatalf("expected cogs 100, got %d", valuation.CogsCents)
	}
	if !valuation.Balanced {
		t.Fatalf("expected valuation to reconcile: %+v", valuation)
	}
	assertLedgerBalanced(t, svc)
}

func TestStockAdjustmentUsesAverageCost(t *testing.T) {
	store, svc := testService(t, 10, 100000)
	store.SetStockValue(testTenant, testProduct, testWarehouse, 10, 1000)

	if err := svc.ReceiveStock(context.Background(), testTenant, testProduct, testWarehouse, -2, 0, "shrinkage"); err != nil {
		t.Fatalf("adjust stock: %v", err)
	}

	valuation, err := svc.Valuation(context.Background(), testTenant)
	if err != nil {
		t.Fatalf("valuation: %v", err)
	}
	if valuation.TotalValueCents != 800 || valuation.LedgerInventoryCents != 800 {
		t.Fatalf("unexpected inventory valuation: %+v", valuation)
	}
	if !valuation.Balanced {
		t.Fatalf("expected valuation to reconcile: %+v", valuation)
	}
	assertLedgerBalanced(t, svc)
}

func TestReceiveStockRequiresReceiptCost(t *testing.T) {
	_, svc := testService(t, 0, 100000)

	err := svc.ReceiveStock(context.Background(), testTenant, testProduct, testWarehouse, 5, -1, "receipt")
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid cost, got %v", err)
	}
}

func stockLevel(t *testing.T, store *MemoryStore) StockLevel {
	t.Helper()

	levels, err := store.ListStock(context.Background(), testTenant)
	if err != nil {
		t.Fatalf("list stock: %v", err)
	}
	if len(levels) != 1 {
		t.Fatalf("expected one stock level, got %d", len(levels))
	}
	return levels[0]
}

func assertLedgerBalanced(t *testing.T, svc *Service) {
	t.Helper()

	accounts, err := svc.TrialBalance(context.Background(), testTenant)
	if err != nil {
		t.Fatalf("trial balance: %v", err)
	}
	var debits, credits int64
	for _, account := range accounts {
		debits += account.DebitCents
		credits += account.CreditCents
	}
	if debits != credits {
		t.Fatalf("ledger is not balanced: debits=%d credits=%d accounts=%+v", debits, credits, accounts)
	}
}
