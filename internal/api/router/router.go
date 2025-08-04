package router

import (
	"net/http"
	"schoolapi/internal/api/handlers"
)

func Router() *http.ServeMux {

	mux := http.NewServeMux()

	mux.HandleFunc("/", handlers.RootHandler)
	mux.HandleFunc("GET /teachers/", handlers.GetTeachers)
	mux.HandleFunc("POST /teachers/", handlers.AddTeachers)
	mux.HandleFunc("PATCH /teachers/", handlers.PatchTeachers)
	mux.HandleFunc("DELETE /teachers/", handlers.DeleteTeachers)

	mux.HandleFunc("GET /teachers/{id}", handlers.GetTeacher)
	mux.HandleFunc("PUT /teachers/{id}", handlers.UpdateTeacher)
	mux.HandleFunc("PATCH /teachers/{id}", handlers.PatchTeacher)
	mux.HandleFunc("DELETE /teachers/{id}", handlers.DeleteTeacher)

	mux.HandleFunc("/students/", handlers.StudentsHandler)
	mux.HandleFunc("/execs/", handlers.ExecsHandler)

	return mux
}
