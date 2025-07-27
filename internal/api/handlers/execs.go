package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

func ExecsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("execs route"))
	switch r.Method {
	case http.MethodGet:
		path := strings.TrimPrefix(r.URL.Path, "/execs/")
		userID := strings.TrimSuffix(path, "/")
		fmt.Println("User id:", userID)
	default:
		fmt.Fprintf(w, "all execs here")
	}
}
