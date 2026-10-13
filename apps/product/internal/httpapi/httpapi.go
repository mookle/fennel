// Package httpapi serves the REST contract in contracts/product.openapi.yaml.
//
// It owns the wire: it parses a request into the types the domain packages
// take, and it maps their results to wire types that own the contract's field
// names. The domain packages carry no JSON tags, so a contract change lands
// here and nowhere else.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// New routes the contract's paths. The contract's server URL ends in /v1, so
// every route carries that prefix.
func New(products productReader) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /v1/products", listProducts{store: products})
	return mux
}

// errorJSON is the contract's Error schema.
type errorJSON struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

// The codes an Error body carries. The contract leaves the vocabulary open,
// so these are this service's own.
const (
	codeInvalidRequest = "invalid_request"
	codeInternal       = "internal"
)

// writeJSON writes v with the status. The status is on the wire before the
// body encodes, so an encoding failure can only be logged.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.ErrorContext(r.Context(), "httpapi: write response", "err", err)
	}
}

// writeError writes an Error body.
func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, details ...string) {
	writeJSON(w, r, status, errorJSON{Code: code, Message: message, Details: details})
}

// writeInternal logs err and answers 500. The body names no cause, because
// the cause is this service's business and not the caller's.
func writeInternal(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "httpapi: "+r.Method+" "+r.URL.Path, "err", err)
	writeError(w, r, http.StatusInternalServerError, codeInternal, "internal error")
}
