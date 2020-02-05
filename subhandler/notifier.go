package subhandler

// Notifier is a interface used for pub-sub like interactions
type Notifier interface {
	Subscribe(string, func())
	Unsubscribe(string, func())
	Notify(string)
}
