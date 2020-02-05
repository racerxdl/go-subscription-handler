package main

import (
    "fmt"
    "github.com/asaskevich/EventBus"
    "github.com/graphql-go/graphql"
    "github.com/graphql-go/handler"
    "github.com/racerxdl/go-subscription-handler/subhandler"
    "net/http"
    "time"
)

var rootSubscriptions = graphql.ObjectConfig{
    Name: "RootSubscriptions",
    Fields: graphql.Fields{
        "serverTime": &graphql.Field{
            Type: graphql.String,
            Resolve: func(p graphql.ResolveParams) (interface{}, error) {
                err := subhandler.Subscribe(p.Context, "serverTime")
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


// Create a notifier
type BusNotifier struct {
    bus EventBus.Bus // You can use any pub-sub lib for that
}

func MakeBusNotifier(bus EventBus.Bus) *BusNotifier {
    return &BusNotifier{
        bus: bus,
    }
}

func (bn *BusNotifier) Subscribe(topic string, cb func()) {
    bn.bus.Subscribe(topic, cb)
}

func (bn *BusNotifier) Unsubscribe(topic string, cb func()) {
    bn.bus.Unsubscribe(topic, cb)
}

func (bn *BusNotifier) Notify(topic string) {
    bn.bus.Publish(topic)
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
            notifier.Notify("serverTime")
            time.Sleep(time.Second) // Sleep some interval
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
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        panic(err)
    }
}