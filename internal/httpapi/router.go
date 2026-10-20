package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cypher682/distro/internal/core"
)

// DefaultTenantID is the seeded demo tenant used when no header is supplied.
const DefaultTenantID = "00000000-0000-0000-0000-000000000001"

type Pinger interface {
	Ping(context.Context) error
}

type createOrderRequest struct {
	RetailerID string                 `json:"retailer_id"`
	Lines      []core.CreateOrderLine `json:"lines"`
}

type receiveStockRequest struct {
	ProductID     string `json:"product_id"`
	WarehouseID   string `json:"warehouse_id"`
	Quantity      int64  `json:"quantity"`
	UnitCostCents int64  `json:"unit_cost_cents"`
	Reason        string `json:"reason"`
}

type paymentRequest struct {
	AmountCents int64 `json:"amount_cents"`
}

type returnLineRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Restock   *bool  `json:"restock"`
}

type returnRequest struct {
	Lines []returnLineRequest `json:"lines"`
}

type trialBalanceResponse struct {
	Accounts    []core.AccountBalance `json:"accounts"`
	DebitCents  int64                 `json:"debit_cents"`
	CreditCents int64                 `json:"credit_cents"`
	Balanced    bool                  `json:"balanced"`
}

func NewRouter(db Pinger, svc *core.Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "distro",
		})
	})

	mux.HandleFunc("GET /health/db", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status":   "error",
				"database": "unavailable",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":   "ok",
			"database": "ok",
		})
	})

	mux.HandleFunc("GET /api/products", func(w http.ResponseWriter, r *http.Request) {
		products, err := svc.ListProducts(r.Context(), tenantID(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, products)
	})

	mux.HandleFunc("POST /api/products", func(w http.ResponseWriter, r *http.Request) {
		var request core.ProductInput
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		product, err := svc.CreateProduct(r.Context(), tenantID(r), request)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, product)
	})

	mux.HandleFunc("PUT /api/products/{id}", func(w http.ResponseWriter, r *http.Request) {
		var request core.ProductInput
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		product, err := svc.UpdateProduct(r.Context(), tenantID(r), r.PathValue("id"), request)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, product)
	})

	mux.HandleFunc("GET /api/warehouses", func(w http.ResponseWriter, r *http.Request) {
		warehouses, err := svc.ListWarehouses(r.Context(), tenantID(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, warehouses)
	})

	mux.HandleFunc("POST /api/warehouses", func(w http.ResponseWriter, r *http.Request) {
		var request core.WarehouseInput
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		warehouse, err := svc.CreateWarehouse(r.Context(), tenantID(r), request)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, warehouse)
	})

	mux.HandleFunc("PUT /api/warehouses/{id}", func(w http.ResponseWriter, r *http.Request) {
		var request core.WarehouseInput
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		warehouse, err := svc.UpdateWarehouse(r.Context(), tenantID(r), r.PathValue("id"), request)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, warehouse)
	})

	mux.HandleFunc("GET /api/retailers", func(w http.ResponseWriter, r *http.Request) {
		retailers, err := svc.ListRetailers(r.Context(), tenantID(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, retailers)
	})

	mux.HandleFunc("POST /api/retailers", func(w http.ResponseWriter, r *http.Request) {
		var request core.RetailerInput
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		retailer, err := svc.CreateRetailer(r.Context(), tenantID(r), request)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, retailer)
	})

	mux.HandleFunc("PUT /api/retailers/{id}", func(w http.ResponseWriter, r *http.Request) {
		var request core.RetailerInput
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		retailer, err := svc.UpdateRetailer(r.Context(), tenantID(r), r.PathValue("id"), request)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, retailer)
	})

	mux.HandleFunc("GET /api/suppliers", func(w http.ResponseWriter, r *http.Request) {
		suppliers, err := svc.ListSuppliers(r.Context(), tenantID(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, suppliers)
	})

	mux.HandleFunc("POST /api/suppliers", func(w http.ResponseWriter, r *http.Request) {
		var request core.SupplierInput
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		supplier, err := svc.CreateSupplier(r.Context(), tenantID(r), request)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, supplier)
	})

	mux.HandleFunc("PUT /api/suppliers/{id}", func(w http.ResponseWriter, r *http.Request) {
		var request core.SupplierInput
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		supplier, err := svc.UpdateSupplier(r.Context(), tenantID(r), r.PathValue("id"), request)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, supplier)
	})

	mux.HandleFunc("GET /api/stock", func(w http.ResponseWriter, r *http.Request) {
		stock, err := svc.ListStock(r.Context(), tenantID(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, stock)
	})

	mux.HandleFunc("GET /api/inventory/valuation", func(w http.ResponseWriter, r *http.Request) {
		valuation, err := svc.Valuation(r.Context(), tenantID(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, valuation)
	})

	mux.HandleFunc("POST /api/stock/receive", func(w http.ResponseWriter, r *http.Request) {
		var request receiveStockRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		if err := svc.ReceiveStock(r.Context(), tenantID(r), request.ProductID, request.WarehouseID, request.Quantity, request.UnitCostCents, request.Reason); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/orders", func(w http.ResponseWriter, r *http.Request) {
		var request createOrderRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		order, err := svc.CreateOrder(r.Context(), tenantID(r), request.RetailerID, request.Lines)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, order)
	})

	mux.HandleFunc("GET /api/orders", func(w http.ResponseWriter, r *http.Request) {
		orders, err := svc.ListOrders(r.Context(), tenantID(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, orders)
	})

	mux.HandleFunc("GET /api/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		order, err := svc.GetOrder(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, order)
	})

	mux.HandleFunc("POST /api/orders/{id}/reserve", func(w http.ResponseWriter, r *http.Request) {
		order, err := svc.ReserveOrder(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, order)
	})

	mux.HandleFunc("POST /api/orders/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		order, err := svc.CancelOrder(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, order)
	})

	mux.HandleFunc("POST /api/orders/{id}/delivery-notes", func(w http.ResponseWriter, r *http.Request) {
		note, err := svc.CreateDeliveryNote(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, note)
	})

	mux.HandleFunc("GET /api/delivery-notes", func(w http.ResponseWriter, r *http.Request) {
		notes, err := svc.ListDeliveryNotes(r.Context(), tenantID(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, notes)
	})

	mux.HandleFunc("GET /api/delivery-notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		note, err := svc.GetDeliveryNote(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, note)
	})

	mux.HandleFunc("POST /api/delivery-notes/{id}/dispatch", func(w http.ResponseWriter, r *http.Request) {
		note, err := svc.DispatchDeliveryNote(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, note)
	})

	mux.HandleFunc("POST /api/delivery-notes/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		note, err := svc.CancelDeliveryNote(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, note)
	})

	mux.HandleFunc("POST /api/orders/{id}/invoice", func(w http.ResponseWriter, r *http.Request) {
		invoice, err := svc.InvoiceOrder(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, invoice)
	})

	mux.HandleFunc("GET /api/retailers/{id}/credit", func(w http.ResponseWriter, r *http.Request) {
		credit, err := svc.GetCredit(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, credit)
	})

	mux.HandleFunc("GET /api/invoices", func(w http.ResponseWriter, r *http.Request) {
		invoices, err := svc.ListInvoices(r.Context(), tenantID(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, invoices)
	})

	mux.HandleFunc("GET /api/invoices/{id}", func(w http.ResponseWriter, r *http.Request) {
		invoice, err := svc.GetInvoice(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, invoice)
	})

	mux.HandleFunc("POST /api/invoices/{id}/payments", func(w http.ResponseWriter, r *http.Request) {
		var request paymentRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		payment, err := svc.RecordPayment(r.Context(), tenantID(r), r.PathValue("id"), request.AmountCents)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, payment)
	})

	mux.HandleFunc("POST /api/invoices/{id}/returns", func(w http.ResponseWriter, r *http.Request) {
		var request returnRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		lines := make([]core.CreateReturnLine, 0, len(request.Lines))
		for _, line := range request.Lines {
			restock := true
			if line.Restock != nil {
				restock = *line.Restock
			}
			lines = append(lines, core.CreateReturnLine{
				ProductID: line.ProductID,
				Quantity:  line.Quantity,
				Restock:   restock,
			})
		}
		salesReturn, err := svc.ReturnInvoice(r.Context(), tenantID(r), r.PathValue("id"), lines)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, salesReturn)
	})

	mux.HandleFunc("POST /api/purchase-orders", func(w http.ResponseWriter, r *http.Request) {
		var request core.CreatePurchaseOrderInput
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, err)
			return
		}
		purchaseOrder, err := svc.CreatePurchaseOrder(r.Context(), tenantID(r), request)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, purchaseOrder)
	})

	mux.HandleFunc("GET /api/purchase-orders", func(w http.ResponseWriter, r *http.Request) {
		purchaseOrders, err := svc.ListPurchaseOrders(r.Context(), tenantID(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, purchaseOrders)
	})

	mux.HandleFunc("GET /api/purchase-orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		purchaseOrder, err := svc.GetPurchaseOrder(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, purchaseOrder)
	})

	mux.HandleFunc("POST /api/purchase-orders/{id}/receive", func(w http.ResponseWriter, r *http.Request) {
		purchaseOrder, err := svc.ReceivePurchaseOrder(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, purchaseOrder)
	})

	mux.HandleFunc("POST /api/purchase-orders/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		purchaseOrder, err := svc.CancelPurchaseOrder(r.Context(), tenantID(r), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, purchaseOrder)
	})

	mux.HandleFunc("GET /api/ledger/trial-balance", func(w http.ResponseWriter, r *http.Request) {
		accounts, err := svc.TrialBalance(r.Context(), tenantID(r))
		if err != nil {
			writeError(w, err)
			return
		}

		response := trialBalanceResponse{Accounts: accounts}
		for _, account := range accounts {
			response.DebitCents += account.DebitCents
			response.CreditCents += account.CreditCents
		}
		response.Balanced = response.DebitCents == response.CreditCents
		writeJSON(w, http.StatusOK, response)
	})

	return mux
}

func tenantID(r *http.Request) string {
	if value := r.Header.Get("X-Tenant-ID"); value != "" {
		return value
	}
	return DefaultTenantID
}

func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errors.Join(core.ErrInvalid, err)
	}
	return nil
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal error"

	switch {
	case errors.Is(err, core.ErrNotFound):
		status = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, core.ErrConflict):
		status = http.StatusConflict
		message = err.Error()
	case errors.Is(err, core.ErrInvalid):
		status = http.StatusBadRequest
		message = err.Error()
	default:
		slog.Error("request failed", "error", err)
	}

	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
