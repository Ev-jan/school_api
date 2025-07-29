package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	mw "schoolapi/internal/api/middlewares"
	"schoolapi/internal/api/router"
	"schoolapi/internal/repository/sqlconnect"
	"schoolapi/pkg/utils"
	"time"

	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading env vars", err)
	}
	_, err = sqlconnect.ConnectDB()

	if err != nil {
		log.Fatal("Error connecting to the DB:", err)
		return
	}

	port := os.Getenv("API_PORT")
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
		WhiteList:                   []string{"sort-by", "sort-order", "class", "first-name", "last-name", "email", "subject"},
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

	fmt.Println("Server running on port ", port)
	log.Fatal(server.ListenAndServeTLS(cert, key))
}
