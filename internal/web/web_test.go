package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/cypher682/distro/internal/core"
)

const (
	testTenant    = "00000000-0000-0000-0000-000000000001"
	testWarehouse = "00000000-0000-0000-0000-000000000010"
	testRetailer  = "00000000-0000-0000-0000-000000000020"
	testProduct   = "00000000-0000-0000-0000-000000000101"
	testSupplier  = "00000000-0000-0000-0000-000000000030"
)

func testServer(t *testing.T) (http.Handler, *core.MemoryStore, *core.Service) {
	t.Helper()
	store := core.NewMemoryStore()
	store.AddWarehouse(testTenant, testWarehouse, "Main warehouse")
	store.AddRetailer(testTenant, core.Retailer{
		ID:               testRetailer,
		Name:             "Demo retailer",
		CreditLimitCents: 100000,
	})
	store.AddProduct(testTenant, core.Product{
		ID:         testProduct,
		SKU:        "RICE-25",
		Name:       "Rice 25kg",
		Unit:       "bag",
		PriceCents: 1000,
	})
	store.AddSupplier(testTenant, core.Supplier{ID: testSupplier, Name: "Golden Grain Co."})
	store.SetStockValue(testTenant, testProduct, testWarehouse, 10, 1000)
	service := core.NewService(store)
	return New(service, testTenant), store, service
}

func TestPagesRender(t *testing.T) {
	server, _, _ := testServer(t)
	for _, tc := range []struct {
		path string
		want string
	}{
		{"/", "Recent invoices"},
		{"/catalog", "Product catalog"},
		{"/inventory", "Stock ledger"},
		{"/purchasing", "Purchase orders"},
		{"/orders", "Order register"},
		{"/dispatch", "Delivery notes"},
		{"/invoices", "Receivables"},
		{"/ledger", "Trial balance"},
	} {
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", tc.path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), tc.want) {
			t.Fatalf("%s: missing %q", tc.path, tc.want)
		}
	}
}

func TestOrderInvoiceReturnFlow(t *testing.T) {
	server, store, service := testServer(t)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, formRequest(http.MethodPost, "/orders", url.Values{
		"retailer_id": {testRetailer},
		"product_id":  {testProduct},
		"quantity":    {"2"},
	}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create order: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}
	orderID := pathID(t, rec.Header().Get("Location"), "/orders/")

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/orders/"+orderID+"/reserve", nil))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("reserve: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/orders/"+orderID+"/delivery-notes", nil))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create delivery note: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}
	noteID := pathID(t, rec.Header().Get("Location"), "/dispatch/")

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dispatch/"+noteID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("delivery note page: expected 200, got %d", rec.Code)
	}
	for _, want := range []string{"Picking", "Demo retailer", "Rice 25kg"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("delivery note page: missing %q", want)
		}
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/dispatch/"+noteID+"/dispatch", nil))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("dispatch: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dispatch/"+noteID, nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Dispatched") {
		t.Fatalf("dispatched page: expected Dispatched, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/orders/"+orderID+"/invoice", nil))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("invoice: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}
	invoiceID := pathID(t, rec.Header().Get("Location"), "/invoices/")

	invoice, err := service.GetInvoice(context.Background(), testTenant, invoiceID)
	if err != nil {
		t.Fatalf("get invoice: %v", err)
	}
	if len(invoice.Lines) != 1 {
		t.Fatalf("expected one invoice line, got %+v", invoice.Lines)
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, formRequest(http.MethodPost, "/invoices/"+invoiceID+"/payments", url.Values{
		"amount": {"5.00"},
	}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("payment: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, formRequest(http.MethodPost, "/invoices/"+invoiceID+"/returns", url.Values{
		"qty_" + invoice.Lines[0].ID:     {"1"},
		"restock_" + invoice.Lines[0].ID: {"on"},
	}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("return: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/invoices/"+invoiceID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("invoice page: expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Return history", "Restocked", "Payment history", "Rice 25kg"} {
		if !strings.Contains(body, want) {
			t.Fatalf("invoice page: missing %q", want)
		}
	}

	detail, err := store.GetInvoice(context.Background(), testTenant, invoiceID)
	if err != nil {
		t.Fatalf("store invoice: %v", err)
	}
	if len(detail.Returns) != 1 || len(detail.Payments) != 1 {
		t.Fatalf("expected stored return and payment: %+v", detail)
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ledger", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Balanced") {
		t.Fatalf("ledger: expected balanced page, got %d", rec.Code)
	}
}

func TestCatalogSetupFlow(t *testing.T) {
	server, store, service := testServer(t)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, formRequest(http.MethodPost, "/catalog/products", url.Values{
		"sku":   {"OIL-5"},
		"name":  {"Cooking oil 5L"},
		"unit":  {"bottle"},
		"price": {"12.50"},
	}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create product: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	products, err := service.ListProducts(context.Background(), testTenant)
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	var created core.Product
	for _, product := range products {
		if product.SKU == "OIL-5" {
			created = product
		}
	}
	if created.ID == "" || created.PriceCents != 1250 || created.Unit != "bottle" {
		t.Fatalf("unexpected product: %+v", created)
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, formRequest(http.MethodPost, "/catalog/retailers", url.Values{
		"name":         {"North Market"},
		"credit_limit": {"2500"},
	}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create retailer: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, formRequest(http.MethodPost, "/catalog/warehouses", url.Values{
		"name": {"Overflow"},
	}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create warehouse: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/catalog?edit_product="+created.ID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("edit page: expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Edit product", `value="OIL-5"`, `value="12.50"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("edit page: missing %q", want)
		}
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, formRequest(http.MethodPost, "/catalog/products/"+created.ID, url.Values{
		"sku":   {"OIL-5"},
		"name":  {"Cooking oil 5L"},
		"unit":  {"case"},
		"price": {"13.00"},
	}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("update product: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}
	products, err = store.ListProducts(context.Background(), testTenant)
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	var updated core.Product
	for _, product := range products {
		if product.ID == created.ID {
			updated = product
		}
	}
	if updated.Unit != "case" || updated.PriceCents != 1300 {
		t.Fatalf("unexpected updated product: %+v", updated)
	}
}

func TestPurchaseOrderFlow(t *testing.T) {
	server, store, _ := testServer(t)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, formRequest(http.MethodPost, "/purchasing", url.Values{
		"supplier_id":  {testSupplier},
		"warehouse_id": {testWarehouse},
		"product_id":   {testProduct},
		"quantity":     {"2"},
		"unit_cost":    {"1.50"},
	}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create purchase order: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}
	purchaseOrderID := pathID(t, rec.Header().Get("Location"), "/purchasing/")

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/purchasing/"+purchaseOrderID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("purchase order page: expected 200, got %d", rec.Code)
	}
	for _, want := range []string{"Ordered", "Golden Grain Co.", "Rice 25kg", "$3.00"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("purchase order page: missing %q", want)
		}
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/purchasing/"+purchaseOrderID+"/receive", nil))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("receive purchase order: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/purchasing/"+purchaseOrderID, nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Received") {
		t.Fatalf("received page: expected Received, got %d", rec.Code)
	}

	levels, err := store.ListStock(context.Background(), testTenant)
	if err != nil {
		t.Fatalf("list stock: %v", err)
	}
	if len(levels) != 1 || levels[0].OnHand != 12 || levels[0].InventoryValueCents != 1300 {
		t.Fatalf("unexpected stock after receipt: %+v", levels)
	}
}

func TestReturnUnknownInvoiceRedirectsError(t *testing.T) {
	server, _, _ := testServer(t)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, formRequest(http.MethodPost, "/invoices/00000000-0000-0000-0000-000000000099/returns", url.Values{
		"qty_missing": {"1"},
	}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Location"), "error=") {
		t.Fatalf("expected error redirect, got %q", rec.Header().Get("Location"))
	}
}

func TestMissingInvoiceReturns404(t *testing.T) {
	server, _, _ := testServer(t)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/invoices/00000000-0000-0000-0000-000000000099", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "That record does not exist.") {
		t.Fatalf("unexpected error page: %s", rec.Body.String())
	}
}

func TestParseCents(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want int64
	}{
		{"12", 1200},
		{"12.3", 1230},
		{"12.34", 1234},
		{"$1,234.50", 123450},
		{"-2.05", -205},
	} {
		got, err := parseCents(tc.raw)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("parse %q: expected %d, got %d", tc.raw, tc.want, got)
		}
	}
	for _, raw := range []string{"", "1.234", "abc", ".50"} {
		if _, err := parseCents(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

func formRequest(method, path string, values url.Values) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}

func pathID(t *testing.T, location, prefix string) string {
	t.Helper()
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	if !strings.HasPrefix(parsed.Path, prefix) {
		t.Fatalf("unexpected location: %s", location)
	}
	return strings.TrimPrefix(parsed.Path, prefix)
}
