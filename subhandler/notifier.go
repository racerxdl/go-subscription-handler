package subhandler

// Notifier is a interface used for pub-sub like interactions
type Notifier interface {
	// Subscribes the specified function from topic
	Subscribe(topic string, cb func(data map[string]interface{}))

	// Unsubscribes the specified function from topic
	Unsubscribe(topic string, cb func(data map[string]interface{}))

	// Notify notifies the topic with the optional data
	Notify(topic string, data map[string]interface{})
}
