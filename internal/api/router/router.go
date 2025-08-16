package router

import (
	"net/http"
	"schoolapi/internal/api/handlers"
)

func Router() *http.ServeMux {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", handlers.RootHandler)

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

	mux.HandleFunc("GET /students", handlers.GetStudents)
	mux.HandleFunc("POST /students", handlers.AddStudents)
	mux.HandleFunc("PATCH /students", handlers.PatchStudents)
	mux.HandleFunc("DELETE /students", handlers.DeleteStudents)

	mux.HandleFunc("GET /students/{id}", handlers.GetStudent)
	mux.HandleFunc("PUT /students/{id}", handlers.UpdateStudent)
	mux.HandleFunc("PATCH /students/{id}", handlers.PatchStudent)
	mux.HandleFunc("DELETE /students/{id}", handlers.DeleteStudent)

	mux.HandleFunc("GET /execs", handlers.ExecsHandler)

	return mux
}
