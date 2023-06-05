package main

import (
	"net/http"

	"github.com/go-chi/chi"
)

func handler() http.Handler {
	r := chi.NewRouter()

	r.Post("/create", CreateVmHandler)
	r.Delete("/delete", DeleteVmHandler)
	r.Post("/stop", StopVmHandler)
	r.Post("/resume", ResumeVmHandler)
	r.Get("/list", ListVmsHandler)

	return r
}
