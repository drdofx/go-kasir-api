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
