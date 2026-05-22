# Phase 2: Customers + Payments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add customer management (CRUD + purchase history), payment types, and split payment support in checkout.

**Architecture:** New `internal/domain/customer/` and `internal/domain/payment/` packages following the existing pattern. Modified `internal/domain/transaction/` to support customer_id and split payments.

**Tech Stack:** Go 1.24, net/http, PostgreSQL, golang-migrate

---

## File Structure

```
internal/domain/
├── customer/              [NEW]
│   ├── handler.go         Customer CRUD + purchase history
│   ├── service.go         Validation + business logic
│   └── repository.go      DB operations
├── payment/               [NEW]
│   ├── handler.go         Payment type listing
│   ├── service.go         Payment processing
│   └── repository.go      DB operations
└── transaction/           [MODIFY]
    ├── handler.go         Updated CheckoutRequest with customer_id + payments
    ├── service.go         Handle split payments + customer validation
    └── repository.go      Insert payments, link customer

migrations/
├── 000006_create_customers.up.sql
├── 000006_create_customers.down.sql
├── 000007_create_payment_tables.up.sql
└── 000007_create_payment_tables.down.sql

cmd/api/main.go            [MODIFY] Add customer + payment routes, seed payment types
```

### Type Definitions

```go
// internal/domain/customer/handler.go
type CustomerRequest struct {
    Name    string  `json:"name"`
    Phone   string  `json:"phone"`
    Email   string  `json:"email"`
    Address string  `json:"address"`
}
type CustomerResponse struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Phone     string    `json:"phone"`
    Email     string    `json:"email"`
    Address   string    `json:"address"`
    CreatedAt string    `json:"created_at"`
}
type CustomerPurchaseHistory struct {
    Customer   CustomerResponse    `json:"customer"`
    Purchases  []PurchaseItem      `json:"purchases"`
}
type PurchaseItem struct {
    TransactionID int    `json:"transaction_id"`
    TotalAmount   int    `json:"total_amount"`
    CreatedAt     string `json:"created_at"`
}

// internal/domain/payment/handler.go
type PaymentTypeResponse struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

// Updated internal/domain/transaction/handler.go
type CheckoutPayment struct {
    Type   string `json:"type"`
    Amount int    `json:"amount"`
}
type CheckoutRequest struct {
    Items      []CheckoutItem   `json:"items"`
    CustomerID *int             `json:"customer_id"`
    Payments   []CheckoutPayment `json:"payments"`
}
```

---

## Tasks

### Task 1: Create migrations

**Files:**
- Create: `migrations/000006_create_customers.up.sql`
- Create: `migrations/000006_create_customers.down.sql`
- Create: `migrations/000007_create_payment_tables.up.sql`
- Create: `migrations/000007_create_payment_tables.down.sql`

#### `000006_create_customers.up.sql`
```sql
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL DEFAULT '',
    email VARCHAR(255) NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE transactions ADD COLUMN IF NOT EXISTS customer_id INTEGER REFERENCES customers(id);
```

#### `000006_create_customers.down.sql`
```sql
ALTER TABLE transactions DROP COLUMN IF EXISTS customer_id;
DROP TABLE IF EXISTS customers;
```

#### `000007_create_payment_tables.up.sql`
```sql
CREATE TABLE payment_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

INSERT INTO payment_types (name) VALUES ('cash'), ('card'), ('qris') ON CONFLICT (name) DO NOTHING;

CREATE TABLE transaction_payments (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER REFERENCES transactions(id) ON DELETE CASCADE,
    payment_type_id INTEGER REFERENCES payment_types(id),
    amount INTEGER NOT NULL CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_transaction_payments_transaction_id ON transaction_payments(transaction_id);
```

#### `000007_create_payment_tables.down.sql`
```sql
DROP INDEX IF EXISTS idx_transaction_payments_transaction_id;
DROP TABLE IF EXISTS transaction_payments;
DROP TABLE IF EXISTS payment_types;
```

---

### Task 2: Create customer domain package

**Files:**
- Create: `internal/domain/customer/repository.go`
- Create: `internal/domain/customer/service.go`
- Create: `internal/domain/customer/handler.go`
- Test: `internal/domain/customer/customer_service_test.go`

#### `repository.go`
```go
package customer

import (
    "database/sql"
    "fmt"
    "time"
)

type Customer struct {
    ID        int
    Name      string
    Phone     string
    Email     string
    Address   string
    CreatedAt time.Time
}

type PurchaseRecord struct {
    TransactionID int
    TotalAmount   int
    CreatedAt     string
}

type customerRepository struct {
    db *sql.DB
}

type CustomerRepository interface {
    FindAll(search string) ([]Customer, error)
    FindByID(id int) (*Customer, error)
    Create(c *Customer) error
    Update(c *Customer) error
    Delete(id int) error
    GetPurchaseHistory(customerID int) ([]PurchaseRecord, error)
}

func NewCustomerRepository(db *sql.DB) CustomerRepository {
    return &customerRepository{db: db}
}

func (r *customerRepository) FindAll(search string) ([]Customer, error) {
    query := "SELECT id, name, phone, email, address, created_at FROM customers"
    var args []interface{}
    if search != "" {
        query += " WHERE name ILIKE $1 OR phone ILIKE $1"
        args = append(args, "%"+search+"%")
    }
    query += " ORDER BY id"
    rows, err := r.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var customers []Customer
    for rows.Next() {
        var c Customer
        if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Address, &c.CreatedAt); err != nil {
            return nil, err
        }
        customers = append(customers, c)
    }
    return customers, rows.Err()
}

func (r *customerRepository) FindByID(id int) (*Customer, error) {
    row := r.db.QueryRow("SELECT id, name, phone, email, address, created_at FROM customers WHERE id = $1", id)
    c := &Customer{}
    if err := row.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Address, &c.CreatedAt); err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    return c, nil
}

func (r *customerRepository) Create(c *Customer) error {
    return r.db.QueryRow(
        "INSERT INTO customers (name, phone, email, address) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
        c.Name, c.Phone, c.Email, c.Address,
    ).Scan(&c.ID, &c.CreatedAt)
}

func (r *customerRepository) Update(c *Customer) error {
    _, err := r.db.Exec("UPDATE customers SET name=$1, phone=$2, email=$3, address=$4 WHERE id=$5",
        c.Name, c.Phone, c.Email, c.Address, c.ID)
    return err
}

func (r *customerRepository) Delete(id int) error {
    _, err := r.db.Exec("DELETE FROM customers WHERE id = $1", id)
    return err
}

func (r *customerRepository) GetPurchaseHistory(customerID int) ([]PurchaseRecord, error) {
    rows, err := r.db.Query("SELECT id, total_amount, created_at FROM transactions WHERE customer_id = $1 ORDER BY created_at DESC", customerID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var records []PurchaseRecord
    for rows.Next() {
        var pr PurchaseRecord
        var createdAt interface{}
        if err := rows.Scan(&pr.TransactionID, &pr.TotalAmount, &createdAt); err != nil {
            return nil, err
        }
        pr.CreatedAt = fmt.Sprintf("%v", createdAt)
        records = append(records, pr)
    }
    return records, rows.Err()
}
```

#### `service.go`
```go
package customer

import "errors"

var (
    ErrCustomerNameRequired = errors.New("customer name is required")
    ErrCustomerNotFound     = errors.New("customer not found")
)

type CustomerService struct {
    repo CustomerRepository
}

func NewCustomerService(repo CustomerRepository) *CustomerService {
    return &CustomerService{repo: repo}
}

func (s *CustomerService) FindAll(search string) ([]Customer, error) {
    return s.repo.FindAll(search)
}

func (s *CustomerService) FindByID(id int) (*Customer, error) {
    return s.repo.FindByID(id)
}

func (s *CustomerService) Create(c *Customer) error {
    if c.Name == "" {
        return ErrCustomerNameRequired
    }
    return s.repo.Create(c)
}

func (s *CustomerService) Update(c *Customer) error {
    if c.Name == "" {
        return ErrCustomerNameRequired
    }
    existing, err := s.repo.FindByID(c.ID)
    if err != nil {
        return err
    }
    if existing == nil {
        return ErrCustomerNotFound
    }
    return s.repo.Update(c)
}

func (s *CustomerService) Delete(id int) error {
    existing, err := s.repo.FindByID(id)
    if err != nil {
        return err
    }
    if existing == nil {
        return ErrCustomerNotFound
    }
    return s.repo.Delete(id)
}

func (s *CustomerService) GetPurchaseHistory(customerID int) ([]PurchaseRecord, error) {
    return s.repo.GetPurchaseHistory(customerID)
}
```

#### `handler.go`
```go
package customer

import (
    "encoding/json"
    "net/http"
    "strconv"
    "go-kasir-api/internal/pkg/helpers"
)

type CustomerHandler struct {
    service *CustomerService
}

func NewCustomerHandler(service *CustomerService) *CustomerHandler {
    return &CustomerHandler{service: service}
}

type customerRequest struct {
    Name    string `json:"name"`
    Phone   string `json:"phone"`
    Email   string `json:"email"`
    Address string `json:"address"`
}

type customerResponse struct {
    ID        int    `json:"id"`
    Name      string `json:"name"`
    Phone     string `json:"phone"`
    Email     string `json:"email"`
    Address   string `json:"address"`
    CreatedAt string `json:"created_at"`
}

type purchaseItemResponse struct {
    TransactionID int    `json:"transaction_id"`
    TotalAmount   int    `json:"total_amount"`
    CreatedAt     string `json:"created_at"`
}

type purchaseHistoryResponse struct {
    Customer  customerResponse      `json:"customer"`
    Purchases []purchaseItemResponse `json:"purchases"`
}

func toCustomerResponse(c Customer) customerResponse {
    return customerResponse{
        ID:        c.ID,
        Name:      c.Name,
        Phone:     c.Phone,
        Email:     c.Email,
        Address:   c.Address,
        CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z"),
    }
}

func toCustomerResponses(customers []Customer) []customerResponse {
    res := make([]customerResponse, len(customers))
    for i, c := range customers {
        res[i] = toCustomerResponse(c)
    }
    return res
}

func (h *CustomerHandler) HandleCustomers(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        h.list(w, r)
    case http.MethodPost:
        h.create(w, r)
    default:
        helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
    }
}

func (h *CustomerHandler) HandleCustomerByID(w http.ResponseWriter, r *http.Request) {
    idStr := r.PathValue("id")
    if idStr == "" {
        helpers.WriteError(w, http.StatusBadRequest, "id is required")
        return
    }
    id, err := strconv.Atoi(idStr)
    if err != nil || id <= 0 {
        helpers.WriteError(w, http.StatusBadRequest, "invalid id")
        return
    }
    switch r.Method {
    case http.MethodGet:
        h.getByID(w, id)
    case http.MethodPut:
        h.update(w, r, id)
    case http.MethodDelete:
        h.delete(w, id)
    default:
        helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
    }
}

func (h *CustomerHandler) list(w http.ResponseWriter, r *http.Request) {
    search := r.URL.Query().Get("search")
    customers, err := h.service.FindAll(search)
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    if customers == nil {
        customers = []Customer{}
    }
    helpers.WriteJSON(w, http.StatusOK, toCustomerResponses(customers))
}

func (h *CustomerHandler) create(w http.ResponseWriter, r *http.Request) {
    var req customerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    c := &Customer{Name: req.Name, Phone: req.Phone, Email: req.Email, Address: req.Address}
    if err := h.service.Create(c); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    helpers.WriteJSON(w, http.StatusCreated, toCustomerResponse(*c))
}

func (h *CustomerHandler) getByID(w http.ResponseWriter, id int) {
    c, err := h.service.FindByID(id)
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    if c == nil {
        helpers.WriteError(w, http.StatusNotFound, "customer not found")
        return
    }
    helpers.WriteJSON(w, http.StatusOK, toCustomerResponse(*c))
}

func (h *CustomerHandler) update(w http.ResponseWriter, r *http.Request, id int) {
    var req customerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    c := &Customer{ID: id, Name: req.Name, Phone: req.Phone, Email: req.Email, Address: req.Address}
    if err := h.service.Update(c); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    helpers.WriteJSON(w, http.StatusOK, toCustomerResponse(*c))
}

func (h *CustomerHandler) delete(w http.ResponseWriter, id int) {
    if err := h.service.Delete(id); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

#### `customer_service_test.go`
```go
package customer

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type mockCustomerRepo struct {
    mock.Mock
}

func (m *mockCustomerRepo) FindAll(search string) ([]Customer, error) {
    args := m.Called(search)
    return args.Get(0).([]Customer), args.Error(1)
}

func (m *mockCustomerRepo) FindByID(id int) (*Customer, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Customer), args.Error(1)
}

func (m *mockCustomerRepo) Create(c *Customer) error {
    args := m.Called(c)
    return args.Error(0)
}

func (m *mockCustomerRepo) Update(c *Customer) error {
    args := m.Called(c)
    return args.Error(0)
}

func (m *mockCustomerRepo) Delete(id int) error {
    args := m.Called(id)
    return args.Error(0)
}

func (m *mockCustomerRepo) GetPurchaseHistory(customerID int) ([]PurchaseRecord, error) {
    args := m.Called(customerID)
    return args.Get(0).([]PurchaseRecord), args.Error(1)
}

func TestCustomerService_Create_Valid(t *testing.T) {
    repo := new(mockCustomerRepo)
    svc := NewCustomerService(repo)
    c := &Customer{Name: "Budi"}
    repo.On("Create", c).Return(nil)
    err := svc.Create(c)
    assert.NoError(t, err)
    repo.AssertExpectations(t)
}

func TestCustomerService_Create_NoName(t *testing.T) {
    repo := new(mockCustomerRepo)
    svc := NewCustomerService(repo)
    err := svc.Create(&Customer{Name: ""})
    assert.ErrorIs(t, err, ErrCustomerNameRequired)
}

func TestCustomerService_GetByID_NotFound(t *testing.T) {
    repo := new(mockCustomerRepo)
    svc := NewCustomerService(repo)
    repo.On("FindByID", 999).Return(nil, nil)
    result, err := svc.FindByID(999)
    assert.NoError(t, err)
    assert.Nil(t, result)
}

func TestCustomerService_Delete_NotFound(t *testing.T) {
    repo := new(mockCustomerRepo)
    svc := NewCustomerService(repo)
    repo.On("FindByID", 999).Return(nil, nil)
    err := svc.Delete(999)
    assert.ErrorIs(t, err, ErrCustomerNotFound)
}
```

---

### Task 3: Create payment domain package

**Files:**
- Create: `internal/domain/payment/repository.go`
- Create: `internal/domain/payment/service.go`
- Create: `internal/domain/payment/handler.go`

#### `repository.go`
```go
package payment

import "database/sql"

type PaymentType struct {
    ID   int
    Name string
}

type TransactionPayment struct {
    TransactionID int
    PaymentTypeID int
    Amount        int
}

type paymentRepository struct {
    db *sql.DB
}

type PaymentRepository interface {
    FindAllPaymentTypes() ([]PaymentType, error)
    InsertPayment(tx *sql.Tx, p TransactionPayment) error
}

func NewPaymentRepository(db *sql.DB) PaymentRepository {
    return &paymentRepository{db: db}
}

func (r *paymentRepository) FindAllPaymentTypes() ([]PaymentType, error) {
    rows, err := r.db.Query("SELECT id, name FROM payment_types ORDER BY id")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var types []PaymentType
    for rows.Next() {
        var pt PaymentType
        if err := rows.Scan(&pt.ID, &pt.Name); err != nil {
            return nil, err
        }
        types = append(types, pt)
    }
    return types, rows.Err()
}

func (r *paymentRepository) InsertPayment(tx *sql.Tx, p TransactionPayment) error {
    _, err := tx.Exec("INSERT INTO transaction_payments (transaction_id, payment_type_id, amount) VALUES ($1, $2, $3)",
        p.TransactionID, p.PaymentTypeID, p.Amount)
    return err
}
```

#### `service.go`
```go
package payment

import "github.com/lib/pq"

type PaymentService struct {
    repo PaymentRepository
}

func NewPaymentService(repo PaymentRepository) *PaymentService {
    return &PaymentService{repo: repo}
}

func (s *PaymentService) GetPaymentTypes() ([]PaymentType, error) {
    return s.repo.FindAllPaymentTypes()
}

func (s *PaymentService) GetPaymentTypeIDByName(tx *pq.Transaction, name string) (int, error) {
    // This is a simplified approach - in practice you'd cache or query
    types, err := s.repo.FindAllPaymentTypes()
    if err != nil {
        return 0, err
    }
    for _, pt := range types {
        if pt.Name == name {
            return pt.ID, nil
        }
    }
    return 0, nil
}
```

Handle the dependency on `*pq.Transaction` - actually, this shouldn't use `pq.Transaction`. The payment service's insert operations use a `*sql.Tx`. Let me fix this:

#### `service.go` (corrected)
```go
package payment

import "errors"

var ErrPaymentTypeNotFound = errors.New("payment type not found")

type PaymentService struct {
    repo PaymentRepository
}

func NewPaymentService(repo PaymentRepository) *PaymentService {
    return &PaymentService{repo: repo}
}

func (s *PaymentService) GetPaymentTypes() ([]PaymentType, error) {
    return s.repo.FindAllPaymentTypes()
}

func (s *PaymentService) GetPaymentTypeIDByName(name string) (int, error) {
    types, err := s.repo.FindAllPaymentTypes()
    if err != nil {
        return 0, err
    }
    for _, pt := range types {
        if pt.Name == name {
            return pt.ID, nil
        }
    }
    return 0, ErrPaymentTypeNotFound
}
```

#### `handler.go`
```go
package payment

import (
    "net/http"
    "go-kasir-api/internal/pkg/helpers"
)

type PaymentHandler struct {
    service *PaymentService
}

func NewPaymentHandler(service *PaymentService) *PaymentHandler {
    return &PaymentHandler{service: service}
}

type paymentTypeResponse struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func toPaymentTypeResponses(types []PaymentType) []paymentTypeResponse {
    res := make([]paymentTypeResponse, len(types))
    for i, pt := range types {
        res[i] = paymentTypeResponse{ID: pt.ID, Name: pt.Name}
    }
    return res
}

func (h *PaymentHandler) HandlePaymentTypes(w http.ResponseWriter, r *http.Request) {
    types, err := h.service.GetPaymentTypes()
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    if types == nil {
        types = []PaymentType{}
    }
    helpers.WriteJSON(w, http.StatusOK, toPaymentTypeResponses(types))
}
```

---

### Task 4: Update transaction domain for customer + payments

**Files:**
- Modify: `internal/domain/transaction/handler.go`
- Modify: `internal/domain/transaction/service.go`
- Modify: `internal/domain/transaction/repository.go`

#### handler.go changes

Add new types at the top:
```go
type checkoutPayment struct {
    Type   string `json:"type"`
    Amount int    `json:"amount"`
}

type checkoutRequest struct {
    Items      []CheckoutItem   `json:"items"`
    CustomerID *int             `json:"customer_id"`
    Payments   []checkoutPayment `json:"payments"`
}
```

Update HandleCheckout:
```go
func (h *TransactionHandler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
    var req checkoutRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    checkoutReq := CheckoutRequest{
        Items:      req.Items,
        CustomerID: req.CustomerID,
    }
    for _, p := range req.Payments {
        checkoutReq.Payments = append(checkoutReq.Payments, CheckoutPayment{
            Type:   p.Type,
            Amount: p.Amount,
        })
    }
    txn, err := h.service.Checkout(checkoutReq)
    if err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    helpers.WriteJSON(w, http.StatusCreated, toTransactionResponse(*txn))
}
```

#### service.go changes

New types:
```go
type CheckoutPayment struct {
    Type   string `json:"type"`
    Amount int    `json:"amount"`
}

type CheckoutRequest struct {
    Items      []CheckoutItem   `json:"items"`
    CustomerID *int             `json:"customer_id"`
    Payments   []CheckoutPayment `json:"payments"`
}
```

Requires a `CustomerRepository` interface and `PaymentService` dependency:

```go
type CustomerRepository interface {
    FindByID(id int) (*Customer, error)
}

type Customer struct {
    ID   int
    Name string
}

type PaymentService interface {
    GetPaymentTypeIDByName(name string) (int, error)
    InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error
}
```

Updated TransactionService:
```go
type TransactionService struct {
    repo            TransactionRepository
    customerRepo    CustomerRepository
    paymentSvc      PaymentService
}

func NewTransactionService(repo TransactionRepository, customerRepo CustomerRepository, paymentSvc PaymentService) *TransactionService {
    return &TransactionService{repo: repo, customerRepo: customerRepo, paymentSvc: paymentSvc}
}
```

Update Checkout method - add customer validation and payment handling:
```go
func (s *TransactionService) Checkout(req CheckoutRequest) (*Transaction, error) {
    if len(req.Items) == 0 {
        return nil, ErrCheckoutEmpty
    }
    if len(req.Items) > maxItemsPerCheckout {
        return nil, ErrCheckoutTooMany
    }
    for _, item := range req.Items {
        if item.Quantity <= 0 {
            return nil, fmt.Errorf("%w for product %d", ErrInvalidQuantity, item.ProductID)
        }
    }

    // Validate customer if provided
    if req.CustomerID != nil && *req.CustomerID > 0 {
        customer, err := s.customerRepo.FindByID(*req.CustomerID)
        if err != nil {
            return nil, fmt.Errorf("find customer: %w", err)
        }
        if customer == nil {
            return nil, errors.New("customer not found")
        }
    }

    productIDs := make([]int, len(req.Items))
    for i, item := range req.Items {
        productIDs[i] = item.ProductID
    }

    tx, err := s.repo.BeginTx()
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    products, err := s.repo.LockProducts(tx, productIDs)
    if err != nil {
        return nil, fmt.Errorf("failed to lock products: %w", err)
    }

    productMap := make(map[int]LockedProduct)
    for _, p := range products {
        productMap[p.ID] = p
    }

    for _, item := range req.Items {
        p, ok := productMap[item.ProductID]
        if !ok {
            return nil, fmt.Errorf("%w: product %d", ErrProductNotFound, item.ProductID)
        }
        if p.Stock < item.Quantity {
            return nil, fmt.Errorf("%w for product %s (available: %d, requested: %d)",
                ErrInsufficientStock, p.Name, p.Stock, item.Quantity)
        }
    }

    var totalAmount int
    for _, item := range req.Items {
        p := productMap[item.ProductID]
        subtotal := p.Price * item.Quantity
        if totalAmount > 1<<31-1-subtotal {
            return nil, ErrAmountOverflow
        }
        totalAmount += subtotal
        if err := s.repo.UpdateStock(tx, item.ProductID, item.Quantity); err != nil {
            return nil, fmt.Errorf("failed to update stock for product %d: %w", item.ProductID, err)
        }
    }

    // Validate payments if provided
    if len(req.Payments) > 0 {
        var paymentTotal int
        for _, p := range req.Payments {
            if p.Amount <= 0 {
                return nil, fmt.Errorf("invalid payment amount: %d", p.Amount)
            }
            typeID, err := s.paymentSvc.GetPaymentTypeIDByName(p.Type)
            if err != nil {
                return nil, fmt.Errorf("invalid payment type: %s", p.Type)
            }
            if typeID == 0 {
                return nil, fmt.Errorf("payment type not found: %s", p.Type)
            }
            if paymentTotal > 1<<31-1-p.Amount {
                return nil, ErrAmountOverflow
            }
            paymentTotal += p.Amount
        }
        if paymentTotal != totalAmount {
            return nil, fmt.Errorf("payment total (%d) does not match total amount (%d)", paymentTotal, totalAmount)
        }
    }

    transactionID, err := s.repo.InsertTransaction(tx, totalAmount, req.CustomerID)
    if err != nil {
        return nil, fmt.Errorf("failed to create transaction: %w", err)
    }

    if err := s.repo.InsertDetails(tx, transactionID, req.Items, products); err != nil {
        return nil, fmt.Errorf("failed to insert transaction details: %w", err)
    }

    // Insert payments
    if len(req.Payments) > 0 {
        for _, p := range req.Payments {
            typeID, _ := s.paymentSvc.GetPaymentTypeIDByName(p.Type)
            if err := s.repo.InsertPayment(tx, transactionID, typeID, p.Amount); err != nil {
                return nil, fmt.Errorf("failed to insert payment: %w", err)
            }
        }
    }

    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }

    return s.repo.FindByID(transactionID)
}
```

#### repository.go changes

Add to TransactionRepository interface:
```go
InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error
```

Change InsertTransaction to accept optional customerID:
```go
InsertTransaction(tx *sql.Tx, total int, customerID *int) (int, error)
```

Implementation:
```go
type NullableInt struct {
    Valid bool
    Int   int
}

func (r *transactionRepository) InsertTransaction(tx *sql.Tx, total int, customerID *int) (int, error) {
    var id int
    if customerID != nil && *customerID > 0 {
        err := tx.QueryRow("INSERT INTO transactions (total_amount, customer_id) VALUES ($1, $2) RETURNING id", total, *customerID).Scan(&id)
        return id, err
    }
    err := tx.QueryRow("INSERT INTO transactions (total_amount) VALUES ($1) RETURNING id", total).Scan(&id)
    return id, err
}

func (r *transactionRepository) InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error {
    _, err := tx.Exec("INSERT INTO transaction_payments (transaction_id, payment_type_id, amount) VALUES ($1, $2, $3)",
        transactionID, paymentTypeID, amount)
    return err
}
```

---

### Task 5: Update main.go

**Files:**
- Modify: `cmd/api/main.go`

Add imports:
```go
"go-kasir-api/internal/domain/customer"
"go-kasir-api/internal/domain/payment"
```

Add to main():
```go
customerRepo := customer.NewCustomerRepository(db)
paymentRepo := payment.NewPaymentRepository(db)

customerService := customer.NewCustomerService(customerRepo)
paymentService := payment.NewPaymentService(paymentRepo)

// Update transaction service with customer repo and payment service
transactionService := transaction.NewTransactionService(transactionRepo, customerRepo, paymentService)

customerHandler := customer.NewCustomerHandler(customerService)
paymentHandler := payment.NewPaymentHandler(paymentService)
```

Add V1 routes:
```go
// Customers
mux.HandleFunc("/api/v1/customers", middleware.Chain(customerHandler.HandleCustomers, jwtMiddleware))
mux.HandleFunc("/api/v1/customers/", middleware.Chain(customerHandler.HandleCustomerByID, jwtMiddleware))

// Payment types
mux.HandleFunc("/api/v1/payment-types", middleware.Chain(paymentHandler.HandlePaymentTypes, jwtMiddleware))
```

---

### Task 6: Write tests for updated transaction service

**Files:**
- Modify: `internal/domain/transaction/transaction_service_test.go`

Add mock implementations for CustomerRepository and PaymentService:
```go
type mockCustomerRepoForTransaction struct {
    mock.Mock
}

func (m *mockCustomerRepoForTransaction) FindByID(id int) (*customerInternal, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*customerInternal), args.Error(1)
}

type mockPaymentSvc struct {
    mock.Mock
}

func (m *mockPaymentSvc) GetPaymentTypeIDByName(name string) (int, error) {
    args := m.Called(name)
    return args.Int(0), args.Error(1)
}

func (m *mockPaymentSvc) InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error {
    args := m.Called(tx, transactionID, paymentTypeID, amount)
    return args.Error(0)
}
```

Update existing tests to pass `nil` for customerRepo and paymentSvc, or update the `NewTransactionService` call.

Update TestCheckout_Success:
```go
func TestCheckout_Success(t *testing.T) {
    repo := new(mockTransactionRepo)
    customerRepo := new(mockCustomerRepoForTransaction)
    paymentSvc := new(mockPaymentSvc)
    svc := NewTransactionService(repo, customerRepo, paymentSvc)
    // ... existing test setup ...
}
```

---

## Self-Checklist

- [ ] No circular imports (customer doesn't import transaction, transaction imports customer interface)
- [ ] Existing checkout still works without customer_id or payments (backward compatible)
- [ ] Payment total validation works (sum of payments == total amount)
- [ ] All routes registered under /api/v1/
- [ ] Migrations are idempotent (IF NOT EXISTS, ON CONFLICT DO NOTHING)
