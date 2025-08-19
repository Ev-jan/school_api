package router

import (
	"net/http"
)

func MainRouter() *http.ServeMux {

	tRouter := teachersRouter()
	sRouter := studentsRouter()

	sRouter.Handle("/", execsRouter())
	tRouter.Handle("/", sRouter)

	return tRouter
}
