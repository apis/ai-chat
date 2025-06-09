package web

import (
	"net/http"
)

type Handler struct {
	Request        func(request *http.Request, simulatedDelay int) *Response
	SimulatedDelay int
}

func (instance Handler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	response := instance.Request(request, instance.SimulatedDelay)
	response.Write(responseWriter)
}
