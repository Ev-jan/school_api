package router

import (
	"net/http"
	"schoolapi/internal/api/handlers"
)

func execsRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /execs", handlers.GetExecs)
	mux.HandleFunc("POST /execs", handlers.AddExecs)
	mux.HandleFunc("PATCH /execs", handlers.PatchExecs)

	mux.HandleFunc("GET /execs/{id}", handlers.GetExec)
	mux.HandleFunc("PATCH /execs/{id}", handlers.PatchExec)
	mux.HandleFunc("DELETE /execs/{id}", handlers.DeleteExec)

	mux.HandleFunc("POST /execs/{id}/update-password", handlers.UpdatePassword)

	mux.HandleFunc("POST /execs/login", handlers.Login)
	mux.HandleFunc("POST /execs/logout", handlers.Logout)
	mux.HandleFunc("POST /execs/forgot-password", handlers.ForgotPassword)
	mux.HandleFunc("POST /execs/reset-password/reset/{resetcode}", handlers.ResetPassword)

	return mux
}
