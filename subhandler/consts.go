package subhandler

import "time"

const (
	// CTXSubscribeTopic is the constant string that represents the subscribe function in context
	CTXSubscribeTopic = "subscription-subscribe-function"
	// CTXGraphQLWSID is the constant string that represents the websocket connection id in context
	CTXGraphQLWSID = "graphql-ws-id"
)

const (
	startMessageTimeout = 2 * time.Second
)
