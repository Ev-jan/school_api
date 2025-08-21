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

	HPPOptions := mw.HPPOptions{
		CheckQuery:                  true,
		CheckBody:                   true,
		CheckBodyOnlyForContentType: "application/x-www-form-urlencoded",
		WhiteList:                   []string{"sort_by", "name", "age", "class"},
	}

	jwtMiddleware := mw.ExcludePaths(mw.JWT, "/execs/login", "/execs/forgot-password", "/execs/reset-password/reset")
	secureMux := utils.ApplyMiddleware(router.MainRouter(), mw.SecurityHeaders, mw.Compression, mw.Hpp(HPPOptions), mw.XSS, jwtMiddleware, mw.ResponseTime, rl.Middleware, mw.Cors)

	server := &http.Server{
		Addr:      port,
		Handler:   secureMux,
		TLSConfig: tlsConfig,
	}

	fmt.Println("Server running on port ", port)
	log.Fatal(server.ListenAndServeTLS(cert, key))
}
