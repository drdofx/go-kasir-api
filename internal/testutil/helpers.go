package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func NewRequest(method, path, body string) *http.Request {
	return httptest.NewRequest(method, path, strings.NewReader(body))
}

func NewJSONRequest(method, path string, body any) *http.Request {
	b, _ := json.Marshal(body)
	return httptest.NewRequest(method, path, strings.NewReader(string(b)))
}

func AssertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	assert.Equal(t, want, rec.Code, "status code mismatch")
}

func ReadBody(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	b, err := io.ReadAll(rec.Body)
	assert.NoError(t, err)
	return string(b)
}
