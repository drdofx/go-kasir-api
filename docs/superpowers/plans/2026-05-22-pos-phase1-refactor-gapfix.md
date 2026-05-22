# Phase 1: Refactor + Gap Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure codebase into domain packages under `internal/domain/`, add `/api/v1/*` routing with dual-run backward compatibility, and fix frontend gap endpoints (`GET /api/transactions`, `POST /api/auth/change-password`).

**Architecture:** Keep existing handler/service/repository pattern but move each domain into its own package under `internal/domain/`. Shared utilities go in `internal/pkg/`. Old routes at `/api/*` continue working alongside new `/api/v1/*` routes.

**Tech Stack:** Go 1.24, net/http (standard library), PostgreSQL, golang-migrate, zerolog, viper, golang-jwt

---

## File Structure

```
internal/
├── domain/
│   ├── auth/
│   │   ├── handler.go            Login, Logout, Me, ChangePassword handlers
│   │   ├── service.go            JWT, bcrypt, Login/ChangePassword logic
│   │   └── repository.go         User lookup by ID/username
│   ├── product/
│   │   ├── handler.go            CRUD handlers
│   │   ├── service.go            Validation + repo delegation
│   │   └── repository.go         DB operations
│   ├── category/
│   │   ├── handler.go            CRUD handlers
│   │   ├── service.go            Validation + repo delegation
│   │   └── repository.go         DB operations
│   ├── transaction/
│   │   ├── handler.go            Checkout, ListTransactions, GetTransactionByID
│   │   ├── service.go            Checkout logic, transaction listing
│   │   └── repository.go         DB operations + batch inserts
│   └── report/
│       ├── handler.go            Today report, date range report
│       └── service.go            Sales aggregation logic
├── pkg/
│   ├── database/
│   │   ├── database.go           InitDB (from internal/database/database.go)
│   │   └── migrate.go            RunMigrations (from internal/database/migrate.go)
│   ├── middleware/
│   │   ├── middleware.go          Chain, CORS, Logger, RequestID, SecurityHeaders
│   │   ├── middleware_test.go
│   │   └── session.go            JWTAuth, UserFromContext
│   ├── helpers/
│   │   ├── helpers.go            parseID, jsonResponse, jsonError
│   │   └── router.go            NewRouter, group routes, v1 route builder
│   └── testutil/
│       └── helpers.go            HTTP test helpers
├── handler/                      [REMOVED after migration]
├── middleware/                    [MOVED to pkg/middleware/]
├── model/                        [REMOVED — types move to respective domains]
├── repository/                   [REMOVED after migration]
├── service/                      [REMOVED after migration]
└── testutil/                     [MOVED to pkg/testutil/]

cmd/api/main.go                   Updated: register v1 routes + keep old routes

migrations/
├── 000005_add_transactions_list_index.up.sql
└── 000005_add_transactions_list_index.down.sql
```

### Type Definitions (per domain)

Each domain defines its own types. The old `internal/model/` package is removed.

```go
// internal/domain/auth/handler.go types
type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}
type LoginResponse struct {
    Message string       `json:"message"`
    Token   string       `json:"token"`
    User    AuthUser     `json:"user"`
}
type AuthUser struct {
    ID        int       `json:"id"`
    Username  string    `json:"username"`
    Name      string    `json:"name"`
    Role      string    `json:"role"`
    CreatedAt time.Time `json:"created_at"`
}
type ChangePasswordRequest struct {
    CurrentPassword string `json:"current_password"`
    NewPassword     string `json:"new_password"`
}
type MessageResponse struct {
    Message string `json:"message"`
}

// internal/domain/product/handler.go types
type ProductRequest struct {
    Name       string `json:"name"`
    Price      int    `json:"price"`
    Stock      int    `json:"stock"`
    CategoryID int    `json:"category_id"`
}
type ProductResponse struct {
    ID           int    `json:"id"`
    Name         string `json:"name"`
    Price        int    `json:"price"`
    Stock        int    `json:"stock"`
    CategoryID   *int   `json:"category_id"`
    CategoryName string `json:"category_name"`
}

// internal/domain/category/handler.go types
type CategoryRequest struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}
type CategoryResponse struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
}

// internal/domain/transaction/handler.go types
type CheckoutItem struct {
    ProductID int `json:"product_id"`
    Quantity  int `json:"quantity"`
}
type CheckoutRequest struct {
    Items []CheckoutItem `json:"items"`
}
type TransactionResponse struct {
    ID          int                    `json:"id"`
    TotalAmount int                    `json:"total_amount"`
    CreatedAt   string                 `json:"created_at"`
    Details     []TransactionDetail    `json:"details"`
}
type TransactionDetail struct {
    ID            int    `json:"id"`
    TransactionID int    `json:"transaction_id"`
    ProductID     int    `json:"product_id"`
    ProductName   string `json:"product_name"`
    Quantity      int    `json:"quantity"`
    Subtotal      int    `json:"subtotal"`
}

// internal/domain/report/handler.go types
type SalesSummary struct {
    TotalRevenue      int            `json:"total_revenue"`
    TotalTransactions int            `json:"total_transactions"`
    TopProduct        *TopProduct    `json:"top_product,omitempty"`
}
type TopProduct struct {
    Name    string `json:"name"`
    QtySold int    `json:"qty_sold"`
}
```

### Interfaces

```go
// internal/domain/auth/service.go
type UserRepository interface {
    FindByID(id int) (*User, error)
    FindByUsername(username string) (*User, error)
    UpdatePassword(userID int, newHash string) error
}
type User struct {
    ID           int
    Username     string
    PasswordHash string
    Name         string
    Role         string
    CreatedAt    time.Time
}

// internal/domain/product/service.go
type ProductRepository interface {
    FindAll(name string) ([]Product, error)
    FindByID(id int) (*Product, error)
    Create(p *Product) error
    Update(p *Product) error
    Delete(id int) error
}
type Product struct {
    ID           int
    Name         string
    Price        int
    Stock        int
    CategoryID   *int
    CategoryName string
}
type CategoryRepository interface {
    FindByID(id int) (*Category, error)
}
type Category struct {
    ID          int
    Name        string
    Description string
}

// internal/domain/category/service.go
type CategoryRepository interface {
    FindAll() ([]Category, error)
    FindByID(id int) (*Category, error)
    Create(c *Category) error
    Update(c *Category) error
    Delete(id int) error
}

// internal/domain/transaction/service.go
type TransactionRepository interface {
    BeginTx() (*sql.Tx, error)
    FindAll() ([]Transaction, error)
    FindByID(id int) (*Transaction, error)
    LockProducts(tx *sql.Tx, ids []int) ([]Product, error)
    UpdateStock(tx *sql.Tx, id, qty int) error
    InsertTransaction(tx *sql.Tx, total int) (int, error)
    InsertDetails(tx *sql.Tx, transactionID int, items []CheckoutItem, products []Product) error
}
type Transaction struct {
    ID          int
    TotalAmount int
    CreatedAt   time.Time
    Details     []TransactionDetailItem
}
type TransactionDetailItem struct {
    ID            int
    TransactionID int
    ProductID     int
    ProductName   string
    Quantity      int
    Subtotal      int
}
```

---

## Tasks

### Task 1: Create `internal/domain/auth/` package

**Files:**
- Create: `internal/domain/auth/repository.go`
- Create: `internal/domain/auth/service.go`
- Create: `internal/domain/auth/handler.go`
- Test: `internal/domain/auth/auth_test.go`

- [ ] **Step 1: Write `repository.go`**

```go
package auth

import (
    "database/sql"
)

type User struct {
    ID           int
    Username     string
    PasswordHash string
    Name         string
    Role         string
    CreatedAt    time.Time
}

type userRepository struct {
    db *sql.DB
}

type UserRepository interface {
    FindByID(id int) (*User, error)
    FindByUsername(username string) (*User, error)
    UpdatePassword(userID int, newHash string) error
}

func NewUserRepository(db *sql.DB) UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) FindByID(id int) (*User, error) {
    row := r.db.QueryRow("SELECT id, username, password_hash, name, role, created_at FROM users WHERE id = $1", id)
    u := &User{}
    err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return u, err
}

func (r *userRepository) FindByUsername(username string) (*User, error) {
    row := r.db.QueryRow("SELECT id, username, password_hash, name, role, created_at FROM users WHERE username = $1", username)
    u := &User{}
    err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return u, err
}

func (r *userRepository) UpdatePassword(userID int, newHash string) error {
    _, err := r.db.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", newHash, userID)
    return err
}
```

- [ ] **Step 2: Write `service.go`**

```go
package auth

import (
    "errors"
    "time"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

type AuthService struct {
    userRepo  UserRepository
    jwtSecret string
}

func NewAuthService(userRepo UserRepository, jwtSecret string) *AuthService {
    return &AuthService{userRepo: userRepo, jwtSecret: jwtSecret}
}

func (s *AuthService) Login(username, password string) (string, *User, error) {
    user, err := s.userRepo.FindByUsername(username)
    if err != nil {
        return "", nil, err
    }
    if user == nil {
        return "", nil, errors.New("invalid credentials")
    }
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
        return "", nil, errors.New("invalid credentials")
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id":  user.ID,
        "username": user.Username,
        "role":     user.Role,
        "exp":      time.Now().Add(24 * time.Hour).Unix(),
    })
    tokenStr, err := token.SignedString([]byte(s.jwtSecret))
    if err != nil {
        return "", nil, err
    }
    return tokenStr, user, nil
}

func (s *AuthService) ValidateToken(tokenStr string) (int, string, string, error) {
    token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
        return []byte(s.jwtSecret), nil
    })
    if err != nil || !token.Valid {
        return 0, "", "", errors.New("invalid token")
    }
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return 0, "", "", errors.New("invalid token claims")
    }
    userID := int(claims["user_id"].(float64))
    username := claims["username"].(string)
    role := claims["role"].(string)
    return userID, username, role, nil
}

func (s *AuthService) ChangePassword(userID int, currentPassword, newPassword string) error {
    user, err := s.userRepo.FindByID(userID)
    if err != nil {
        return err
    }
    if user == nil {
        return errors.New("user not found")
    }
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
        return errors.New("current password is incorrect")
    }
    hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    return s.userRepo.UpdatePassword(userID, string(hash))
}
```

- [ ] **Step 3: Write `handler.go`**

```go
package auth

import (
    "encoding/json"
    "errors"
    "net/http"
    "go-kasir-api/internal/pkg/helpers"
    "go-kasir-api/internal/pkg/middleware"
)

type AuthHandler struct {
    service *AuthService
}

func NewAuthHandler(service *AuthService) *AuthHandler {
    return &AuthHandler{service: service}
}

type loginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type loginResponse struct {
    Message string   `json:"message"`
    Token   string   `json:"token"`
    User    userJSON `json:"user"`
}

type userJSON struct {
    ID        int    `json:"id"`
    Username  string `json:"username"`
    Name      string `json:"name"`
    Role      string `json:"role"`
    CreatedAt string `json:"created_at"`
}

type changePasswordRequest struct {
    CurrentPassword string `json:"current_password"`
    NewPassword     string `json:"new_password"`
}

type messageResponse struct {
    Message string `json:"message"`
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
    var req loginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    if req.Username == "" || req.Password == "" {
        helpers.WriteError(w, http.StatusBadRequest, "username and password are required")
        return
    }
    token, user, err := h.service.Login(req.Username, req.Password)
    if err != nil {
        if err.Error() == "invalid credentials" {
            helpers.WriteError(w, http.StatusUnauthorized, "invalid credentials")
            return
        }
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    helpers.WriteJSON(w, http.StatusOK, loginResponse{
        Message: "login successful",
        Token:   token,
        User: userJSON{
            ID:        user.ID,
            Username:  user.Username,
            Name:      user.Name,
            Role:      user.Role,
            CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
        },
    })
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
    helpers.WriteJSON(w, http.StatusOK, messageResponse{Message: "logout successful"})
}

func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
    user := middleware.UserFromContext(r.Context())
    if user == nil {
        helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
        return
    }
    helpers.WriteJSON(w, http.StatusOK, userJSON{
        ID:        user.ID,
        Username:  user.Username,
        Name:      user.Name,
        Role:      user.Role,
        CreatedAt: user.CreatedAt,
    })
}

func (h *AuthHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
    user := middleware.UserFromContext(r.Context())
    if user == nil {
        helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
        return
    }
    var req changePasswordRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    if req.CurrentPassword == "" || req.NewPassword == "" {
        helpers.WriteError(w, http.StatusBadRequest, "current_password and new_password are required")
        return
    }
    if len(req.NewPassword) < 6 {
        helpers.WriteError(w, http.StatusBadRequest, "new password must be at least 6 characters")
        return
    }
    if err := h.service.ChangePassword(user.ID, req.CurrentPassword, req.NewPassword); err != nil {
        if err.Error() == "current password is incorrect" {
            helpers.WriteError(w, http.StatusBadRequest, "current password is incorrect")
            return
        }
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    helpers.WriteJSON(w, http.StatusOK, messageResponse{Message: "password changed successfully"})
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/domain/auth/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/domain/auth/
git commit -m "feat: add auth domain package with login, change-password"
```

---

### Task 2: Create `internal/domain/product/` package

**Files:**
- Create: `internal/domain/product/repository.go`
- Create: `internal/domain/product/service.go`
- Create: `internal/domain/product/handler.go`

- [ ] **Step 1: Write `repository.go`**

```go
package product

import (
    "database/sql"
    "fmt"
    "strings"
)

type Product struct {
    ID           int
    Name         string
    Price        int
    Stock        int
    CategoryID   *int
    CategoryName string
}

type productRepository struct {
    db *sql.DB
}

type ProductRepository interface {
    FindAll(name string) ([]Product, error)
    FindByID(id int) (*Product, error)
    Create(p *Product) error
    Update(p *Product) error
    Delete(id int) error
}

func NewProductRepository(db *sql.DB) ProductRepository {
    return &productRepository{db: db}
}

func (r *productRepository) FindAll(name string) ([]Product, error) {
    query := `SELECT p.id, p.name, p.price, p.stock, p.category_id, COALESCE(c.name, '') as category_name
              FROM products p LEFT JOIN categories c ON p.category_id = c.id`
    var args []interface{}
    if name != "" {
        query += " WHERE p.name ILIKE $1"
        args = append(args, "%"+name+"%")
    }
    query += " ORDER BY p.id"
    rows, err := r.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var products []Product
    for rows.Next() {
        var p Product
        if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &p.CategoryID, &p.CategoryName); err != nil {
            return nil, err
        }
        products = append(products, p)
    }
    return products, rows.Err()
}

func (r *productRepository) FindByID(id int) (*Product, error) {
    row := r.db.QueryRow(`SELECT p.id, p.name, p.price, p.stock, p.category_id, COALESCE(c.name, '')
        FROM products p LEFT JOIN categories c ON p.category_id = c.id WHERE p.id = $1`, id)
    p := &Product{}
    if err := row.Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &p.CategoryID, &p.CategoryName); err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    return p, nil
}

func (r *productRepository) Create(p *Product) error {
    return r.db.QueryRow(
        "INSERT INTO products (name, price, stock, category_id) VALUES ($1, $2, $3, $4) RETURNING id",
        p.Name, p.Price, p.Stock, p.CategoryID,
    ).Scan(&p.ID)
}

func (r *productRepository) Update(p *Product) error {
    _, err := r.db.Exec("UPDATE products SET name=$1, price=$2, stock=$3, category_id=$4 WHERE id=$5",
        p.Name, p.Price, p.Stock, p.CategoryID, p.ID)
    return err
}

func (r *productRepository) Delete(id int) error {
    _, err := r.db.Exec("DELETE FROM products WHERE id = $1", id)
    return err
}
```

- [ ] **Step 2: Write `service.go`**

```go
package product

import "errors"

type CategoryRepository interface {
    FindByID(id int) (*Category, error)
}

type Category struct {
    ID          int
    Name        string
    Description string
}

type ProductService struct {
    repo             ProductRepository
    categoryRepo     CategoryRepository
}

func NewProductService(repo ProductRepository, categoryRepo CategoryRepository) *ProductService {
    return &ProductService{repo: repo, categoryRepo: categoryRepo}
}

func (s *ProductService) FindAll(name string) ([]Product, error) {
    return s.repo.FindAll(name)
}

func (s *ProductService) FindByID(id int) (*Product, error) {
    return s.repo.FindByID(id)
}

func (s *ProductService) Create(p *Product) error {
    if p.Name == "" {
        return errors.New("product name is required")
    }
    if p.Price < 0 {
        return errors.New("price must be non-negative")
    }
    if p.Stock < 0 {
        return errors.New("stock must be non-negative")
    }
    if p.CategoryID != nil && *p.CategoryID > 0 {
        cat, err := s.categoryRepo.FindByID(*p.CategoryID)
        if err != nil {
            return err
        }
        if cat == nil {
            return errors.New("category not found")
        }
    }
    return s.repo.Create(p)
}

func (s *ProductService) Update(p *Product) error {
    if p.Name == "" {
        return errors.New("product name is required")
    }
    if p.Price < 0 {
        return errors.New("price must be non-negative")
    }
    if p.Stock < 0 {
        return errors.New("stock must be non-negative")
    }
    existing, err := s.repo.FindByID(p.ID)
    if err != nil {
        return err
    }
    if existing == nil {
        return errors.New("product not found")
    }
    if p.CategoryID != nil && *p.CategoryID > 0 {
        cat, err := s.categoryRepo.FindByID(*p.CategoryID)
        if err != nil {
            return err
        }
        if cat == nil {
            return errors.New("category not found")
        }
    }
    return s.repo.Update(p)
}

func (s *ProductService) Delete(id int) error {
    existing, err := s.repo.FindByID(id)
    if err != nil {
        return err
    }
    if existing == nil {
        return errors.New("product not found")
    }
    return s.repo.Delete(id)
}
```

- [ ] **Step 3: Write `handler.go`**

```go
package product

import (
    "encoding/json"
    "net/http"
    "strconv"
    "go-kasir-api/internal/pkg/helpers"
)

type ProductHandler struct {
    service *ProductService
}

func NewProductHandler(service *ProductService) *ProductHandler {
    return &ProductHandler{service: service}
}

type productRequest struct {
    Name       string `json:"name"`
    Price      int    `json:"price"`
    Stock      int    `json:"stock"`
    CategoryID *int   `json:"category_id"`
}

type productResponse struct {
    ID           int    `json:"id"`
    Name         string `json:"name"`
    Price        int    `json:"price"`
    Stock        int    `json:"stock"`
    CategoryID   *int   `json:"category_id"`
    CategoryName string `json:"category_name"`
}

func toResponse(p Product) productResponse {
    return productResponse{
        ID:           p.ID,
        Name:         p.Name,
        Price:        p.Price,
        Stock:        p.Stock,
        CategoryID:   p.CategoryID,
        CategoryName: p.CategoryName,
    }
}

func toResponses(products []Product) []productResponse {
    res := make([]productResponse, len(products))
    for i, p := range products {
        res[i] = toResponse(p)
    }
    return res
}

func (h *ProductHandler) HandleProducts(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        h.list(w, r)
    case http.MethodPost:
        h.create(w, r)
    default:
        helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
    }
}

func (h *ProductHandler) HandleProductByID(w http.ResponseWriter, r *http.Request) {
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

func (h *ProductHandler) list(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get("name")
    products, err := h.service.FindAll(name)
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    if products == nil {
        products = []Product{}
    }
    helpers.WriteJSON(w, http.StatusOK, toResponses(products))
}

func (h *ProductHandler) create(w http.ResponseWriter, r *http.Request) {
    var req productRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    p := &Product{Name: req.Name, Price: req.Price, Stock: req.Stock, CategoryID: req.CategoryID}
    if err := h.service.Create(p); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    helpers.WriteJSON(w, http.StatusCreated, toResponse(*p))
}

func (h *ProductHandler) getByID(w http.ResponseWriter, id int) {
    p, err := h.service.FindByID(id)
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    if p == nil {
        helpers.WriteError(w, http.StatusNotFound, "product not found")
        return
    }
    helpers.WriteJSON(w, http.StatusOK, toResponse(*p))
}

func (h *ProductHandler) update(w http.ResponseWriter, r *http.Request, id int) {
    var req productRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    p := &Product{ID: id, Name: req.Name, Price: req.Price, Stock: req.Stock, CategoryID: req.CategoryID}
    if err := h.service.Update(p); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    helpers.WriteJSON(w, http.StatusOK, toResponse(*p))
}

func (h *ProductHandler) delete(w http.ResponseWriter, id int) {
    if err := h.service.Delete(id); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/domain/product/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/domain/product/
git commit -m "feat: add product domain package"
```

---

### Task 3: Create `internal/domain/category/` package

**Files:**
- Create: `internal/domain/category/repository.go`
- Create: `internal/domain/category/service.go`
- Create: `internal/domain/category/handler.go`

- [ ] **Step 1: Write `repository.go`**

```go
package category

import "database/sql"

type Category struct {
    ID          int
    Name        string
    Description string
}

type categoryRepository struct {
    db *sql.DB
}

type CategoryRepository interface {
    FindAll() ([]Category, error)
    FindByID(id int) (*Category, error)
    Create(c *Category) error
    Update(c *Category) error
    Delete(id int) error
}

func NewCategoryRepository(db *sql.DB) CategoryRepository {
    return &categoryRepository{db: db}
}

func (r *categoryRepository) FindAll() ([]Category, error) {
    rows, err := r.db.Query("SELECT id, name, description FROM categories ORDER BY id")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var categories []Category
    for rows.Next() {
        var c Category
        if err := rows.Scan(&c.ID, &c.Name, &c.Description); err != nil {
            return nil, err
        }
        categories = append(categories, c)
    }
    return categories, rows.Err()
}

func (r *categoryRepository) FindByID(id int) (*Category, error) {
    row := r.db.QueryRow("SELECT id, name, description FROM categories WHERE id = $1", id)
    c := &Category{}
    if err := row.Scan(&c.ID, &c.Name, &c.Description); err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    return c, nil
}

func (r *categoryRepository) Create(c *Category) error {
    return r.db.QueryRow("INSERT INTO categories (name, description) VALUES ($1, $2) RETURNING id",
        c.Name, c.Description).Scan(&c.ID)
}

func (r *categoryRepository) Update(c *Category) error {
    _, err := r.db.Exec("UPDATE categories SET name=$1, description=$2 WHERE id=$3",
        c.Name, c.Description, c.ID)
    return err
}

func (r *categoryRepository) Delete(id int) error {
    _, err := r.db.Exec("DELETE FROM categories WHERE id = $1", id)
    return err
}
```

- [ ] **Step 2: Write `service.go`**

```go
package category

import "errors"

type CategoryService struct {
    repo CategoryRepository
}

func NewCategoryService(repo CategoryRepository) *CategoryService {
    return &CategoryService{repo: repo}
}

func (s *CategoryService) FindAll() ([]Category, error) {
    return s.repo.FindAll()
}

func (s *CategoryService) FindByID(id int) (*Category, error) {
    return s.repo.FindByID(id)
}

func (s *CategoryService) Create(c *Category) error {
    if c.Name == "" {
        return errors.New("category name is required")
    }
    if c.Description == "" {
        return errors.New("category description is required")
    }
    return s.repo.Create(c)
}

func (s *CategoryService) Update(c *Category) error {
    if c.Name == "" {
        return errors.New("category name is required")
    }
    if c.Description == "" {
        return errors.New("category description is required")
    }
    existing, err := s.repo.FindByID(c.ID)
    if err != nil {
        return err
    }
    if existing == nil {
        return errors.New("category not found")
    }
    return s.repo.Update(c)
}

func (s *CategoryService) Delete(id int) error {
    existing, err := s.repo.FindByID(id)
    if err != nil {
        return err
    }
    if existing == nil {
        return errors.New("category not found")
    }
    return s.repo.Delete(id)
}
```

- [ ] **Step 3: Write `handler.go`**

```go
package category

import (
    "encoding/json"
    "net/http"
    "strconv"
    "go-kasir-api/internal/pkg/helpers"
)

type CategoryHandler struct {
    service *CategoryService
}

func NewCategoryHandler(service *CategoryService) *CategoryHandler {
    return &CategoryHandler{service: service}
}

type categoryRequest struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}

type categoryResponse struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
}

func toResponse(c Category) categoryResponse {
    return categoryResponse{ID: c.ID, Name: c.Name, Description: c.Description}
}

func toResponses(categories []Category) []categoryResponse {
    res := make([]categoryResponse, len(categories))
    for i, c := range categories {
        res[i] = toResponse(c)
    }
    return res
}

func (h *CategoryHandler) HandleCategories(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        h.list(w, r)
    case http.MethodPost:
        h.create(w, r)
    default:
        helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
    }
}

func (h *CategoryHandler) HandleCategoryByID(w http.ResponseWriter, r *http.Request) {
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

func (h *CategoryHandler) list(w http.ResponseWriter, r *http.Request) {
    categories, err := h.service.FindAll()
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    if categories == nil {
        categories = []Category{}
    }
    helpers.WriteJSON(w, http.StatusOK, toResponses(categories))
}

func (h *CategoryHandler) create(w http.ResponseWriter, r *http.Request) {
    var req categoryRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    c := &Category{Name: req.Name, Description: req.Description}
    if err := h.service.Create(c); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    helpers.WriteJSON(w, http.StatusCreated, toResponse(*c))
}

func (h *CategoryHandler) getByID(w http.ResponseWriter, id int) {
    c, err := h.service.FindByID(id)
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    if c == nil {
        helpers.WriteError(w, http.StatusNotFound, "category not found")
        return
    }
    helpers.WriteJSON(w, http.StatusOK, toResponse(*c))
}

func (h *CategoryHandler) update(w http.ResponseWriter, r *http.Request, id int) {
    var req categoryRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    c := &Category{ID: id, Name: req.Name, Description: req.Description}
    if err := h.service.Update(c); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    helpers.WriteJSON(w, http.StatusOK, toResponse(*c))
}

func (h *CategoryHandler) delete(w http.ResponseWriter, id int) {
    if err := h.service.Delete(id); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/domain/category/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/domain/category/
git commit -m "feat: add category domain package"
```

---

### Task 4: Create `internal/domain/transaction/` package

**Files:**
- Create: `internal/domain/transaction/repository.go`
- Create: `internal/domain/transaction/service.go`
- Create: `internal/domain/transaction/handler.go`

- [ ] **Step 1: Write `repository.go`**

```go
package transaction

import (
    "database/sql"
    "fmt"
    "strings"
)

type Transaction struct {
    ID          int
    TotalAmount int
    CreatedAt   string
    Details     []DetailItem
}

type DetailItem struct {
    ID            int
    TransactionID int
    ProductID     int
    ProductName   string
    Quantity      int
    Subtotal      int
}

type transactionRepository struct {
    db *sql.DB
}

type TransactionRepository interface {
    BeginTx() (*sql.Tx, error)
    FindAll() ([]Transaction, error)
    FindByID(id int) (*Transaction, error)
    LockProducts(tx *sql.Tx, ids []int) ([]LockedProduct, error)
    UpdateStock(tx *sql.Tx, id, qty int) error
    InsertTransaction(tx *sql.Tx, total int) (int, error)
    InsertDetails(tx *sql.Tx, transactionID int, items []CheckoutItem, products []LockedProduct) error
}

type LockedProduct struct {
    ID    int
    Name  string
    Price int
    Stock int
}

func NewTransactionRepository(db *sql.DB) TransactionRepository {
    return &transactionRepository{db: db}
}

func (r *transactionRepository) BeginTx() (*sql.Tx, error) {
    return r.db.Begin()
}

func (r *transactionRepository) FindAll() ([]Transaction, error) {
    rows, err := r.db.Query("SELECT id, total_amount, created_at FROM transactions ORDER BY created_at DESC")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var txns []Transaction
    for rows.Next() {
        var t Transaction
        var createdAt interface{}
        if err := rows.Scan(&t.ID, &t.TotalAmount, &createdAt); err != nil {
            return nil, err
        }
        t.CreatedAt = fmt.Sprintf("%v", createdAt)
        txns = append(txns, t)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    for i := range txns {
        details, err := r.getDetails(txns[i].ID)
        if err != nil {
            return nil, err
        }
        txns[i].Details = details
    }
    return txns, nil
}

func (r *transactionRepository) FindByID(id int) (*Transaction, error) {
    row := r.db.QueryRow("SELECT id, total_amount, created_at FROM transactions WHERE id = $1", id)
    t := &Transaction{}
    var createdAt interface{}
    if err := row.Scan(&t.ID, &t.TotalAmount, &createdAt); err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    t.CreatedAt = fmt.Sprintf("%v", createdAt)
    details, err := r.getDetails(t.ID)
    if err != nil {
        return nil, err
    }
    t.Details = details
    return t, nil
}

func (r *transactionRepository) getDetails(transactionID int) ([]DetailItem, error) {
    rows, err := r.db.Query(`SELECT td.id, td.transaction_id, td.product_id, COALESCE(p.name, ''), td.quantity, td.subtotal
        FROM transaction_details td LEFT JOIN products p ON td.product_id = p.id
        WHERE td.transaction_id = $1 ORDER BY td.id`, transactionID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var details []DetailItem
    for rows.Next() {
        var d DetailItem
        if err := rows.Scan(&d.ID, &d.TransactionID, &d.ProductID, &d.ProductName, &d.Quantity, &d.Subtotal); err != nil {
            return nil, err
        }
        details = append(details, d)
    }
    return details, rows.Err()
}

func (r *transactionRepository) LockProducts(tx *sql.Tx, ids []int) ([]LockedProduct, error) {
    placeholders := make([]string, len(ids))
    args := make([]interface{}, len(ids))
    for i, id := range ids {
        placeholders[i] = fmt.Sprintf("$%d", i+1)
        args[i] = id
    }
    query := fmt.Sprintf("SELECT id, name, price, stock FROM products WHERE id IN (%s) ORDER BY id", strings.Join(placeholders, ","))
    rows, err := tx.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var products []LockedProduct
    for rows.Next() {
        var p LockedProduct
        if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock); err != nil {
            return nil, err
        }
        products = append(products, p)
    }
    return products, rows.Err()
}

func (r *transactionRepository) UpdateStock(tx *sql.Tx, id, qty int) error {
    _, err := tx.Exec("UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1", qty, id)
    return err
}

func (r *transactionRepository) InsertTransaction(tx *sql.Tx, total int) (int, error) {
    var id int
    err := tx.QueryRow("INSERT INTO transactions (total_amount) VALUES ($1) RETURNING id", total).Scan(&id)
    return id, err
}

func (r *transactionRepository) InsertDetails(tx *sql.Tx, transactionID int, items []CheckoutItem, products []LockedProduct) error {
    if len(items) == 0 {
        return nil
    }
    productMap := make(map[int]LockedProduct)
    for _, p := range products {
        productMap[p.ID] = p
    }
    var valueStrings []string
    var args []interface{}
    argIdx := 1
    for _, item := range items {
        p, ok := productMap[item.ProductID]
        if !ok {
            continue
        }
        subtotal := p.Price * item.Quantity
        valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d)", argIdx, argIdx+1, argIdx+2, argIdx+3))
        args = append(args, transactionID, item.ProductID, item.Quantity, subtotal)
        argIdx += 4
    }
    query := fmt.Sprintf("INSERT INTO transaction_details (transaction_id, product_id, quantity, subtotal) VALUES %s",
        strings.Join(valueStrings, ","))
    _, err := tx.Exec(query, args...)
    return err
}
```

- [ ] **Step 2: Write `service.go`**

```go
package transaction

import (
    "errors"
    "fmt"
)

type CheckoutItem struct {
    ProductID int `json:"product_id"`
    Quantity  int `json:"quantity"`
}

type CheckoutRequest struct {
    Items []CheckoutItem `json:"items"`
}

const maxItemsPerCheckout = 100

type TransactionService struct {
    repo TransactionRepository
}

func NewTransactionService(repo TransactionRepository) *TransactionService {
    return &TransactionService{repo: repo}
}

func (s *TransactionService) Checkout(req CheckoutRequest) (*Transaction, error) {
    if len(req.Items) == 0 {
        return nil, errors.New("checkout requires at least one item")
    }
    if len(req.Items) > maxItemsPerCheckout {
        return nil, fmt.Errorf("maximum %d items per checkout", maxItemsPerCheckout)
    }
    for _, item := range req.Items {
        if item.Quantity <= 0 {
            return nil, fmt.Errorf("invalid quantity for product %d", item.ProductID)
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
            return nil, fmt.Errorf("product %d not found", item.ProductID)
        }
        if p.Stock < item.Quantity {
            return nil, fmt.Errorf("insufficient stock for product %s (available: %d, requested: %d)",
                p.Name, p.Stock, item.Quantity)
        }
    }
    var totalAmount int
    for _, item := range req.Items {
        p := productMap[item.ProductID]
        subtotal := p.Price * item.Quantity
        if totalAmount > 1<<31-1-subtotal {
            return nil, errors.New("total amount overflow")
        }
        totalAmount += subtotal
        if err := s.repo.UpdateStock(tx, item.ProductID, item.Quantity); err != nil {
            return nil, fmt.Errorf("failed to update stock for product %d: %w", item.ProductID, err)
        }
    }
    transactionID, err := s.repo.InsertTransaction(tx, totalAmount)
    if err != nil {
        return nil, fmt.Errorf("failed to create transaction: %w", err)
    }
    if err := s.repo.InsertDetails(tx, transactionID, req.Items, products); err != nil {
        return nil, fmt.Errorf("failed to insert transaction details: %w", err)
    }
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }
    return s.repo.FindByID(transactionID)
}

func (s *TransactionService) FindAll() ([]Transaction, error) {
    return s.repo.FindAll()
}

func (s *TransactionService) FindByID(id int) (*Transaction, error) {
    return s.repo.FindByID(id)
}
```

- [ ] **Step 3: Write `handler.go`**

```go
package transaction

import (
    "encoding/json"
    "net/http"
    "strconv"
    "go-kasir-api/internal/pkg/helpers"
)

type TransactionHandler struct {
    service *TransactionService
}

func NewTransactionHandler(service *TransactionService) *TransactionHandler {
    return &TransactionHandler{service: service}
}

type checkoutRequest struct {
    Items []CheckoutItem `json:"items"`
}

type transactionResponse struct {
    ID          int                `json:"id"`
    TotalAmount int                `json:"total_amount"`
    CreatedAt   string             `json:"created_at"`
    Details     []detailResponse   `json:"details"`
}

type detailResponse struct {
    ID            int    `json:"id"`
    TransactionID int    `json:"transaction_id"`
    ProductID     int    `json:"product_id"`
    ProductName   string `json:"product_name"`
    Quantity      int    `json:"quantity"`
    Subtotal      int    `json:"subtotal"`
}

func toTransactionResponse(t Transaction) transactionResponse {
    details := make([]detailResponse, len(t.Details))
    for i, d := range t.Details {
        details[i] = detailResponse{
            ID:            d.ID,
            TransactionID: d.TransactionID,
            ProductID:     d.ProductID,
            ProductName:   d.ProductName,
            Quantity:      d.Quantity,
            Subtotal:      d.Subtotal,
        }
    }
    return transactionResponse{
        ID:          t.ID,
        TotalAmount: t.TotalAmount,
        CreatedAt:   t.CreatedAt,
        Details:     details,
    }
}

func toTransactionResponses(txns []Transaction) []transactionResponse {
    res := make([]transactionResponse, len(txns))
    for i, t := range txns {
        res[i] = toTransactionResponse(t)
    }
    return res
}

func (h *TransactionHandler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
    var req checkoutRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    checkoutReq := CheckoutRequest{Items: req.Items}
    txn, err := h.service.Checkout(checkoutReq)
    if err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    helpers.WriteJSON(w, http.StatusCreated, toTransactionResponse(*txn))
}

func (h *TransactionHandler) HandleTransactions(w http.ResponseWriter, r *http.Request) {
    txns, err := h.service.FindAll()
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    if txns == nil {
        txns = []Transaction{}
    }
    helpers.WriteJSON(w, http.StatusOK, toTransactionResponses(txns))
}

func (h *TransactionHandler) HandleTransactionByID(w http.ResponseWriter, r *http.Request) {
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
    txn, err := h.service.FindByID(id)
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    if txn == nil {
        helpers.WriteError(w, http.StatusNotFound, "transaction not found")
        return
    }
    helpers.WriteJSON(w, http.StatusOK, toTransactionResponse(*txn))
}
```

- [ ] **Step 4: Write `handler.go` checklist item for new endpoint

Wait, I already included HandleTransactions and HandleTransactionByID above. The `GET /api/v1/transactions` endpoint maps to HandleTransactions.

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/domain/transaction/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/domain/transaction/
git commit -m "feat: add transaction domain package with list endpoint"
```

---

### Task 5: Create `internal/domain/report/` package

**Files:**
- Create: `internal/domain/report/handler.go`
- Create: `internal/domain/report/service.go`

- [ ] **Step 1: Write `service.go`**

```go
package report

import (
    "database/sql"
    "time"
)

type SalesSummary struct {
    TotalRevenue      int         `json:"total_revenue"`
    TotalTransactions int         `json:"total_transactions"`
    TopProduct        *TopProduct `json:"top_product,omitempty"`
}

type TopProduct struct {
    Name    string `json:"name"`
    QtySold int    `json:"qty_sold"`
}

type ReportService struct {
    db *sql.DB
}

func NewReportService(db *sql.DB) *ReportService {
    return &ReportService{db: db}
}

func (s *ReportService) GetSalesSummary(startDate, endDate string) (*SalesSummary, error) {
    summary := &SalesSummary{}
    err := s.db.QueryRow(`
        SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
        FROM transactions WHERE DATE(created_at) >= $1 AND DATE(created_at) <= $2`,
        startDate, endDate).Scan(&summary.TotalRevenue, &summary.TotalTransactions)
    if err != nil {
        return nil, err
    }
    top := &TopProduct{}
    err = s.db.QueryRow(`
        SELECT COALESCE(p.name, ''), COALESCE(SUM(td.quantity), 0)
        FROM transaction_details td
        JOIN products p ON td.product_id = p.id
        JOIN transactions t ON td.transaction_id = t.id
        WHERE DATE(t.created_at) >= $1 AND DATE(t.created_at) <= $2
        GROUP BY p.name
        ORDER BY SUM(td.quantity) DESC LIMIT 1`,
        startDate, endDate).Scan(&top.Name, &top.QtySold)
    if err == sql.ErrNoRows || top.Name == "" {
        summary.TopProduct = nil
    } else {
        summary.TopProduct = top
    }
    return summary, nil
}
```

- [ ] **Step 2: Write `handler.go`**

```go
package report

import (
    "net/http"
    "time"
    "go-kasir-api/internal/pkg/helpers"
)

type ReportHandler struct {
    service *ReportService
}

func NewReportHandler(service *ReportService) *ReportHandler {
    return &ReportHandler{service: service}
}

func (h *ReportHandler) HandleTodayReport(w http.ResponseWriter, r *http.Request) {
    today := time.Now().Format("2006-01-02")
    summary, err := h.service.GetSalesSummary(today, today)
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    helpers.WriteJSON(w, http.StatusOK, summary)
}

func (h *ReportHandler) HandleReport(w http.ResponseWriter, r *http.Request) {
    startDate := r.URL.Query().Get("start_date")
    endDate := r.URL.Query().Get("end_date")
    if startDate == "" || endDate == "" {
        helpers.WriteError(w, http.StatusBadRequest, "start_date and end_date are required")
        return
    }
    summary, err := h.service.GetSalesSummary(startDate, endDate)
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    helpers.WriteJSON(w, http.StatusOK, summary)
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/domain/report/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/domain/report/
git commit -m "feat: add report domain package"
```

---

### Task 6: Create `internal/pkg/` shared packages

**Files:**
- Create: `internal/pkg/helpers/helpers.go`
- Create: `internal/pkg/helpers/router.go`
- Move: `internal/database/database.go` → `internal/pkg/database/database.go`
- Move: `internal/database/migrate.go` → `internal/pkg/database/migrate.go`
- Move: `internal/middleware/middleware.go` → `internal/pkg/middleware/middleware.go`
- Move: `internal/middleware/middleware_test.go` → `internal/pkg/middleware/middleware_test.go`
- Move: `internal/middleware/request_id_context.go` → `internal/pkg/middleware/request_id_context.go`
- Move: `internal/middleware/session.go` → `internal/pkg/middleware/session.go`
- Move: `internal/testutil/helpers.go` → `internal/pkg/testutil/helpers.go`

- [ ] **Step 1: Create `internal/pkg/helpers/helpers.go`**

```go
package helpers

import (
    "encoding/json"
    "net/http"
    "strconv"
)

func ParseID(idStr string) (int, error) {
    id, err := strconv.Atoi(idStr)
    if err != nil || id <= 0 {
        return 0, err
    }
    return id, nil
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(data); err != nil {
        http.Error(w, "internal server error", http.StatusInternalServerError)
    }
}

type errorResponse struct {
    Message string `json:"message"`
    Code    int    `json:"code"`
}

func WriteError(w http.ResponseWriter, status int, message string) {
    WriteJSON(w, status, errorResponse{Message: message, Code: status})
}
```

- [ ] **Step 2: Create `internal/pkg/helpers/router.go`**

```go
package helpers

import "net/http"

type Route struct {
    Method  string
    Pattern string
    Handler http.HandlerFunc
}

func RegisterRoutes(mux *http.ServeMux, prefix string, routes []Route) {
    for _, route := range routes {
        fullPath := prefix + route.Pattern
        mux.HandleFunc(fullPath, route.Handler)
    }
}
```

- [ ] **Step 3: Move database package**

Copy `internal/database/database.go` to `internal/pkg/database/database.go` and `internal/database/migrate.go` to `internal/pkg/database/migrate.go`. Update the package declaration to `package database`.

- [ ] **Step 4: Move middleware package**

Copy all files from `internal/middleware/` to `internal/pkg/middleware/`. Update imports within the files — any reference to `go-kasir-api/internal/middleware` changes to `go-kasir-api/internal/pkg/middleware`.

- [ ] **Step 5: Move testutil**

Copy `internal/testutil/helpers.go` to `internal/pkg/testutil/helpers.go`.

- [ ] **Step 6: Verify compilation**

Run: `go build ./internal/pkg/...`
Expected: no errors

- [ ] **Step 7: Commit**

```bash
git add internal/pkg/
git commit -m "refactor: create shared pkg packages (helpers, database, middleware, testutil)"
```

---

### Task 7: Update `cmd/api/main.go` — new router with v1 + old routes

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Rewrite `main.go`**

```go
package main

import (
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "go-kasir-api/internal/domain/auth"
    "go-kasir-api/internal/domain/category"
    "go-kasir-api/internal/domain/product"
    "go-kasir-api/internal/domain/report"
    "go-kasir-api/internal/domain/transaction"
    "go-kasir-api/internal/pkg/database"
    "go-kasir-api/internal/pkg/helpers"
    "go-kasir-api/internal/pkg/middleware"

    "github.com/rs/zerolog"
    "github.com/spf13/viper"
)

func main() {
    viper.SetDefault("PORT", "8080")
    viper.SetDefault("CORS_ALLOWED_ORIGIN", "http://localhost:8080")
    viper.SetDefault("LOG_LEVEL", "info")
    viper.SetDefault("MIGRATIONS_PATH", "migrations")
    viper.AutomaticEnv()
    viper.SetConfigFile(".env")
    viper.ReadInConfig()

    logLevel, err := zerolog.ParseLevel(viper.GetString("LOG_LEVEL"))
    if err != nil {
        logLevel = zerolog.InfoLevel
    }
    zerolog.SetGlobalLevel(logLevel)

    db := database.InitDB(viper.GetString("DB_CONN"))
    defer db.Close()

    if err := database.RunMigrations(db, viper.GetString("MIGRATIONS_PATH")); err != nil {
        log.Fatalf("failed to run migrations: %v", err)
    }

    // Repositories
    userRepo := auth.NewUserRepository(db)
    productRepo := product.NewProductRepository(db)
    categoryRepo := category.NewCategoryRepository(db)
    transactionRepo := transaction.NewTransactionRepository(db)

    // CategoryRepository interface for product service
    catRepoForProduct := &categoryRepoWrapper{categoryRepo}

    // Services
    authService := auth.NewAuthService(userRepo, viper.GetString("JWT_SECRET"))
    categoryService := category.NewCategoryService(categoryRepo)
    productService := product.NewProductService(productRepo, catRepoForProduct)
    transactionService := transaction.NewTransactionService(transactionRepo)
    reportService := report.NewReportService(db)

    // Handlers
    authHandler := auth.NewAuthHandler(authService)
    productHandler := product.NewProductHandler(productService)
    categoryHandler := category.NewCategoryHandler(categoryService)
    transactionHandler := transaction.NewTransactionHandler(transactionService)
    reportHandler := report.NewReportHandler(reportService)

    // Middleware
    corsMiddleware := middleware.CORS(viper.GetString("CORS_ALLOWED_ORIGIN"))
    jwtMiddleware := middleware.JWTAuth(authService)

    mux := http.NewServeMux()

    // Old routes (backward compatibility)
    mux.HandleFunc("/health", handlerHealth)
    mux.HandleFunc("/", handlerRoot)

    // Old API routes — delegate to v1 handlers for backward compatibility
    mux.HandleFunc("/api/auth/login", authHandler.HandleLogin)
    mux.HandleFunc("/api/auth/logout", authHandler.HandleLogout)
    mux.HandleFunc("/api/auth/me", authHandler.HandleMe)
    mux.HandleFunc("/api/auth/change-password", middleware.Chain(authHandler.HandleChangePassword, jwtMiddleware))
    mux.HandleFunc("/api/products", middleware.Chain(productHandler.HandleProducts, jwtMiddleware))
    mux.HandleFunc("/api/products/", middleware.Chain(productHandler.HandleProductByID, jwtMiddleware))
    mux.HandleFunc("/api/categories", middleware.Chain(categoryHandler.HandleCategories, jwtMiddleware))
    mux.HandleFunc("/api/categories/", middleware.Chain(categoryHandler.HandleCategoryByID, jwtMiddleware))
    mux.HandleFunc("/api/checkout", middleware.Chain(transactionHandler.HandleCheckout, jwtMiddleware))
    mux.HandleFunc("/api/transactions", middleware.Chain(transactionHandler.HandleTransactions, jwtMiddleware))
    mux.HandleFunc("/api/report/hari-ini", middleware.Chain(reportHandler.HandleTodayReport, jwtMiddleware))
    mux.HandleFunc("/api/report", middleware.Chain(reportHandler.HandleReport, jwtMiddleware))

    // V1 routes
    mux.HandleFunc("/api/v1/auth/login", authHandler.HandleLogin)
    mux.HandleFunc("/api/v1/auth/logout", authHandler.HandleLogout)
    mux.HandleFunc("/api/v1/auth/me", authHandler.HandleMe)
    mux.HandleFunc("/api/v1/auth/change-password", middleware.Chain(authHandler.HandleChangePassword, jwtMiddleware))
    mux.HandleFunc("/api/v1/products", middleware.Chain(productHandler.HandleProducts, jwtMiddleware))
    mux.HandleFunc("/api/v1/products/", middleware.Chain(productHandler.HandleProductByID, jwtMiddleware))
    mux.HandleFunc("/api/v1/categories", middleware.Chain(categoryHandler.HandleCategories, jwtMiddleware))
    mux.HandleFunc("/api/v1/categories/", middleware.Chain(categoryHandler.HandleCategoryByID, jwtMiddleware))
    mux.HandleFunc("/api/v1/checkout", middleware.Chain(transactionHandler.HandleCheckout, jwtMiddleware))
    mux.HandleFunc("/api/v1/transactions", middleware.Chain(transactionHandler.HandleTransactions, jwtMiddleware))
    mux.HandleFunc("/api/v1/transactions/", middleware.Chain(transactionHandler.HandleTransactionByID, jwtMiddleware))
    mux.HandleFunc("/api/v1/report/hari-ini", middleware.Chain(reportHandler.HandleTodayReport, jwtMiddleware))
    mux.HandleFunc("/api/v1/report", middleware.Chain(reportHandler.HandleReport, jwtMiddleware))

    wrapped := middleware.Chain(mux.ServeHTTP,
        middleware.RequestID,
        corsMiddleware,
        middleware.SecurityHeaders,
        middleware.Logger,
    )

    server := &http.Server{
        Addr:    ":" + viper.GetString("PORT"),
        Handler: wrapped,
    }

    go func() {
        log.Printf("server starting on port %s", viper.GetString("PORT"))
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("shutting down server...")
    server.Close()
}

type categoryRepoWrapper struct {
    repo category.CategoryRepository
}

func (w *categoryRepoWrapper) FindByID(id int) (*product.Category, error) {
    c, err := w.repo.FindByID(id)
    if err != nil {
        return nil, err
    }
    if c == nil {
        return nil, nil
    }
    return &product.Category{ID: c.ID, Name: c.Name, Description: c.Description}, nil
}

func handlerHealth(w http.ResponseWriter, r *http.Request) {
    helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "OK"})
}

func handlerRoot(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/health", http.StatusFound)
}
```

Note: Update the `middleware.UserFromContext` to return the correct type. The middleware package now lives in `internal/pkg/middleware/session.go`. Update `UserFromContext` to return:

```go
type ContextUser struct {
    ID       int
    Username string
    Name     string
    Role     string
    CreatedAt string
}
```

And update `internal/pkg/middleware/session.go` to store/return `*ContextUser`.

- [ ] **Step 2: Update middleware/session.go UserFromContext**

The `JWTAuth` middleware in `internal/pkg/middleware/session.go` needs updating to use `auth.AuthService` for token validation. It currently references the old auth service. Update the import and the `ValidateToken` call.

- [ ] **Step 3: Verify compilation**

Run: `go build ./cmd/api/`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add cmd/api/ internal/pkg/middleware/
git commit -m "refactor: update main.go with v1 routes and backward compatibility"
```

---

### Task 8: Clean up old structure

**Files:**
- Delete: `internal/handler/`
- Delete: `internal/model/`
- Delete: `internal/repository/`
- Delete: `internal/service/`
- Delete: `internal/middleware/` (already moved)
- Delete: `internal/database/` (already moved)
- Delete: `internal/testutil/` (already moved)

- [ ] **Step 1: Remove old directories**

Run: `rm -rf internal/handler internal/model internal/repository internal/service internal/middleware internal/database internal/testutil`

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Run existing tests**

Run: `go test ./...`
Expected: all existing category and transaction tests pass (after updating their imports)

Note: The old test files were in `internal/service/category_service_test.go` and `internal/service/transaction_service_test.go`. Those have been cleaned up. We'll need new tests for the new domain packages, but that's covered in subsequent tasks.

- [ ] **Step 4: Commit**

```bash
git rm -r internal/handler internal/model internal/repository internal/service internal/middleware internal/database internal/testutil
git commit -m "refactor: remove old flat structure after migration to domain packages"
```

---

### Task 9: Write tests for new domain packages

**Files:**
- Create: `internal/domain/category/category_service_test.go`
- Create: `internal/domain/transaction/transaction_service_test.go`

- [ ] **Step 1: Write category service tests**

```go
package category

import (
    "errors"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type mockCategoryRepo struct {
    mock.Mock
}

func (m *mockCategoryRepo) FindAll() ([]Category, error) {
    args := m.Called()
    return args.Get(0).([]Category), args.Error(1)
}

func (m *mockCategoryRepo) FindByID(id int) (*Category, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Category), args.Error(1)
}

func (m *mockCategoryRepo) Create(c *Category) error {
    args := m.Called(c)
    return args.Error(0)
}

func (m *mockCategoryRepo) Update(c *Category) error {
    args := m.Called(c)
    return args.Error(0)
}

func (m *mockCategoryRepo) Delete(id int) error {
    args := m.Called(id)
    return args.Error(0)
}

func TestCategoryService_GetAll(t *testing.T) {
    repo := new(mockCategoryRepo)
    svc := NewCategoryService(repo)
    expected := []Category{{ID: 1, Name: "Minuman", Description: "Minuman"}}
    repo.On("FindAll").Return(expected, nil)
    result, err := svc.FindAll()
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
    repo.AssertExpectations(t)
}

func TestCategoryService_Create_Valid(t *testing.T) {
    repo := new(mockCategoryRepo)
    svc := NewCategoryService(repo)
    c := &Category{Name: "Makanan", Description: "Makanan ringan"}
    repo.On("Create", c).Return(nil)
    err := svc.Create(c)
    assert.NoError(t, err)
    repo.AssertExpectations(t)
}

func TestCategoryService_Create_Invalid(t *testing.T) {
    repo := new(mockCategoryRepo)
    svc := NewCategoryService(repo)
    err := svc.Create(&Category{Name: "", Description: ""})
    assert.Error(t, err)
    repo.AssertNotCalled(t, "Create")
}

func TestCategoryService_GetByID_NotFound(t *testing.T) {
    repo := new(mockCategoryRepo)
    svc := NewCategoryService(repo)
    repo.On("FindByID", 999).Return(nil, nil)
    result, err := svc.FindByID(999)
    assert.NoError(t, err)
    assert.Nil(t, result)
}

func TestCategoryService_Delete(t *testing.T) {
    repo := new(mockCategoryRepo)
    svc := NewCategoryService(repo)
    repo.On("FindByID", 1).Return(&Category{ID: 1}, nil)
    repo.On("Delete", 1).Return(nil)
    err := svc.Delete(1)
    assert.NoError(t, err)
    repo.AssertExpectations(t)
}
```

- [ ] **Step 2: Run category tests**

Run: `go test ./internal/domain/category/... -v`
Expected: all tests PASS

- [ ] **Step 3: Write transaction service tests**

```go
package transaction

import (
    "database/sql"
    "errors"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type mockTransactionRepo struct {
    mock.Mock
}

func (m *mockTransactionRepo) BeginTx() (*sql.Tx, error) {
    args := m.Called()
    return args.Get(0).(*sql.Tx), args.Error(1)
}

func (m *mockTransactionRepo) FindAll() ([]Transaction, error) {
    args := m.Called()
    return args.Get(0).([]Transaction), args.Error(1)
}

func (m *mockTransactionRepo) FindByID(id int) (*Transaction, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Transaction), args.Error(1)
}

func (m *mockTransactionRepo) LockProducts(tx *sql.Tx, ids []int) ([]LockedProduct, error) {
    args := m.Called(tx, ids)
    return args.Get(0).([]LockedProduct), args.Error(1)
}

func (m *mockTransactionRepo) UpdateStock(tx *sql.Tx, id, qty int) error {
    args := m.Called(tx, id, qty)
    return args.Error(0)
}

func (m *mockTransactionRepo) InsertTransaction(tx *sql.Tx, total int) (int, error) {
    args := m.Called(tx, total)
    return args.Int(0), args.Error(1)
}

func (m *mockTransactionRepo) InsertDetails(tx *sql.Tx, transactionID int, items []CheckoutItem, products []LockedProduct) error {
    args := m.Called(tx, transactionID, items, products)
    return args.Error(0)
}

func TestTransactionService_Checkout_EmptyItems(t *testing.T) {
    repo := new(mockTransactionRepo)
    svc := NewTransactionService(repo)
    _, err := svc.Checkout(CheckoutRequest{Items: []CheckoutItem{}})
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "at least one item")
}

func TestTransactionService_Checkout_TooManyItems(t *testing.T) {
    repo := new(mockTransactionRepo)
    svc := NewTransactionService(repo)
    items := make([]CheckoutItem, 101)
    for i := range items {
        items[i] = CheckoutItem{ProductID: i + 1, Quantity: 1}
    }
    _, err := svc.Checkout(CheckoutRequest{Items: items})
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "maximum 100")
}

func TestTransactionService_Checkout_InvalidQuantity(t *testing.T) {
    repo := new(mockTransactionRepo)
    svc := NewTransactionService(repo)
    _, err := svc.Checkout(CheckoutRequest{Items: []CheckoutItem{{ProductID: 1, Quantity: 0}}})
    assert.Error(t, err)
}

func TestTransactionService_Checkout_Success(t *testing.T) {
    repo := new(mockTransactionRepo)
    svc := NewTransactionService(repo)
    mockTx := &sql.Tx{}
    products := []LockedProduct{{ID: 1, Name: "Kopi", Price: 10000, Stock: 50}}
    repo.On("BeginTx").Return(mockTx, nil)
    repo.On("LockProducts", mockTx, []int{1}).Return(products, nil)
    repo.On("UpdateStock", mockTx, 1, 2).Return(nil)
    repo.On("InsertTransaction", mockTx, 20000).Return(1, nil)
    repo.On("InsertDetails", mockTx, 1, []CheckoutItem{{ProductID: 1, Quantity: 2}}, products).Return(nil)
    repo.On("FindByID", 1).Return(&Transaction{ID: 1, TotalAmount: 20000}, nil)
    txn, err := svc.Checkout(CheckoutRequest{Items: []CheckoutItem{{ProductID: 1, Quantity: 2}}})
    assert.NoError(t, err)
    assert.Equal(t, 20000, txn.TotalAmount)
    repo.AssertExpectations(t)
}
```

- [ ] **Step 4: Run transaction tests**

Run: `go test ./internal/domain/transaction/... -v`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/category/category_service_test.go internal/domain/transaction/transaction_service_test.go
git commit -m "test: add service tests for category and transaction domains"
```

---

### Task 10: Add migration for transactions list index

**Files:**
- Create: `migrations/000005_add_transactions_list_index.up.sql`
- Create: `migrations/000005_add_transactions_list_index.down.sql`

- [ ] **Step 1: Write up migration**

```sql
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transaction_details_transaction_id ON transaction_details(transaction_id);
```

- [ ] **Step 2: Write down migration**

```sql
DROP INDEX IF EXISTS idx_transactions_created_at;
DROP INDEX IF EXISTS idx_transaction_details_transaction_id;
```

- [ ] **Step 3: Commit**

```bash
git add migrations/000005_*
git commit -m "feat: add indexes for transactions listing performance"
```

---

### Task 11: Final verification

- [ ] **Step 1: Full build check**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 2: Run all tests**

Run: `go test ./...`
Expected: all tests pass

- [ ] **Step 3: Verify old routes work**

```bash
go run ./cmd/api/ &
sleep 2
curl -s http://localhost:8080/health | jq .
curl -s http://localhost:8080/api/v1/health | jq .
kill %1
```

Expected: Both return `{"status":"OK"}`

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: final verification phase 1 complete"
```
