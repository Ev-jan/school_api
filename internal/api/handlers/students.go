package handlers

import (
	"fmt"
	"net/http"
)

func StudentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("students route"))
	switch r.Method {
	case http.MethodGet:
		fmt.Println("Method get")
	case http.MethodPost:
		addTeachers(w, r)
		fmt.Println("Method post")
	}
}
