# Golang Simple GraphQL Subscription Handler

## Usage

Subscription node:

```go
var rootSubscriptions = graphql.ObjectConfig{
    Name: "RootSubscriptions",
    Fields: graphql.Fields{
        "serverTime": &graphql.Field{
            Type: graphql.Float,
            Resolve: func(p graphql.ResolveParams) (interface{}, error) {
                err := subhandler.Subscribe(p.Context, "serverTime")

                if p.Source != nil {
                    // We received a event data
                    v, ok := p.Source.(map[string]interface{})
                    if ok && v["time"] != nil {
                        return v["time"], nil
                    }
                }

                // We didn't receive a event data, so resolve normally
                return time.Now().String(), err
            },
        },
    },
}

var rootQuery = graphql.ObjectConfig{
    Name: "RootQuery",
    Fields: graphql.Fields{
        "serverTime": &graphql.Field{
            Type: graphql.Float,
            Resolve: func(p graphql.ResolveParams) (interface{}, error) {
                return time.Now().Unix(), nil
            },
        },
    },
}


var schemaConfig = graphql.SchemaConfig{
    Query:        graphql.NewObject(rootQuery),
    Subscription: graphql.NewObject(rootSubscriptions),
}

func GetSchema() (graphql.Schema, error) {
    return graphql.NewSchema(schemaConfig)
}

```

Main Code:

```go
package main

import (
    "github.com/asaskevich/EventBus"
    "github.com/graphql-go/handler"
    "github.com/racerxdl/go-subscription-handler/subhandler"
    "net/http"
    "time"
    "fmt"
)

// Create a notifier
type BusNotifier struct {
    bus EventBus.Bus // You can use any pub-sub lib for that
}

func MakeBusNotifier(bus EventBus.Bus) *BusNotifier {
    return &BusNotifier{
        bus: bus,
    }
}

func (bn *BusNotifier) Subscribe(topic string, cb func(data map[string]interface{})) {
    bn.bus.Subscribe(topic, cb)
}

func (bn *BusNotifier) Unsubscribe(topic string, cb func(data map[string]interface{})) {
    bn.bus.Unsubscribe(topic, cb)
}

func (bn *BusNotifier) Notify(topic string, data map[string]interface{}) {
    bn.bus.Publish(topic, data)
}

// Initialize a Sub Handler
func main() {

    schema, _ := GetSchema()

    // Create normal mutation / query handlers
    h := handler.New(&handler.Config{
        Schema:     &schema,
        Pretty:     true,
        Playground: true,
    })
    
    // Initialize our notifier
    notifier := MakeBusNotifier(EventBus.New())

    // Create a goroutine to send notifications through notifier
    go func() {
        for {
            data := map[string]interface{}{
                "time": time.Now().String(),
            }
            notifier.Notify("serverTime", data)
            time.Sleep(time.Second) // Sleep some interval

            // This also works, makes the resolver decide what to do
            notifier.Notify("serverTime", nil)
        }
    }()

    // Create the subscription handler using notifier and schema
    sh := subhandler.MakeSubscriptionHandler(notifier, schema)

    // Optional, check origin of the request
    sh.SetCheckOrigin(func(r *http.Request) bool {
        // Accept any origin
        return true
    })

    // Attach the normal query / mutation handlers
    http.Handle("/", h)
    
    // Attach the subscription handlers
    http.Handle("/subscriptions", sh)
    
    fmt.Println("Listening in :8080")
    http.ListenAndServe(":8080", nil)
}
```

