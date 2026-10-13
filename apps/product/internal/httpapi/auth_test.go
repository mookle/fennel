package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireBearer(t *testing.T) {
	callers := []Caller{
		{Name: "cart", Token: "cart-token"},
		{Name: "dev", Token: "dev-token"},
	}

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"the first caller's token passes", "Bearer cart-token", http.StatusOK},
		{"the second caller's token passes", "Bearer dev-token", http.StatusOK},
		{"the scheme ignores case", "bearer cart-token", http.StatusOK},
		{"no header is rejected", "", http.StatusUnauthorized},
		{"an unknown token is rejected", "Bearer order-token", http.StatusUnauthorized},
		{"a prefix of a token is rejected", "Bearer cart", http.StatusUnauthorized},
		{"an empty token is rejected", "Bearer ", http.StatusUnauthorized},
		{"another scheme is rejected", "Basic cart-token", http.StatusUnauthorized},
		{"a bare token is rejected", "cart-token", http.StatusUnauthorized},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reached := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reached = true
			})

			req := httptest.NewRequest(http.MethodGet, "/v1/products", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			RequireBearer(callers, next).ServeHTTP(rec, req)

			if rec.Code != c.want {
				t.Fatalf("status = %d, want %d", rec.Code, c.want)
			}
			if reached != (c.want == http.StatusOK) {
				t.Errorf("next reached = %v, want %v", reached, c.want == http.StatusOK)
			}
			if c.want != http.StatusUnauthorized {
				return
			}

			if got := rec.Header().Get("WWW-Authenticate"); got != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", got)
			}
			var body errorJSON
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode %s: %v", rec.Body, err)
			}
			if body.Code != codeUnauthorized {
				t.Errorf("code = %q, want %q", body.Code, codeUnauthorized)
			}
		})
	}
}

func TestRequireBearerWithNoCallersRejectsEverything(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next was reached")
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/products", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	RequireBearer(nil, next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
