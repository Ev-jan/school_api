package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	mw "schoolapi/internal/api/middlewares"
	"strings"
	"time"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("root route"))
}

func teachersHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("teachers route"))
	switch r.Method {
	case http.MethodGet:
		params := r.URL.Query()
		sortBy := params.Get("sort-by")
		key := params.Get("key")
		sortOrder := params.Get("sort-order")

		fmt.Println(sortBy, key, sortOrder)
	}
}

func execsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("execs route"))
	switch r.Method {
	case http.MethodGet:
		path := strings.TrimPrefix(r.URL.Path, "/teachers/")
		userID := strings.TrimSuffix(path, "/")
		fmt.Println("User id:", userID)
	default:
		fmt.Fprintf(w, "all execs here")
	}
}

type Middleware func(http.Handler) http.Handler

func applyMiddleware(handler http.Handler, middlewares ...Middleware)http.Handler {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}

func main() {
	port := ":3000"

	cert := "cert.pem"
	key := "key.pem"

	mux := http.NewServeMux()

	fmt.Println("Server is running on port:", port)

	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/teachers", teachersHandler)
	mux.HandleFunc("/execs", execsHandler)

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	rl := mw.NewRateLimiter(5, time.Minute)

	hppOptions := mw.HPPOptions{
		CheckQuery: true,
		CheckBody: true,
		CheckBodyOnlyForContentType: "application/x-www-form-urlencoded",
		WhiteList: []string{"sort-by","sort-order", "name", "age", "class"},
	}
	// Prepare middlewares
	// secureMux := mw.Cors(rl.Middleware(mw.ResponseTime(mw.SecurityHeaders(mw.Compression(mw.Hpp(hppOptions)(mux))))))
	secureMux := applyMiddleware(mux, mw.Hpp(hppOptions), mw.Compression, mw.SecurityHeaders, mw.ResponseTime, rl.Middleware, mw.Cors)
	// create custom server
	server := &http.Server{
		Addr:      port,
		Handler:   secureMux,
		TLSConfig: tlsConfig,
	}

	log.Fatal(server.ListenAndServeTLS(cert, key))
}
