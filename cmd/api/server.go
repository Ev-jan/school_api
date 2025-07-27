package main

import (
	"crypto/tls"
	"log"
	"net/http"
	mw "schoolapi/internal/api/middlewares"
	"schoolapi/internal/api/router"
	"schoolapi/pkg/utils"
	"time"
)

func main() {
	port := ":3000"
	cert := "cert.pem"
	key := "key.pem"

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	rl := mw.NewRateLimiter(5, time.Minute)

	hppOptions := mw.HPPOptions{
		CheckQuery:                  true,
		CheckBody:                   true,
		CheckBodyOnlyForContentType: "application/x-www-form-urlencoded",
		WhiteList:                   []string{"sort-by", "sort-order", "name", "age", "class", "first_name", "last_name"},
	}
	// Prepare middlewares
	// secureMux := mw.Cors(rl.Middleware(mw.ResponseTime(mw.SecurityHeaders(mw.Compression(mw.Hpp(hppOptions)(mux))))))
	// secureMux := applyMiddleware(mux, mw.Hpp(hppOptions), mw.Compression, mw.SecurityHeaders, mw.ResponseTime, rl.Middleware, mw.Cors)
	secureMux := utils.ApplyMiddleware(router.Router(), mw.Hpp(hppOptions), rl.Middleware)
	// create custom server
	server := &http.Server{
		Addr:      port,
		Handler:   secureMux,
		TLSConfig: tlsConfig,
	}

	log.Fatal(server.ListenAndServeTLS(cert, key))
}
