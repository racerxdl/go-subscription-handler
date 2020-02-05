package subhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"net/http"
	"sync"
	"time"
)

type SubscriptionHandler struct {
	upgrader websocket.Upgrader
	notifier Notifier
	schema   graphql.Schema
}

func MakeSubscriptionHandler(notifier Notifier, schema graphql.Schema) *SubscriptionHandler {
	return &SubscriptionHandler{
		notifier: notifier,
		schema:   schema,
		upgrader: websocket.Upgrader{
			Subprotocols: []string{"graphql-ws"},
		},
	}
}

func (sh *SubscriptionHandler) SetCheckOrigin(fn func(r *http.Request) bool) {
	sh.upgrader.CheckOrigin = fn
}

func (sh *SubscriptionHandler) dryRunQuery(ctx context.Context, socket *websocket.Conn, queryData *graphQLWSRequest) bool {
	res := graphql.Do(graphql.Params{
		Schema:         sh.schema,
		RequestString:  queryData.Payload.Query,
		VariableValues: queryData.Payload.Variables,
		OperationName:  queryData.Payload.OperationName,
		Context:        ctx,
	})

	return len(res.Errors) == 0
}

func (sh *SubscriptionHandler) runQuery(ctx context.Context, socket *websocket.Conn, queryData *graphQLWSRequest, rootObject map[string]interface{}) bool {
	res := graphql.Do(graphql.Params{
		Schema:         sh.schema,
		RequestString:  queryData.Payload.Query,
		VariableValues: queryData.Payload.Variables,
		OperationName:  queryData.Payload.OperationName,
		Context:        ctx,
		RootObject:     rootObject,
	})

	err := sendGraphWSMessage(socket, graphQLWSMessage{
		Id:      queryData.Id,
		Type:    "data",
		Payload: res,
	})

	return len(res.Errors) == 0 && err == nil
}

func (sh *SubscriptionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try upgrade connection to Websocket
	socket, err := sh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Error upgrading connection: %s", err)))
		return
	}
	defer socket.Close()

	// Create new UUID for this websocket connection
	wsId := uuid.New().String()

	// Inject on Context
	ctx := context.WithValue(r.Context(), CTXGraphQLWSID, wsId)

	// On Change Callbacks
	onChangeChannel := make(chan map[string]interface{})
	onChange := func(data map[string]interface{}) {
		onChangeChannel <- data
	}

	// Track Subscribed Topics
	topics := make(map[string]string, 0)
	l := sync.Mutex{}

	// Create the subscribe function for the query resolvers
	subTopic := func(topic string) {
		l.Lock()
		if _, ok := topics[topic]; !ok {
			if sh.notifier != nil {
				sh.notifier.Subscribe(topic, onChange)
			}
			topics[topic] = topic
		}
		l.Unlock()
	}

	// Inject subscribe function in query resolver
	ctx = context.WithValue(ctx, CTXSubscribeTopic, subTopic)

	// Create a callback channel for graphql websocket messages
	newMessage := make(chan graphQLWSMessage)

	// Start the clientLoop in a async goroutine
	go sh.clientLoop(socket, newMessage)

	// Wait for the message with type `start`
	var msg graphQLWSMessage

	msg.Type = "nil"

	select {
	case msg = <-newMessage:
	case <-time.After(startMessageTimeout): // Timeout
	}

	if msg.Type == "nil" {
		return
	}

	m, err := msg.ToGraphQLRequest()

	if err != nil {
		// Timed out
		return
	}

	// Run the first query
	subRunning := sh.dryRunQuery(ctx, socket, m)

	// Loop until a stop message or disconnect is received
	for subRunning {
		select {
		case data := <-onChangeChannel:
			subRunning = sh.runQuery(ctx, socket, m, data)
		case msg := <-newMessage:
			if msg.Type == "stop" || msg.Type == "disconnected" {
				subRunning = false
				break
			}
		}
	}

	// Clean-up all subscribed topics
	if sh.notifier != nil {
		for _, topic := range topics {
			sh.notifier.Unsubscribe(topic, onChange)
		}
	}
}

func (sh *SubscriptionHandler) clientLoop(socket *websocket.Conn, newMessage chan graphQLWSMessage) error {
	for {
		_, msgBytes, err := socket.ReadMessage()
		if err != nil {
			newMessage <- graphQLWSMessage{
				Type: "disconnected",
			}
			return err
		}

		queryData := graphQLWSMessage{}

		err = json.Unmarshal(msgBytes, &queryData)

		if err != nil {
			newMessage <- graphQLWSMessage{
				Type:    "error",
				Payload: err.Error(),
			}
			return err
		}

		if queryData.Type == "start" || queryData.Type == "stop" {
			// Only notify start/stop
			newMessage <- queryData
		}

		if queryData.Type == "stop" {
			// Break this loop
			return nil
		}
	}
}
