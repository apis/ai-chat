package websocketServer

import (
	"ai-chat/internal/pkg/cookies"
	"context"
	"errors"
	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"net/http"
	"sync"
	"time"
)

type messagePacket struct {
	Id      uuid.UUID
	Message []byte
}

const subscriberMessageBufferSize = 16

type websocketServerImpl struct {
	mutex       sync.Mutex
	subscribers map[*serverSubscriber]any
}

func New() WebsocketServer {
	wsNotificationServer := &websocketServerImpl{
		subscribers: make(map[*serverSubscriber]any),
	}
	return wsNotificationServer
}

type serverSubscriber struct {
	messageChannel chan messagePacket
	closeSlow      func()
}

func (instance *websocketServerImpl) Handler(responseWriter http.ResponseWriter, request *http.Request) {
	id := cookies.GetIdFromCookie(request)
	if id == uuid.Nil {
		http.Error(responseWriter, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err := instance.subscribe(responseWriter, request, id)
	if errors.Is(err, context.Canceled) {
		return
	}

	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}

	if err != nil {
		log.Error().Err(err).Msg("subscribe() failed")
		return
	}
}

func (instance *websocketServerImpl) subscribe(responseWriter http.ResponseWriter, request *http.Request, id uuid.UUID) error {
	websocketConnection, err := websocket.Accept(responseWriter, request, nil)
	if err != nil {
		// Accept will write a response to responseWriter on all errors
		log.Error().Err(err).Msg("websocket.Accept() failed")
		return err
	}

	defer func() {
		err := websocketConnection.CloseNow()
		if err != nil {
			log.Error().Err(err).Msg("websocket.Conn.CloseNow() failed")
		}
	}()

	subscriber := &serverSubscriber{
		messageChannel: make(chan messagePacket, subscriberMessageBufferSize),
		closeSlow: func() {
			if websocketConnection != nil {
				err := websocketConnection.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
				if err != nil {
					log.Error().Err(err).Msg("websocket.Conn.Close() failed")
				}
			}
		},
	}

	instance.addSubscriber(subscriber)
	defer instance.deleteSubscriber(subscriber)

	ctx := websocketConnection.CloseRead(context.Background())

	for {
		select {
		case packet := <-subscriber.messageChannel:
			if packet.Id == id {
				err := writeTimeout(ctx, time.Second*5, websocketConnection, packet.Message)
				if err != nil {
					return err
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (instance *websocketServerImpl) Publish(id uuid.UUID, message []byte) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	for subscriber := range instance.subscribers {
		select {
		case subscriber.messageChannel <- messagePacket{Id: id, Message: message}:
		default:
			go subscriber.closeSlow()
		}
	}
}

func (instance *websocketServerImpl) addSubscriber(subscriber *serverSubscriber) {
	instance.mutex.Lock()
	instance.subscribers[subscriber] = struct{}{}
	instance.mutex.Unlock()
}

func (instance *websocketServerImpl) deleteSubscriber(subscriber *serverSubscriber) {
	instance.mutex.Lock()
	delete(instance.subscribers, subscriber)
	instance.mutex.Unlock()
}

func writeTimeout(ctx context.Context, timeout time.Duration, websocketConnection *websocket.Conn, msg []byte) error {
	writeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := websocketConnection.Write(writeCtx, websocket.MessageText, msg)
	if err != nil {
		log.Error().Err(err).Msg("websocket.Conn.Write() failed")
		return err
	}

	return nil
}
