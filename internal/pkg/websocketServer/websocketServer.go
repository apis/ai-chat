package websocketServer

import (
	"github.com/google/uuid"
	"net/http"
)

type WebsocketServer interface {
	Handler(responseWriter http.ResponseWriter, request *http.Request)
	Publish(id uuid.UUID, message []byte)
}
