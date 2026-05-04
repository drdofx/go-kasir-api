# Production Hardening + TDD Implementation Plan

> **For agentic workers:** Execute tasks sequentially. Each task produces working, verifiable code.

**Goal:** Fix all 29 audit findings, add test infrastructure, and establish TDD workflow

**Architecture:** Security fixes first (P0), then test infrastructure, then TDD for business logic, then operations hardening

**Tech Stack:** Go 1.24, golang-migrate, testify, go-sqlmock, httptest

---

### Task 1: P0 Security Fixes (main.go + .env + docker-compose + Dockerfile)

**Files:**
- Modify: `cmd/api/main.go`
- Modify: `.env`
- Modify: `docker-compose.yml`
- Modify: `Dockerfile`

- [ ] **Step 1: Fix hardcoded admin password + plaintext logging in main.go**

```go
// Before (line 60-67):
func seedAdmin(db *sql.DB, password string) {
	if password == "" {
		password = "admin123"  // HARDCODED FALLBACK
	}
	// ...
	log.Printf("Seeded default admin user (admin / %s)", password)  // PLAINTEXT LOGGING

// After:
func seedAdmin(db *sql.DB, password string) {
	if password == "" {
		log.Println("Skipping admin seed: ADMIN_PASSWORD not set")
		return
	}
	// ...
	log.Println("Seeded default admin user")
}
```

- [ ] **Step 2: Fix path traversal in SPA handler**

Add `"path/filepath"` to imports. Replace the SPA handler (lines 157-168):

```go
mux.HandleFunc("/app/", func(w http.ResponseWriter, r *http.Request) {
    p := strings.TrimPrefix(r.URL.Path, "/app/")
    p = filepath.Clean("/" + p)
    if p == "/" || p == "\\" {
        p = "/index.html"
    }
    fullPath := filepath.Join("web", p)
    absWeb, _ := filepath.Abs("web")
    absPath, _ := filepath.Abs(fullPath)
    if !strings.HasPrefix(absPath, absWeb) {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    if _, err := os.Stat(fullPath); os.IsNotExist(err) {
        http.ServeFile(w, r, "web/index.html")
        return
    }
    http.StripPrefix("/app/", http.FileServer(webFS)).ServeHTTP(w, r)
})
```

- [ ] **Step 3: Fix .env - remove admin password from version control**

```
PORT=8080
DB_CONN=postgres://kasir:kasir123@localhost:5432/kasir?sslmode=disable
CORS_ALLOWED_ORIGIN=http://localhost:5173,http://localhost:8080
LOG_LEVEL=info
```

- [ ] **Step 4: Fix docker-compose.yml - remove hardcoded creds + restrict DB port**

```yaml
environment:
    ADMIN_PASSWORD: ${ADMIN_PASSWORD:-admin123}
  # ...
  POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-kasir123}
ports:
    - "127.0.0.1:5432:5432"
```

- [ ] **Step 5: Fix Dockerfile - pin Alpine version + add non-root user**

```
FROM alpine:3.19
# ... after COPY commands ...
RUN adduser -D -u 1000 appuser
USER appuser
CMD ["./server"]
```

- [ ] **Step 6: Verify build**

Run: `go build ./... && go vet ./...`


### Task 2: Add Security Headers Middleware

**Files:**
- Modify: `internal/middleware/middleware.go`

- [ ] **Step 1: Write the failing test**

Create `internal/middleware/middleware_test.go`:

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := SecurityHeaders()(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %s", resp.Header.Get("X-Content-Type-Options"))
	}
	if resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options: DENY")
	}
	if resp.Header.Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Errorf("expected Referrer-Policy header")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/middleware/ -run TestSecurityHeaders -v`
Expected: FAIL - SecurityHeaders not defined

- [ ] **Step 3: Implement SecurityHeaders middleware**

```go
func SecurityHeaders() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 4: Wire into main.go Chain**

```go
rootHandler := middleware.Chain(
    mux,
    middleware.RequestID(),
    middleware.CORS(config.CORSAllowedOrigin),
    middleware.SecurityHeaders(),
    middleware.Logger(config.LogLevel),
)
```

- [ ] **Step 5: Verify build + test passes**

Run: `go test ./internal/middleware/ -run TestSecurityHeaders -v`
Expected: PASS


### Task 3: Add Test Infrastructure (testify + service interfaces + test helpers)

**Files:**
- Modify: `go.mod` (add testify)
- Create: `internal/testutil/helpers.go`
- Modify: `internal/service/auth_service.go` (add interfaces)
- Modify: `internal/service/product_service.go` (add interfaces)
- Modify: `internal/service/category_service.go` (add interfaces)
- Modify: `internal/service/transaction_service.go` (add interfaces)

- [ ] **Step 1: Install testify**

Run: `go get github.com/stretchr/testify`

- [ ] **Step 2: Write failing test for Service interfaces pattern**

- [ ] **Step 3: Add repository interfaces to each service constructor**

```go
// in internal/service/ interfaces
type ProductRepository interface {
    GetAll(ctx context.Context, name string) ([]model.Product, error)
    GetByID(ctx context.Context, id int) (*model.Product, error)
    Create(ctx context.Context, product *model.Product) error
    Update(ctx context.Context, product *model.Product) error
    Delete(ctx context.Context, id int) error
}

type CategoryRepository interface {
    GetAll(ctx context.Context) ([]model.Category, error)
    GetByID(ctx context.Context, id int) (*model.Category, error)
    Create(ctx context.Context, category *model.Category) error
    Update(ctx context.Context, category *model.Category) error
    Delete(ctx context.Context, id int) error
}
```

- [ ] **Step 4: Create testutil package with HTTP test helpers**

```go
package testutil

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func NewRequest(method, path, body string) *http.Request {
    req := httptest.NewRequest(method, path, strings.NewReader(body))
    return req
}

func AssertStatus(t *testing.T, got, want int) {
    t.Helper()
    if got != want {
        t.Errorf("expected status %d, got %d", want, got)
    }
}

func AssertJSON(t *testing.T, got []byte, want any) {
    t.Helper()
    var gotVal, wantVal any
    if err := json.Unmarshal(got, &gotVal); err != nil {
        t.Fatalf("invalid JSON: %v", err)
    }
    wantBytes, _ := json.Marshal(want)
    json.Unmarshal(wantBytes, &wantVal)
    // simple deep comparison
}
```


### Task 4: TDD - Service Layer Unit Tests (CategoryService)

**Files:**
- Create: `internal/service/category_service_test.go`

- [ ] **Step 1: Write failing test for GetAll**

```go
func TestCategoryService_GetAll(t *testing.T) {
    // ...
}
```

- [ ] **Step 2: Run to verify failure**

- [ ] **Step 3: Implement mock + test**

- [ ] **Step 4: Write failing test for validation (empty name)**

- [ ] **Step 5: Implement validation test**


### Task 5: TDD - Service Layer Unit Tests (TransactionService Checkout)

**Files:**
- Create: `internal/service/transaction_service_test.go`

- [ ] **Step 1: Write failing test for Checkout success path**

- [ ] **Step 2: Write failing test for insufficient stock**

- [ ] **Step 3: Write failing test for rollback on failure**


### Task 6: Fix Remaining High/Medium Issues

**Files:**
- Modify: `internal/handler/helpers.go` (check json.Encode errors)
- Modify: `internal/service/transaction_service.go` (integer overflow check)
- Modify: `internal/middleware/middleware.go` (CORS origin validation)
- Modify: `internal/handler/transaction.go` (max items validation)
- Modify: `internal/handler/helpers.go` (parseID positive check)
- Modify: `internal/handler/auth.go` (forwarded-proto for secure cookie)

- [ ] **Step 1: Fix jsonResponse to check encode errors**

```go
func jsonResponse(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(data); err != nil {
        log.Printf("json encode error: %v", err)
    }
}
```

- [ ] **Step 2: Add integer overflow check in transaction_service.go**

```go
if subtotal > math.MaxInt32 - totalAmount {
    return nil, errors.New("total amount exceeds maximum allowed")
}
```

- [ ] **Step 3: Add max item limit in transaction handler + service**

```go
const maxCheckoutItems = 100
// ...
if len(items) > maxCheckoutItems {
    return nil, fmt.Errorf("maximum %d items per checkout", maxCheckoutItems)
}
```

- [ ] **Step 4: Add X-Forwarded-Proto support for secure cookie**

- [ ] **Step 5: Add positive ID check in parseID**

```go
func parseID(path string, prefix string) (int, error) {
    idStr := strings.TrimPrefix(path, prefix)
    id, err := strconv.Atoi(idStr)
    if err != nil {
        return 0, err
    }
    if id <= 0 {
        return 0, errors.New("id must be positive")
    }
    return id, nil
}
```


### Task 7: Fix Error Message Disclosure

**Files:**
- Modify: `internal/handler/product.go`
- Modify: `internal/handler/category.go`
- Modify: `internal/handler/transaction.go`

- [ ] **Step 1: Replace raw err.Error() in all handlers**

Replace all instances of `http.Error(w, err.Error(), http.StatusInternalServerError)` with generic messages + server-side logging:

```go
// Before:
http.Error(w, err.Error(), http.StatusInternalServerError)
// After:
log.Printf("internal error: %v", err)
http.Error(w, "Internal server error", http.StatusInternalServerError)
```


### Task 8: Final Verification

- [ ] **Step 1: Run full build**

Run: `go build ./...`

- [ ] **Step 2: Run vet**

Run: `go vet ./...`

- [ ] **Step 3: Run all tests**

Run: `go test ./... -v -count=1`

- [ ] **Step 4: Verify all 29 audit findings addressed**

Checklist against audit report.
