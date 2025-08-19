package router

import (
	"net/http"
	"schoolapi/internal/api/handlers"
)

func teachersRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /teachers", handlers.GetTeachers)
	mux.HandleFunc("POST /teachers", handlers.AddTeachers)
	mux.HandleFunc("PATCH /teachers", handlers.PatchTeachers)
	mux.HandleFunc("DELETE /teachers", handlers.DeleteTeachers)

	mux.HandleFunc("GET /teachers/{id}", handlers.GetTeacher)
	mux.HandleFunc("PUT /teachers/{id}", handlers.UpdateTeacher)
	mux.HandleFunc("PATCH /teachers/{id}", handlers.PatchTeacher)
	mux.HandleFunc("DELETE /teachers/{id}", handlers.DeleteTeacher)

	mux.HandleFunc("GET /teachers/{id}/students", handlers.GetStudentsByTeacherId)
	mux.HandleFunc("GET /teachers/{id}/studentcount", handlers.GetStudentsCountByTeacherId)
	return mux
}
