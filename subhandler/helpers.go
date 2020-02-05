package subhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
)

// sendGraphWSMessage sends a graphQLWSMessage object through a websocket
func sendGraphWSMessage(socket *websocket.Conn, msg graphQLWSMessage) error {
	data, _ := json.Marshal(msg)
	return socket.WriteMessage(websocket.TextMessage, data)
}

// Subscribe calls a subscription function from handler that has been passed through context
func Subscribe(ctx context.Context, topic string) error {
	mci := ctx.Value(CTXSubscribeTopic)
	if mci != nil {
		mci.(func(string))(topic)
		return nil
	}

	return fmt.Errorf("no subscribe function in context")
}
