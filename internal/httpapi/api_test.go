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
	apiSupplier  = "00000000-0000-0000-0000-000000000030"
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
	store.AddSupplier(apiTenant, core.Supplier{ID: apiSupplier, Name: "Golden Grain Co."})
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

func TestSalesReturnEndpointUpdatesBalance(t *testing.T) {
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

	reserveRequest := httptest.NewRequest(http.MethodPost, "/api/orders/"+order.ID+"/reserve", nil)
	reserveRec := httptest.NewRecorder()
	router.ServeHTTP(reserveRec, reserveRequest)
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
	if len(invoice.Lines) != 1 {
		t.Fatalf("expected invoice lines, got %+v", invoice.Lines)
	}

	returnRequest := httptest.NewRequest(http.MethodPost, "/api/invoices/"+invoice.ID+"/returns", strings.NewReader(`{
		"lines": [{"product_id": "`+apiProduct+`", "quantity": 1}]
	}`))
	returnRequest.Header.Set("Content-Type", "application/json")
	returnRec := httptest.NewRecorder()
	router.ServeHTTP(returnRec, returnRequest)
	if returnRec.Code != http.StatusCreated {
		t.Fatalf("return: expected 201, got %d: %s", returnRec.Code, returnRec.Body.String())
	}
	var salesReturn core.SalesReturn
	if err := json.NewDecoder(returnRec.Body).Decode(&salesReturn); err != nil {
		t.Fatalf("decode return: %v", err)
	}
	if salesReturn.TotalRefundCents != 1000 || len(salesReturn.Lines) != 1 || !salesReturn.Lines[0].Restock {
		t.Fatalf("unexpected return: %+v", salesReturn)
	}

	creditRequest := httptest.NewRequest(http.MethodGet, "/api/retailers/"+apiRetailer+"/credit", nil)
	creditRec := httptest.NewRecorder()
	router.ServeHTTP(creditRec, creditRequest)
	var credit core.CreditView
	if err := json.NewDecoder(creditRec.Body).Decode(&credit); err != nil {
		t.Fatalf("decode credit: %v", err)
	}
	if credit.ExposureCents != 1000 {
		t.Fatalf("expected 1000 exposure after return, got %+v", credit)
	}

	overReturn := httptest.NewRequest(http.MethodPost, "/api/invoices/"+invoice.ID+"/returns", strings.NewReader(`{
		"lines": [{"product_id": "`+apiProduct+`", "quantity": 2}]
	}`))
	overReturn.Header.Set("Content-Type", "application/json")
	overRec := httptest.NewRecorder()
	router.ServeHTTP(overRec, overReturn)
	if overRec.Code != http.StatusConflict {
		t.Fatalf("expected over-return conflict, got %d: %s", overRec.Code, overRec.Body.String())
	}
}

func TestReadEndpointsReturnRecords(t *testing.T) {
	router := apiRouter()

	create := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{
		"retailer_id": "`+apiRetailer+`",
		"lines": [{"product_id": "`+apiProduct+`", "quantity": 2}]
	}`))
	create.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, create)
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

	returnRequest := httptest.NewRequest(http.MethodPost, "/api/invoices/"+invoice.ID+"/returns", strings.NewReader(`{
		"lines": [{"product_id": "`+apiProduct+`", "quantity": 1, "restock": false}]
	}`))
	returnRequest.Header.Set("Content-Type", "application/json")
	returnRec := httptest.NewRecorder()
	router.ServeHTTP(returnRec, returnRequest)
	if returnRec.Code != http.StatusCreated {
		t.Fatalf("return: expected 201, got %d: %s", returnRec.Code, returnRec.Body.String())
	}

	ordersRec := httptest.NewRecorder()
	router.ServeHTTP(ordersRec, httptest.NewRequest(http.MethodGet, "/api/orders", nil))
	var orders []core.Order
	if err := json.NewDecoder(ordersRec.Body).Decode(&orders); err != nil {
		t.Fatalf("decode orders: %v", err)
	}
	if len(orders) != 1 || len(orders[0].Lines) != 1 {
		t.Fatalf("unexpected orders: %+v", orders)
	}

	invoicesRec := httptest.NewRecorder()
	router.ServeHTTP(invoicesRec, httptest.NewRequest(http.MethodGet, "/api/invoices", nil))
	var invoices []core.Invoice
	if err := json.NewDecoder(invoicesRec.Body).Decode(&invoices); err != nil {
		t.Fatalf("decode invoices: %v", err)
	}
	if len(invoices) != 1 || invoices[0].ReturnedCents != 1000 {
		t.Fatalf("unexpected invoices: %+v", invoices)
	}

	detailRec := httptest.NewRecorder()
	router.ServeHTTP(detailRec, httptest.NewRequest(http.MethodGet, "/api/invoices/"+invoice.ID, nil))
	var detail core.InvoiceDetail
	if err := json.NewDecoder(detailRec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode invoice detail: %v", err)
	}
	if len(detail.Lines) != 1 || len(detail.Returns) != 1 || detail.Returns[0].Lines[0].Restock {
		t.Fatalf("unexpected detail: %+v", detail)
	}

	warehousesRec := httptest.NewRecorder()
	router.ServeHTTP(warehousesRec, httptest.NewRequest(http.MethodGet, "/api/warehouses", nil))
	var warehouses []core.Warehouse
	if err := json.NewDecoder(warehousesRec.Body).Decode(&warehouses); err != nil {
		t.Fatalf("decode warehouses: %v", err)
	}
	if len(warehouses) != 1 || warehouses[0].Name != "Main Warehouse" {
		t.Fatalf("unexpected warehouses: %+v", warehouses)
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

func TestCatalogEndpointsCreateAndUpdate(t *testing.T) {
	router := apiRouter()

	create := httptest.NewRequest(http.MethodPost, "/api/products", strings.NewReader(`{
		"sku": "oil-5", "name": "Cooking oil 5L", "unit": "bottle", "price_cents": 1250
	}`))
	create.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, create)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create product: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var product core.Product
	if err := json.NewDecoder(createRec.Body).Decode(&product); err != nil {
		t.Fatalf("decode product: %v", err)
	}
	if product.ID == "" || product.PriceCents != 1250 {
		t.Fatalf("unexpected product: %+v", product)
	}

	duplicate := httptest.NewRequest(http.MethodPost, "/api/products", strings.NewReader(`{
		"sku": "oil-5", "name": "Another oil", "unit": "bottle", "price_cents": 900
	}`))
	duplicate.Header.Set("Content-Type", "application/json")
	duplicateRec := httptest.NewRecorder()
	router.ServeHTTP(duplicateRec, duplicate)
	if duplicateRec.Code != http.StatusConflict {
		t.Fatalf("duplicate sku: expected 409, got %d: %s", duplicateRec.Code, duplicateRec.Body.String())
	}

	update := httptest.NewRequest(http.MethodPut, "/api/products/"+product.ID, strings.NewReader(`{
		"sku": "oil-5", "name": "Cooking oil 5L", "unit": "bottle", "price_cents": 1400
	}`))
	update.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, update)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update product: expected 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}

	invalid := httptest.NewRequest(http.MethodPost, "/api/retailers", strings.NewReader(`{"name": "  "}`))
	invalid.Header.Set("Content-Type", "application/json")
	invalidRec := httptest.NewRecorder()
	router.ServeHTTP(invalidRec, invalid)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid retailer: expected 400, got %d: %s", invalidRec.Code, invalidRec.Body.String())
	}

	warehouse := httptest.NewRequest(http.MethodPost, "/api/warehouses", strings.NewReader(`{"name": "Overflow"}`))
	warehouse.Header.Set("Content-Type", "application/json")
	warehouseRec := httptest.NewRecorder()
	router.ServeHTTP(warehouseRec, warehouse)
	if warehouseRec.Code != http.StatusCreated {
		t.Fatalf("create warehouse: expected 201, got %d: %s", warehouseRec.Code, warehouseRec.Body.String())
	}

	supplier := httptest.NewRequest(http.MethodPost, "/api/suppliers", strings.NewReader(`{
		"name": "Coastal Foods", "contact_name": "Sam Reed", "phone": "+254 700 111"
	}`))
	supplier.Header.Set("Content-Type", "application/json")
	supplierRec := httptest.NewRecorder()
	router.ServeHTTP(supplierRec, supplier)
	if supplierRec.Code != http.StatusCreated {
		t.Fatalf("create supplier: expected 201, got %d: %s", supplierRec.Code, supplierRec.Body.String())
	}
}

func TestPurchaseOrderReceiveEndpoint(t *testing.T) {
	router := apiRouter()

	create := httptest.NewRequest(http.MethodPost, "/api/purchase-orders", strings.NewReader(`{
		"supplier_id": "`+apiSupplier+`",
		"warehouse_id": "`+apiWarehouse+`",
		"lines": [{"product_id": "`+apiProduct+`", "quantity": 2, "unit_cost_cents": 900}]
	}`))
	create.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, create)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create purchase order: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var purchaseOrder core.PurchaseOrder
	if err := json.NewDecoder(createRec.Body).Decode(&purchaseOrder); err != nil {
		t.Fatalf("decode purchase order: %v", err)
	}
	if purchaseOrder.TotalCents != 1800 || purchaseOrder.Status != "ordered" {
		t.Fatalf("unexpected purchase order: %+v", purchaseOrder)
	}

	receive := httptest.NewRequest(http.MethodPost, "/api/purchase-orders/"+purchaseOrder.ID+"/receive", nil)
	receiveRec := httptest.NewRecorder()
	router.ServeHTTP(receiveRec, receive)
	if receiveRec.Code != http.StatusOK {
		t.Fatalf("receive purchase order: expected 200, got %d: %s", receiveRec.Code, receiveRec.Body.String())
	}

	detail := httptest.NewRequest(http.MethodGet, "/api/purchase-orders/"+purchaseOrder.ID, nil)
	detailRec := httptest.NewRecorder()
	router.ServeHTTP(detailRec, detail)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("get purchase order: expected 200, got %d: %s", detailRec.Code, detailRec.Body.String())
	}
	var received core.PurchaseOrder
	if err := json.NewDecoder(detailRec.Body).Decode(&received); err != nil {
		t.Fatalf("decode purchase order detail: %v", err)
	}
	if received.Status != "received" || len(received.Lines) != 1 || received.ReceivedAt == nil {
		t.Fatalf("unexpected received order: %+v", received)
	}

	stock := httptest.NewRequest(http.MethodGet, "/api/stock", nil)
	stockRec := httptest.NewRecorder()
	router.ServeHTTP(stockRec, stock)
	var levels []core.StockLevel
	if err := json.NewDecoder(stockRec.Body).Decode(&levels); err != nil {
		t.Fatalf("decode stock: %v", err)
	}
	if len(levels) != 1 || levels[0].OnHand != 12 || levels[0].InventoryValueCents != 1800 {
		t.Fatalf("unexpected stock after receive: %+v", levels)
	}

	balance := httptest.NewRequest(http.MethodGet, "/api/ledger/trial-balance", nil)
	balanceRec := httptest.NewRecorder()
	router.ServeHTTP(balanceRec, balance)
	var trial trialBalanceResponse
	if err := json.NewDecoder(balanceRec.Body).Decode(&trial); err != nil {
		t.Fatalf("decode trial balance: %v", err)
	}
	if !trial.Balanced {
		t.Fatalf("expected balanced ledger, got %+v", trial)
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
