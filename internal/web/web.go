package web

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cypher682/distro/internal/core"
)

//go:embed templates/*.html static/*
var assets embed.FS

var pageTemplates = []string{
	"dashboard.html",
	"inventory.html",
	"catalog.html",
	"orders.html",
	"order.html",
	"invoices.html",
	"invoice.html",
	"ledger.html",
	"error.html",
}

type Server struct {
	svc      *core.Service
	tenantID string
	pages    map[string]*template.Template
}

type pageData struct {
	Title         string
	Section       string
	TenantID      string
	Flash         string
	Error         string
	IsError       bool
	Today         time.Time
	Products      []core.Product
	Warehouses    []core.Warehouse
	Retailers     []core.Retailer
	Stock         []core.StockLevel
	Valuation     core.InventoryValuation
	Orders        []core.Order
	Order         core.Order
	Invoices      []core.Invoice
	Invoice       core.InvoiceDetail
	Accounts      []core.AccountBalance
	Credit        []creditRow
	EditProduct   *core.Product
	EditRetailer  *core.Retailer
	EditWarehouse *core.Warehouse
	Linked        *core.Invoice
	ProductsBy    map[string]core.Product
	RetailersBy   map[string]core.Retailer
	WarehousesBy  map[string]core.Warehouse
	ReturnedBy    map[string]int
	DebitCents    int64
	CreditCents   int64
	Balanced      bool
	OpenCents     int64
	Returned      int64
}

type creditRow struct {
	Retailer core.Retailer
	Credit   core.CreditView
}

// New returns the browser console handler for a single tenant.
func New(svc *core.Service, tenantID string) http.Handler {
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	server := &Server{
		svc:      svc,
		tenantID: tenantID,
		pages:    parsePages(),
	}
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /{$}", server.dashboard)
	mux.HandleFunc("GET /inventory", server.inventory)
	mux.HandleFunc("POST /inventory/receive", server.receiveStock)
	mux.HandleFunc("GET /catalog", server.catalog)
	mux.HandleFunc("POST /catalog/products", server.createProduct)
	mux.HandleFunc("POST /catalog/products/{id}", server.updateProduct)
	mux.HandleFunc("POST /catalog/retailers", server.createRetailer)
	mux.HandleFunc("POST /catalog/retailers/{id}", server.updateRetailer)
	mux.HandleFunc("POST /catalog/warehouses", server.createWarehouse)
	mux.HandleFunc("POST /catalog/warehouses/{id}", server.updateWarehouse)
	mux.HandleFunc("GET /orders", server.orders)
	mux.HandleFunc("POST /orders", server.createOrder)
	mux.HandleFunc("GET /orders/{id}", server.order)
	mux.HandleFunc("POST /orders/{id}/reserve", server.reserveOrder)
	mux.HandleFunc("POST /orders/{id}/cancel", server.cancelOrder)
	mux.HandleFunc("POST /orders/{id}/invoice", server.invoiceOrder)
	mux.HandleFunc("GET /invoices", server.invoices)
	mux.HandleFunc("GET /invoices/{id}", server.invoice)
	mux.HandleFunc("POST /invoices/{id}/payments", server.recordPayment)
	mux.HandleFunc("POST /invoices/{id}/returns", server.returnInvoice)
	mux.HandleFunc("GET /ledger", server.ledger)
	return mux
}

func parsePages() map[string]*template.Template {
	funcs := template.FuncMap{
		"money":       money,
		"shortID":     shortID,
		"date":        formatDate,
		"statusClass": statusClass,
		"label":       label,
		"amount":      amount,
		"add":         func(a, b int64) int64 { return a + b },
		"sub":         func(a, b int64) int64 { return a - b },
	}
	out := make(map[string]*template.Template, len(pageTemplates))
	for _, page := range pageTemplates {
		set := template.New("base").Funcs(funcs)
		set = template.Must(set.ParseFS(assets, "templates/base.html", "templates/partials.html", "templates/"+page))
		out[page] = set
	}
	return out
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	stock, err := s.svc.ListStock(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Overview", err)
		return
	}
	valuation, err := s.svc.Valuation(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Overview", err)
		return
	}
	products, err := s.svc.ListProducts(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Overview", err)
		return
	}
	warehouses, err := s.svc.ListWarehouses(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Overview", err)
		return
	}
	retailers, err := s.svc.ListRetailers(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Overview", err)
		return
	}
	invoices, err := s.svc.ListInvoices(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Overview", err)
		return
	}
	orders, err := s.svc.ListOrders(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Overview", err)
		return
	}
	accounts, err := s.svc.TrialBalance(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Overview", err)
		return
	}

	data := s.page("Overview", "Overview", r)
	data.Stock = stock
	data.Valuation = valuation
	data.Products = products
	data.Warehouses = warehouses
	data.Retailers = retailers
	data.Invoices = invoices
	data.Orders = orders
	data.Accounts = accounts
	data.ProductsBy = productMap(products)
	data.WarehousesBy = warehouseMap(warehouses)
	data.RetailersBy = retailerMap(retailers)
	data.Credit = s.creditRows(r, retailers)
	for _, invoice := range invoices {
		data.OpenCents += invoiceBalance(invoice)
		data.Returned += invoice.ReturnedCents
	}
	for _, account := range accounts {
		data.DebitCents += account.DebitCents
		data.CreditCents += account.CreditCents
	}
	data.Balanced = data.DebitCents == data.CreditCents
	if len(data.Invoices) > 6 {
		data.Invoices = data.Invoices[:6]
	}
	s.render(w, "dashboard.html", data)
}

func (s *Server) inventory(w http.ResponseWriter, r *http.Request) {
	stock, err := s.svc.ListStock(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Inventory", err)
		return
	}
	products, err := s.svc.ListProducts(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Inventory", err)
		return
	}
	warehouses, err := s.svc.ListWarehouses(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Inventory", err)
		return
	}
	valuation, err := s.svc.Valuation(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Inventory", err)
		return
	}

	data := s.page("Inventory", "Inventory", r)
	data.Stock = stock
	data.Products = products
	data.Warehouses = warehouses
	data.Valuation = valuation
	data.ProductsBy = productMap(products)
	data.WarehousesBy = warehouseMap(warehouses)
	s.render(w, "inventory.html", data)
}

func (s *Server) receiveStock(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.redirect(w, r, "/inventory", "error", "Could not read the form.")
		return
	}
	quantity, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("quantity")), 10, 64)
	if err != nil {
		s.redirect(w, r, "/inventory", "error", "Quantity must be a whole number.")
		return
	}
	var unitCost int64
	if quantity > 0 {
		unitCost, err = parseCents(r.FormValue("unit_cost"))
		if err != nil {
			s.redirect(w, r, "/inventory", "error", "Unit cost must be a valid amount.")
			return
		}
	}
	err = s.svc.ReceiveStock(
		r.Context(),
		s.tenantID,
		r.FormValue("product_id"),
		r.FormValue("warehouse_id"),
		quantity,
		unitCost,
		r.FormValue("reason"),
	)
	if err != nil {
		s.redirect(w, r, "/inventory", "error", publicError(err))
		return
	}
	s.redirect(w, r, "/inventory", "flash", "Stock movement posted.")
}

func (s *Server) catalog(w http.ResponseWriter, r *http.Request) {
	products, err := s.svc.ListProducts(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Catalog", err)
		return
	}
	retailers, err := s.svc.ListRetailers(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Catalog", err)
		return
	}
	warehouses, err := s.svc.ListWarehouses(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Catalog", err)
		return
	}

	data := s.page("Catalog", "Catalog", r)
	data.Products = products
	data.Retailers = retailers
	data.Warehouses = warehouses
	if id := r.URL.Query().Get("edit_product"); id != "" {
		for i := range products {
			if products[i].ID == id {
				data.EditProduct = &products[i]
				break
			}
		}
	}
	if id := r.URL.Query().Get("edit_retailer"); id != "" {
		for i := range retailers {
			if retailers[i].ID == id {
				data.EditRetailer = &retailers[i]
				break
			}
		}
	}
	if id := r.URL.Query().Get("edit_warehouse"); id != "" {
		for i := range warehouses {
			if warehouses[i].ID == id {
				data.EditWarehouse = &warehouses[i]
				break
			}
		}
	}
	s.render(w, "catalog.html", data)
}

func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.redirect(w, r, "/catalog", "error", "Could not read the form.")
		return
	}
	input, err := productInput(r)
	if err != nil {
		s.redirect(w, r, "/catalog", "error", "Price must be a valid amount.")
		return
	}
	if _, err := s.svc.CreateProduct(r.Context(), s.tenantID, input); err != nil {
		s.redirect(w, r, "/catalog", "error", publicError(err))
		return
	}
	s.redirect(w, r, "/catalog", "flash", "Product created.")
}

func (s *Server) updateProduct(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		s.redirect(w, r, catalogEdit("product", productID), "error", "Could not read the form.")
		return
	}
	input, err := productInput(r)
	if err != nil {
		s.redirect(w, r, catalogEdit("product", productID), "error", "Price must be a valid amount.")
		return
	}
	if _, err := s.svc.UpdateProduct(r.Context(), s.tenantID, productID, input); err != nil {
		s.redirect(w, r, catalogEdit("product", productID), "error", publicError(err))
		return
	}
	s.redirect(w, r, "/catalog", "flash", "Product updated.")
}

func (s *Server) createRetailer(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.redirect(w, r, "/catalog", "error", "Could not read the form.")
		return
	}
	input, err := retailerInput(r)
	if err != nil {
		s.redirect(w, r, "/catalog", "error", "Credit limit must be a valid amount.")
		return
	}
	if _, err := s.svc.CreateRetailer(r.Context(), s.tenantID, input); err != nil {
		s.redirect(w, r, "/catalog", "error", publicError(err))
		return
	}
	s.redirect(w, r, "/catalog", "flash", "Retailer created.")
}

func (s *Server) updateRetailer(w http.ResponseWriter, r *http.Request) {
	retailerID := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		s.redirect(w, r, catalogEdit("retailer", retailerID), "error", "Could not read the form.")
		return
	}
	input, err := retailerInput(r)
	if err != nil {
		s.redirect(w, r, catalogEdit("retailer", retailerID), "error", "Credit limit must be a valid amount.")
		return
	}
	if _, err := s.svc.UpdateRetailer(r.Context(), s.tenantID, retailerID, input); err != nil {
		s.redirect(w, r, catalogEdit("retailer", retailerID), "error", publicError(err))
		return
	}
	s.redirect(w, r, "/catalog", "flash", "Retailer updated.")
}

func (s *Server) createWarehouse(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.redirect(w, r, "/catalog", "error", "Could not read the form.")
		return
	}
	input := core.WarehouseInput{Name: r.FormValue("name")}
	if _, err := s.svc.CreateWarehouse(r.Context(), s.tenantID, input); err != nil {
		s.redirect(w, r, "/catalog", "error", publicError(err))
		return
	}
	s.redirect(w, r, "/catalog", "flash", "Warehouse created.")
}

func (s *Server) updateWarehouse(w http.ResponseWriter, r *http.Request) {
	warehouseID := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		s.redirect(w, r, catalogEdit("warehouse", warehouseID), "error", "Could not read the form.")
		return
	}
	input := core.WarehouseInput{Name: r.FormValue("name")}
	if _, err := s.svc.UpdateWarehouse(r.Context(), s.tenantID, warehouseID, input); err != nil {
		s.redirect(w, r, catalogEdit("warehouse", warehouseID), "error", publicError(err))
		return
	}
	s.redirect(w, r, "/catalog", "flash", "Warehouse updated.")
}

func (s *Server) orders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.svc.ListOrders(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Orders", err)
		return
	}
	products, err := s.svc.ListProducts(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Orders", err)
		return
	}
	retailers, err := s.svc.ListRetailers(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Orders", err)
		return
	}
	invoices, err := s.svc.ListInvoices(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Orders", err)
		return
	}

	data := s.page("Orders", "Orders", r)
	data.Orders = orders
	data.Products = products
	data.Retailers = retailers
	data.Invoices = invoices
	data.ProductsBy = productMap(products)
	data.RetailersBy = retailerMap(retailers)
	s.render(w, "orders.html", data)
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.redirect(w, r, "/orders", "error", "Could not read the form.")
		return
	}
	productIDs := r.Form["product_id"]
	quantities := r.Form["quantity"]
	lines := make([]core.CreateOrderLine, 0, len(productIDs))
	for i, productID := range productIDs {
		productID = strings.TrimSpace(productID)
		quantityRaw := ""
		if i < len(quantities) {
			quantityRaw = strings.TrimSpace(quantities[i])
		}
		if productID == "" && quantityRaw == "" {
			continue
		}
		quantity, err := strconv.Atoi(quantityRaw)
		if err != nil || quantity <= 0 {
			s.redirect(w, r, "/orders", "error", "Each order line needs a product and a positive quantity.")
			return
		}
		lines = append(lines, core.CreateOrderLine{ProductID: productID, Quantity: quantity})
	}
	if len(lines) == 0 {
		s.redirect(w, r, "/orders", "error", "Add at least one order line.")
		return
	}
	order, err := s.svc.CreateOrder(r.Context(), s.tenantID, r.FormValue("retailer_id"), lines)
	if err != nil {
		s.redirect(w, r, "/orders", "error", publicError(err))
		return
	}
	s.redirect(w, r, "/orders/"+order.ID, "flash", "Draft order created.")
}

func (s *Server) order(w http.ResponseWriter, r *http.Request) {
	order, err := s.svc.GetOrder(r.Context(), s.tenantID, r.PathValue("id"))
	if err != nil {
		s.failStatus(w, "Order", err)
		return
	}
	products, err := s.svc.ListProducts(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Order", err)
		return
	}
	retailers, err := s.svc.ListRetailers(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Order", err)
		return
	}
	invoices, err := s.svc.ListInvoices(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Order", err)
		return
	}

	data := s.page("Order "+shortID(order.ID), "Orders", r)
	data.Order = order
	data.Products = products
	data.Retailers = retailers
	data.ProductsBy = productMap(products)
	data.RetailersBy = retailerMap(retailers)
	for i := range invoices {
		if invoices[i].OrderID == order.ID {
			data.Linked = &invoices[i]
			break
		}
	}
	s.render(w, "order.html", data)
}

func (s *Server) reserveOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if _, err := s.svc.ReserveOrder(r.Context(), s.tenantID, orderID); err != nil {
		s.redirect(w, r, "/orders/"+orderID, "error", publicError(err))
		return
	}
	s.redirect(w, r, "/orders/"+orderID, "flash", "Stock reserved and credit checked.")
}

func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if _, err := s.svc.CancelOrder(r.Context(), s.tenantID, orderID); err != nil {
		s.redirect(w, r, "/orders/"+orderID, "error", publicError(err))
		return
	}
	s.redirect(w, r, "/orders/"+orderID, "flash", "Order cancelled.")
}

func (s *Server) invoiceOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	invoice, err := s.svc.InvoiceOrder(r.Context(), s.tenantID, orderID)
	if err != nil {
		s.redirect(w, r, "/orders/"+orderID, "error", publicError(err))
		return
	}
	s.redirect(w, r, "/invoices/"+invoice.ID, "flash", "Order invoiced. Stock, COGS, and revenue posted.")
}

func (s *Server) invoices(w http.ResponseWriter, r *http.Request) {
	invoices, err := s.svc.ListInvoices(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Invoices", err)
		return
	}
	retailers, err := s.svc.ListRetailers(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Invoices", err)
		return
	}

	data := s.page("Invoices", "Invoices", r)
	data.Invoices = invoices
	data.Retailers = retailers
	data.RetailersBy = retailerMap(retailers)
	for _, invoice := range invoices {
		data.OpenCents += invoiceBalance(invoice)
		data.Returned += invoice.ReturnedCents
	}
	s.render(w, "invoices.html", data)
}

func (s *Server) invoice(w http.ResponseWriter, r *http.Request) {
	invoice, err := s.svc.GetInvoice(r.Context(), s.tenantID, r.PathValue("id"))
	if err != nil {
		s.failStatus(w, "Invoice", err)
		return
	}
	products, err := s.svc.ListProducts(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Invoice", err)
		return
	}
	retailers, err := s.svc.ListRetailers(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Invoice", err)
		return
	}
	warehouses, err := s.svc.ListWarehouses(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Invoice", err)
		return
	}

	data := s.page("Invoice "+shortID(invoice.ID), "Invoices", r)
	data.Invoice = invoice
	data.Products = products
	data.Retailers = retailers
	data.Warehouses = warehouses
	data.ProductsBy = productMap(products)
	data.RetailersBy = retailerMap(retailers)
	data.WarehousesBy = warehouseMap(warehouses)
	data.ReturnedBy = make(map[string]int)
	for _, salesReturn := range invoice.Returns {
		for _, line := range salesReturn.Lines {
			data.ReturnedBy[line.InvoiceLineID] += line.Quantity
		}
	}
	s.render(w, "invoice.html", data)
}

func (s *Server) recordPayment(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		s.redirect(w, r, "/invoices/"+invoiceID, "error", "Could not read the form.")
		return
	}
	amount, err := parseCents(r.FormValue("amount"))
	if err != nil || amount <= 0 {
		s.redirect(w, r, "/invoices/"+invoiceID, "error", "Payment must be a positive amount.")
		return
	}
	if _, err := s.svc.RecordPayment(r.Context(), s.tenantID, invoiceID, amount); err != nil {
		s.redirect(w, r, "/invoices/"+invoiceID, "error", publicError(err))
		return
	}
	s.redirect(w, r, "/invoices/"+invoiceID, "flash", "Payment recorded.")
}

func (s *Server) returnInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		s.redirect(w, r, "/invoices/"+invoiceID, "error", "Could not read the form.")
		return
	}
	invoice, err := s.svc.GetInvoice(r.Context(), s.tenantID, invoiceID)
	if err != nil {
		s.redirect(w, r, "/invoices/"+invoiceID, "error", publicError(err))
		return
	}
	lines := make([]core.CreateReturnLine, 0)
	for _, line := range invoice.Lines {
		raw := strings.TrimSpace(r.FormValue("qty_" + line.ID))
		if raw == "" || raw == "0" {
			continue
		}
		quantity, err := strconv.Atoi(raw)
		if err != nil || quantity < 0 {
			s.redirect(w, r, "/invoices/"+invoiceID, "error", "Return quantities must be whole numbers.")
			return
		}
		lines = append(lines, core.CreateReturnLine{
			ProductID: line.ProductID,
			Quantity:  quantity,
			Restock:   r.FormValue("restock_"+line.ID) != "",
		})
	}
	if len(lines) == 0 {
		s.redirect(w, r, "/invoices/"+invoiceID, "error", "Enter a quantity for at least one line.")
		return
	}
	if _, err := s.svc.ReturnInvoice(r.Context(), s.tenantID, invoiceID, lines); err != nil {
		s.redirect(w, r, "/invoices/"+invoiceID, "error", publicError(err))
		return
	}
	s.redirect(w, r, "/invoices/"+invoiceID, "flash", "Return posted. Receivable and inventory updated.")
}

func (s *Server) ledger(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.svc.TrialBalance(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Ledger", err)
		return
	}
	valuation, err := s.svc.Valuation(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Ledger", err)
		return
	}
	products, err := s.svc.ListProducts(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Ledger", err)
		return
	}
	warehouses, err := s.svc.ListWarehouses(r.Context(), s.tenantID)
	if err != nil {
		s.fail(w, "Ledger", err)
		return
	}

	data := s.page("Ledger", "Ledger", r)
	data.Accounts = accounts
	data.Valuation = valuation
	data.ProductsBy = productMap(products)
	data.WarehousesBy = warehouseMap(warehouses)
	for _, account := range accounts {
		data.DebitCents += account.DebitCents
		data.CreditCents += account.CreditCents
	}
	data.Balanced = data.DebitCents == data.CreditCents
	s.render(w, "ledger.html", data)
}

func (s *Server) page(title, section string, r *http.Request) pageData {
	return pageData{
		Title:    title,
		Section:  section,
		TenantID: s.tenantID,
		Flash:    r.URL.Query().Get("flash"),
		Error:    r.URL.Query().Get("error"),
		Today:    time.Now(),
	}
}

func (s *Server) creditRows(r *http.Request, retailers []core.Retailer) []creditRow {
	rows := make([]creditRow, 0, len(retailers))
	for _, retailer := range retailers {
		credit, err := s.svc.GetCredit(r.Context(), s.tenantID, retailer.ID)
		if err != nil {
			continue
		}
		rows = append(rows, creditRow{Retailer: retailer, Credit: credit})
	}
	return rows
}

func (s *Server) render(w http.ResponseWriter, page string, data pageData) {
	set, ok := s.pages[page]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	var body bytes.Buffer
	if err := set.ExecuteTemplate(&body, "base", data); err != nil {
		slog.Error("render page", "page", page, "error", err)
		http.Error(w, "Could not render this page.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = body.WriteTo(w)
}

func (s *Server) fail(w http.ResponseWriter, title string, err error) {
	slog.Error("web request failed", "error", err)
	s.failStatus(w, title, err)
}

func (s *Server) failStatus(w http.ResponseWriter, title string, err error) {
	status := http.StatusInternalServerError
	message := "Could not load this page. Try again."
	switch {
	case errors.Is(err, core.ErrNotFound):
		status = http.StatusNotFound
		message = "That record does not exist."
	case errors.Is(err, core.ErrInvalid):
		status = http.StatusBadRequest
		message = publicError(err)
	}
	data := pageData{Title: title, Section: title, Error: message, IsError: true, Today: time.Now(), TenantID: s.tenantID}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = s.pages["error.html"].ExecuteTemplate(w, "base", data)
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request, path, key, message string) {
	u, err := url.Parse(path)
	if err != nil {
		http.Redirect(w, r, path, http.StatusSeeOther)
		return
	}
	query := u.Query()
	query.Set(key, message)
	u.RawQuery = query.Encode()
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

func productInput(r *http.Request) (core.ProductInput, error) {
	price, err := parseCents(r.FormValue("price"))
	if err != nil {
		return core.ProductInput{}, err
	}
	return core.ProductInput{
		SKU:        r.FormValue("sku"),
		Name:       r.FormValue("name"),
		Unit:       r.FormValue("unit"),
		PriceCents: price,
	}, nil
}

func retailerInput(r *http.Request) (core.RetailerInput, error) {
	limit, err := parseCents(r.FormValue("credit_limit"))
	if err != nil {
		return core.RetailerInput{}, err
	}
	return core.RetailerInput{
		Name:             r.FormValue("name"),
		CreditLimitCents: limit,
	}, nil
}

func catalogEdit(kind, id string) string {
	values := url.Values{}
	values.Set("edit_"+kind, id)
	return "/catalog?" + values.Encode()
}

func publicError(err error) string {
	switch {
	case errors.Is(err, core.ErrInvalid), errors.Is(err, core.ErrConflict), errors.Is(err, core.ErrNotFound):
		return err.Error()
	default:
		slog.Error("unexpected web error", "error", err)
		return "Something went wrong. Try again."
	}
}

func productMap(products []core.Product) map[string]core.Product {
	out := make(map[string]core.Product, len(products))
	for _, product := range products {
		out[product.ID] = product
	}
	return out
}

func retailerMap(retailers []core.Retailer) map[string]core.Retailer {
	out := make(map[string]core.Retailer, len(retailers))
	for _, retailer := range retailers {
		out[retailer.ID] = retailer
	}
	return out
}

func warehouseMap(warehouses []core.Warehouse) map[string]core.Warehouse {
	out := make(map[string]core.Warehouse, len(warehouses))
	for _, warehouse := range warehouses {
		out[warehouse.ID] = warehouse
	}
	return out
}

func (d pageData) DetailBalance() int64 {
	return invoiceBalance(d.Invoice.Invoice)
}

func (d pageData) DetailCost() int64 {
	var total int64
	for _, line := range d.Invoice.Lines {
		total += line.TotalCostCents
	}
	return total
}

func (d pageData) RemainingUnits() int {
	total := 0
	for _, line := range d.Invoice.Lines {
		remaining := line.Quantity - d.ReturnedBy[line.ID]
		if remaining > 0 {
			total += remaining
		}
	}
	return total
}

func (d pageData) ReturnHandling(salesReturn core.SalesReturn) string {
	restocked := 0
	for _, line := range salesReturn.Lines {
		if line.Restock {
			restocked++
		}
	}
	switch {
	case restocked == 0:
		return "Damaged"
	case restocked == len(salesReturn.Lines):
		return "Restocked"
	default:
		return "Mixed"
	}
}

func (d pageData) ReturnHandlingClass(salesReturn core.SalesReturn) string {
	switch d.ReturnHandling(salesReturn) {
	case "Restocked":
		return "good"
	case "Damaged":
		return "warn"
	default:
		return "neutral"
	}
}

func (d pageData) PaidInvoices() int {
	total := 0
	for _, invoice := range d.Invoices {
		if invoice.Status == "paid" {
			total++
		}
	}
	return total
}

func (d pageData) TotalOnHand() int64 {
	var total int64
	for _, item := range d.Stock {
		total += item.OnHand
	}
	return total
}

func (d pageData) TotalReserved() int64 {
	var total int64
	for _, item := range d.Stock {
		total += item.Reserved
	}
	return total
}

func (d pageData) TotalAvailable() int64 {
	var total int64
	for _, item := range d.Stock {
		total += item.Available
	}
	return total
}

func (d pageData) ProductName(id string) string {
	if product, ok := d.ProductsBy[id]; ok {
		return product.Name
	}
	return shortID(id)
}

func (d pageData) ProductSKU(id string) string {
	if product, ok := d.ProductsBy[id]; ok {
		return product.SKU
	}
	return shortID(id)
}

func (d pageData) ProductUnit(id string) string {
	if product, ok := d.ProductsBy[id]; ok {
		return product.Unit
	}
	return "unit"
}

func (d pageData) RetailerName(id string) string {
	if retailer, ok := d.RetailersBy[id]; ok {
		return retailer.Name
	}
	return shortID(id)
}

func (d pageData) WarehouseName(id string) string {
	if warehouse, ok := d.WarehousesBy[id]; ok {
		return warehouse.Name
	}
	return shortID(id)
}

func (d pageData) ReturnedQty(lineID string) int {
	return d.ReturnedBy[lineID]
}

func (d pageData) RemainingQty(lineID string, quantity int) int {
	remaining := quantity - d.ReturnedBy[lineID]
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (d pageData) InvoiceBalance(invoice core.Invoice) int64 {
	return invoiceBalance(invoice)
}

func invoiceBalance(invoice core.Invoice) int64 {
	return invoice.TotalCents - invoice.PaidCents - invoice.ReturnedCents
}

func (d pageData) InvoiceForOrder(orderID string) *core.Invoice {
	for i := range d.Invoices {
		if d.Invoices[i].OrderID == orderID {
			return &d.Invoices[i]
		}
	}
	return nil
}

func amount(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func money(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s$%d.%02d", sign, cents/100, cents%100)
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.Local().Format("02 Jan 2006 15:04")
}

func statusClass(status string) string {
	switch status {
	case "paid", "invoiced":
		return "good"
	case "reserved", "open":
		return "warn"
	case "cancelled", "void":
		return "muted"
	default:
		return "neutral"
	}
}

func label(value string) string {
	value = strings.ReplaceAll(value, "_", " ")
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func parseCents(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "$")
	value = strings.ReplaceAll(value, ",", "")
	if value == "" {
		return 0, errors.New("amount is required")
	}
	negative := strings.HasPrefix(value, "-")
	if negative {
		value = strings.TrimPrefix(value, "-")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, errors.New("invalid amount")
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return 0, errors.New("invalid amount")
		}
		fraction = parts[1] + strings.Repeat("0", 2-len(parts[1]))
	}
	cents, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, err
	}
	total := whole*100 + cents
	if negative {
		total = -total
	}
	return total, nil
}
