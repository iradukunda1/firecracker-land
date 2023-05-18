package main

import (
	"net/http"

	"github.com/go-chi/chi"
)

func handler() http.Handler {
	r := chi.NewRouter()

	//Create vm instance
	r.Post("/create", createRequestHandler)

	//Delete vm instance
	r.Delete("/delete", deleteRequestHandler)

	return r
}
