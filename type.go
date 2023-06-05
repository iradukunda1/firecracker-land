package main

import (
	"net"
	"net/http"
)

type CreateRequest struct {
	Name        string `json:"name" validate:"required"`
	DockerImage string `json:"docker-image" validate:"required"`
}

type CreateResponse struct {
	ID     string `json:"id,omitempty"`
	PID    int64  `json:"pid,omitempty"`
	Name   string `json:"name,omitempty"`
	IpAddr string `json:"ip_address,omitempty"`
	Agent  net.IP `json:"agent,omitempty"`
}

type DeleteRequest struct {
	ID string `json:"id" validate:"required"`
}

// responseMessage
type responseMessage struct {
	Message string `json:"message"`
}

type Middleware func(h http.Handler) http.Handler
