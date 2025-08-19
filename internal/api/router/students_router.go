package router

import (
	"net/http"
	"schoolapi/internal/api/handlers"
)

func studentsRouter() *http.ServeMux {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /students", handlers.GetStudents)
	mux.HandleFunc("POST /students", handlers.AddStudents)
	mux.HandleFunc("PATCH /students", handlers.PatchStudents)
	mux.HandleFunc("DELETE /students", handlers.DeleteStudents)

	mux.HandleFunc("GET /students/{id}", handlers.GetStudent)
	mux.HandleFunc("PUT /students/{id}", handlers.UpdateStudent)
	mux.HandleFunc("PATCH /students/{id}", handlers.PatchStudent)
	mux.HandleFunc("DELETE /students/{id}", handlers.DeleteStudent)

	return mux
}
