package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultPage    = 1
	defaultPerPage = 50
	maxPerPage     = 200
)

type Pagination struct {
	Page    int
	PerPage int
}

func DecodeJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra interface{}
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("invalid json body")
	}
	return nil
}

func RequireNonEmpty(fields map[string]string) error {
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}

func ParsePagination(r *http.Request) (Pagination, error) {
	page := defaultPage
	perPage := defaultPerPage
	var err error
	if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page <= 0 {
			return Pagination{}, fmt.Errorf("page must be a positive integer")
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("per_page")); raw != "" {
		perPage, err = strconv.Atoi(raw)
		if err != nil || perPage <= 0 {
			return Pagination{}, fmt.Errorf("per_page must be a positive integer")
		}
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return Pagination{Page: page, PerPage: perPage}, nil
}

func Paginate[T any](items []T, p Pagination) []T {
	if len(items) == 0 {
		return []T{}
	}
	start := (p.Page - 1) * p.PerPage
	if start >= len(items) {
		return []T{}
	}
	end := start + p.PerPage
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func WritePaginationHeaders(w http.ResponseWriter, p Pagination, total int) {
	w.Header().Set("X-Page", strconv.Itoa(p.Page))
	w.Header().Set("X-Per-Page", strconv.Itoa(p.PerPage))
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
}
