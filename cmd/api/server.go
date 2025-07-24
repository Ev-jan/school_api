package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type person struct {
	Name string `json:"name"`
	Age int `json:"age"`
	City string `json:"city"`
}

func rootHandler (w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("root route"))
	}

func teachersHandler (w http.ResponseWriter, r *http.Request) {
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

func execsHandler (w http.ResponseWriter, r *http.Request) {
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
	// create custom server
	server := &http.Server{
		Addr: port,
		Handler: nil,
		TLSConfig: tlsConfig,

	}

	log.Fatal(server.ListenAndServeTLS(cert, key))
}