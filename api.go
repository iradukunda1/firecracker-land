package main

import (
	"net/http"

	"github.com/go-chi/chi"
)

func handler() http.Handler {
	r := chi.NewRouter()

	//Create vm instance
	r.Post("/create", createVmHandler)

	//Delete vm instance
	r.Delete("/delete", deleteVmHandler)

	return r
}
