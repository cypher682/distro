package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cypher682/distro/internal/core"
)

const (
	apiTenant    = "00000000-0000-0000-0000-000000000001"
	apiWarehouse = "00000000-0000-0000-0000-000000000010"
	apiRetailer  = "00000000-0000-0000-0000-000000000020"
	apiProduct   = "00000000-0000-0000-0000-000000000101"
)

func apiRouter() http.Handler {
	store := core.NewMemoryStore()
	store.AddWarehouse(apiTenant, apiWarehouse, "Main Warehouse")
	store.AddRetailer(apiTenant, core.Retailer{
		ID:               apiRetailer,
		Name:             "Demo Retailer",
		CreditLimitCents: 10000,
	})
	store.AddProduct(apiTenant, core.Product{
		ID:         apiProduct,
		SKU:        "RICE-25",
		Name:       "Rice 25kg",
		Unit:       "bag",
		PriceCents: 1000,
	})
	store.SetStock(apiTenant, apiProduct, apiWarehouse, 10)
	return NewRouter(fakePinger{}, core.NewService(store))
}

func TestOrderToPaymentFlow(t *testing.T) {
	router := apiRouter()

	create := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{
		"retailer_id": "`+apiRetailer+`",
		"lines": [{"product_id": "`+apiProduct+`", "quantity": 2}]
	}`))
	create.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, create)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create order: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var order core.Order
	if err := json.NewDecoder(createRec.Body).Decode(&order); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	if order.TotalCents != 2000 {
		t.Fatalf("expected total 2000, got %d", order.TotalCents)
	}

	reserve := httptest.NewRequest(http.MethodPost, "/api/orders/"+order.ID+"/reserve", nil)
	reserveRec := httptest.NewRecorder()
	router.ServeHTTP(reserveRec, reserve)
	if reserveRec.Code != http.StatusOK {
		t.Fatalf("reserve: expected 200, got %d: %s", reserveRec.Code, reserveRec.Body.String())
	}

	invoiceRequest := httptest.NewRequest(http.MethodPost, "/api/orders/"+order.ID+"/invoice", nil)
	invoiceRec := httptest.NewRecorder()
	router.ServeHTTP(invoiceRec, invoiceRequest)
	if invoiceRec.Code != http.StatusCreated {
		t.Fatalf("invoice: expected 201, got %d: %s", invoiceRec.Code, invoiceRec.Body.String())
	}

	var invoice core.Invoice
	if err := json.NewDecoder(invoiceRec.Body).Decode(&invoice); err != nil {
		t.Fatalf("decode invoice: %v", err)
	}

	paymentRequest := httptest.NewRequest(http.MethodPost, "/api/invoices/"+invoice.ID+"/payments",
		strings.NewReader(`{"amount_cents": 2000}`))
	paymentRequest.Header.Set("Content-Type", "application/json")
	paymentRec := httptest.NewRecorder()
	router.ServeHTTP(paymentRec, paymentRequest)
	if paymentRec.Code != http.StatusCreated {
		t.Fatalf("payment: expected 201, got %d: %s", paymentRec.Code, paymentRec.Body.String())
	}

	creditRequest := httptest.NewRequest(http.MethodGet, "/api/retailers/"+apiRetailer+"/credit", nil)
	creditRec := httptest.NewRecorder()
	router.ServeHTTP(creditRec, creditRequest)
	if creditRec.Code != http.StatusOK {
		t.Fatalf("credit: expected 200, got %d: %s", creditRec.Code, creditRec.Body.String())
	}

	var credit core.CreditView
	if err := json.NewDecoder(creditRec.Body).Decode(&credit); err != nil {
		t.Fatalf("decode credit: %v", err)
	}
	if credit.ExposureCents != 0 {
		t.Fatalf("expected zero exposure after payment, got %+v", credit)
	}

	balanceRequest := httptest.NewRequest(http.MethodGet, "/api/ledger/trial-balance", nil)
	balanceRec := httptest.NewRecorder()
	router.ServeHTTP(balanceRec, balanceRequest)
	if balanceRec.Code != http.StatusOK {
		t.Fatalf("trial balance: expected 200, got %d: %s", balanceRec.Code, balanceRec.Body.String())
	}

	var balance trialBalanceResponse
	if err := json.NewDecoder(balanceRec.Body).Decode(&balance); err != nil {
		t.Fatalf("decode trial balance: %v", err)
	}
	if !balance.Balanced {
		t.Fatalf("expected balanced ledger, got %+v", balance)
	}
}

func TestInventoryValuationEndpointReconciles(t *testing.T) {
	router := apiRouter()

	receive := httptest.NewRequest(http.MethodPost, "/api/stock/receive", strings.NewReader(`{
		"product_id": "`+apiProduct+`",
		"warehouse_id": "`+apiWarehouse+`",
		"quantity": 10,
		"unit_cost_cents": 100
	}`))
	receive.Header.Set("Content-Type", "application/json")
	receiveRec := httptest.NewRecorder()
	router.ServeHTTP(receiveRec, receive)
	if receiveRec.Code != http.StatusOK {
		t.Fatalf("receive stock: expected 200, got %d: %s", receiveRec.Code, receiveRec.Body.String())
	}

	create := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{
		"retailer_id": "`+apiRetailer+`",
		"lines": [{"product_id": "`+apiProduct+`", "quantity": 2}]
	}`))
	create.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, create)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create order: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var order core.Order
	if err := json.NewDecoder(createRec.Body).Decode(&order); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	reserve := httptest.NewRequest(http.MethodPost, "/api/orders/"+order.ID+"/reserve", nil)
	reserveRec := httptest.NewRecorder()
	router.ServeHTTP(reserveRec, reserve)
	if reserveRec.Code != http.StatusOK {
		t.Fatalf("reserve: expected 200, got %d: %s", reserveRec.Code, reserveRec.Body.String())
	}
	invoice := httptest.NewRequest(http.MethodPost, "/api/orders/"+order.ID+"/invoice", nil)
	invoiceRec := httptest.NewRecorder()
	router.ServeHTTP(invoiceRec, invoice)
	if invoiceRec.Code != http.StatusCreated {
		t.Fatalf("invoice: expected 201, got %d: %s", invoiceRec.Code, invoiceRec.Body.String())
	}

	valuationRequest := httptest.NewRequest(http.MethodGet, "/api/inventory/valuation", nil)
	valuationRec := httptest.NewRecorder()
	router.ServeHTTP(valuationRec, valuationRequest)
	if valuationRec.Code != http.StatusOK {
		t.Fatalf("valuation: expected 200, got %d: %s", valuationRec.Code, valuationRec.Body.String())
	}

	var valuation core.InventoryValuation
	if err := json.NewDecoder(valuationRec.Body).Decode(&valuation); err != nil {
		t.Fatalf("decode valuation: %v", err)
	}
	if valuation.TotalValueCents != 900 || valuation.LedgerInventoryCents != 900 {
		t.Fatalf("unexpected valuation: %+v", valuation)
	}
	if valuation.CogsCents != 100 || !valuation.Balanced {
		t.Fatalf("expected cogs 100 and balanced valuation: %+v", valuation)
	}
}

func TestReserveOversellReturnsConflict(t *testing.T) {
	router := apiRouter()

	create := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{
		"retailer_id": "`+apiRetailer+`",
		"lines": [{"product_id": "`+apiProduct+`", "quantity": 11}]
	}`))
	create.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, create)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create order: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var order core.Order
	if err := json.NewDecoder(createRec.Body).Decode(&order); err != nil {
		t.Fatalf("decode order: %v", err)
	}

	reserve := httptest.NewRequest(http.MethodPost, "/api/orders/"+order.ID+"/reserve", nil)
	reserveRec := httptest.NewRecorder()
	router.ServeHTTP(reserveRec, reserve)
	if reserveRec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", reserveRec.Code, reserveRec.Body.String())
	}
}
