package middlewares

import (
	"fmt"
	"net/http"
	"strings"
)

func ExcludePaths(middleware func(http.Handler) http.Handler, excludedPaths ...string) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, path := range excludedPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}
			fmt.Println("Sent response from Cors middleware")
			middleware(next).ServeHTTP(w, r)
		})
	}
}
