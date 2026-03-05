# go-messaging-amqp

<p align="center">
  <strong>AMQP/RabbitMQ transport for gomessaging -- opinionated event-driven messaging with deterministic topology, CloudEvents, and OpenTelemetry.</strong>
</p>

<p align="center">
  <a href="https://github.com/sparetimecoders/go-messaging-amqp/actions"><img alt="CI" src="https://github.com/sparetimecoders/go-messaging-amqp/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://pkg.go.dev/github.com/sparetimecoders/go-messaging-amqp"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/sparetimecoders/go-messaging-amqp.svg"></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-blue.svg"></a>
</p>

---

This package implements the [gomessaging specification](https://github.com/sparetimecoders/messaging) for AMQP 0-9-1 (RabbitMQ). It provides deterministic exchange/queue naming, CloudEvents 1.0 metadata, quorum queues with single active consumer by default, publisher confirms, and full OpenTelemetry tracing and Prometheus metrics.

> **Deep dives**: See the [docs/](docs/) directory for detailed guides on [connection lifecycle](docs/connection.md), [consumers](docs/consumers.md), [publishers](docs/publishers.md), [request-response](docs/request-response.md), [observability](docs/observability.md), and [topology & naming](docs/topology.md).

## Installation

```sh
go get github.com/sparetimecoders/go-messaging-amqp
```

Requires Go 1.24+.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sparetimecoders/go-messaging-amqp"
    "github.com/sparetimecoders/messaging/specification/spec"
)

type OrderCreated struct {
    OrderID string `json:"order_id"`
    Amount  int    `json:"amount"`
}

func main() {
    ctx := context.Background()

    conn, err := amqp.NewFromURL("order-service", "amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Fatal(err)
    }

    pub := amqp.NewPublisher()

    err = conn.Start(ctx,
        amqp.EventStreamPublisher(pub),
        amqp.EventStreamConsumer("Order.Created", func(ctx context.Context, e spec.ConsumableEvent[OrderCreated]) error {
            fmt.Printf("received order %s, amount %d\n", e.Payload.OrderID, e.Payload.Amount)
            return nil
        }),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    err = pub.Publish(ctx, "Order.Created", OrderCreated{OrderID: "abc-123", Amount: 42})
    if err != nil {
        log.Fatal(err)
    }
}
```

## Messaging Patterns

### Event Stream

Publish domain events to the shared `events.topic.exchange`. Any number of services subscribe by routing key. Durable consumers use quorum queues that survive restarts. Transient consumers auto-delete on disconnect.

```go
pub := amqp.NewPublisher()

conn.Start(ctx,
    // Publisher
    amqp.EventStreamPublisher(pub),

    // Durable consumer (quorum queue, single active consumer)
    amqp.EventStreamConsumer("Order.Created", func(ctx context.Context, e spec.ConsumableEvent[OrderCreated]) error {
        return processOrder(e.Payload)
    }),

    // Transient consumer (auto-delete queue, ephemeral)
    amqp.TransientEventStreamConsumer("Order.*", func(ctx context.Context, e spec.ConsumableEvent[any]) error {
        log.Println("transient:", e.DeliveryInfo.Key)
        return nil
    }),
)

pub.Publish(ctx, "Order.Created", OrderCreated{OrderID: "abc-123", Amount: 42})
```

### Custom Stream

Same as event stream but on a named exchange instead of the default `events` exchange. Use when events belong to a separate domain.

```go
pub := amqp.NewPublisher()

conn.Start(ctx,
    amqp.StreamPublisher("audit", pub),
    amqp.StreamConsumer("audit", "User.Login", func(ctx context.Context, e spec.ConsumableEvent[UserLogin]) error {
        return recordAuditEntry(e.Payload)
    }),
)

pub.Publish(ctx, "User.Login", UserLogin{UserID: "u-42"})
```

### Service Request-Response

Synchronous request-reply between services. The handler receives a request and returns a response that is automatically published back to the calling service.

```go
// --- billing-service ---
billingConn.Start(ctx,
    amqp.RequestResponseHandler("Invoice.Create", func(ctx context.Context, e spec.ConsumableEvent[InvoiceRequest]) (InvoiceResponse, error) {
        invoice := createInvoice(e.Payload)
        return InvoiceResponse{InvoiceID: invoice.ID}, nil
    }),
)

// --- order-service (caller) ---
pub := amqp.NewPublisher()

orderConn.Start(ctx,
    amqp.ServicePublisher("billing-service", pub),
    amqp.ServiceResponseConsumer("billing-service", "Invoice.Create", func(ctx context.Context, e spec.ConsumableEvent[InvoiceResponse]) error {
        fmt.Println("invoice created:", e.Payload.InvoiceID)
        return nil
    }),
)

pub.Publish(ctx, "Invoice.Create", InvoiceRequest{OrderID: "abc-123"})
```

### Queue Publish

Direct publish to a named queue using sender-selected distribution. The sender picks the destination queue. Useful for work queues and task distribution.

```go
pub := amqp.NewPublisher()

conn.Start(ctx,
    amqp.QueuePublisher(pub, "email-tasks"),
)

pub.Publish(ctx, "", EmailTask{To: "user@example.com", Subject: "Hello"})
```

## Configuration

### Connection Options

Setup functions are passed to `conn.Start()` to configure the connection before consumers start.

| Function | Description |
|----------|-------------|
| `WithLogger(logger)` | Use a custom `*slog.Logger`. Defaults to `slog.Default()`. |
| `WithTracing(tp)` | Set a `trace.TracerProvider`. Defaults to `otel.GetTracerProvider()`. |
| `WithPropagator(p)` | Set a `propagation.TextMapPropagator`. Defaults to `otel.GetTextMapPropagator()`. |
| `WithPrefetchLimit(n)` | Number of messages to prefetch per consumer channel. Default: 20. Set to 1 for strict round-robin. |
| `WithSpanNameFn(fn)` | Custom span name for consume operations. Receives `spec.DeliveryInfo`. |
| `WithPublishSpanNameFn(fn)` | Custom span name for publish operations. Receives `(exchange, routingKey)`. |
| `WithLegacySupport()` | Enrich incoming messages that lack CloudEvents headers with synthetic metadata. |
| `CloseListener(ch)` | Receive `error` values on channel/connection close events. |
| `WithNotificationChannel(ch)` | Receive `spec.Notification` on successful event processing. |
| `WithErrorChannel(ch)` | Receive `spec.ErrorNotification` on failed event processing. |

### Consumer Options

Passed as variadic arguments to consumer setup functions (e.g., `EventStreamConsumer`, `StreamConsumer`).

| Function | Description |
|----------|-------------|
| `AddQueueNameSuffix(suffix)` | Append a suffix to the queue name. Useful when multiple consumers in the same service subscribe to the same routing key. |
| `DisableSingleActiveConsumer()` | Allow multiple active consumers on the queue (default is single active consumer). |
| `WithDeadLetter(exchange)` | Route rejected/expired messages to the named dead letter exchange. |
| `WithDeadLetterRoutingKey(key)` | Set a custom routing key for dead-lettered messages. |

### Publisher Options

Passed to `NewPublisher()`.

| Function | Description |
|----------|-------------|
| `WithConfirm(ch)` | Enable publisher confirms with a custom confirmation channel. |
| `WithoutPublisherConfirms()` | Disable publisher confirms (enabled by default). Use for high-throughput scenarios where occasional message loss is acceptable. |

## Observability

### Tracing

OpenTelemetry spans are created for every publish and consume operation with semantic convention attributes (`messaging.system`, `messaging.operation`, `messaging.destination.name`, `messaging.rabbitmq.routing_key`). Trace context propagates through AMQP message headers.

```go
import (
    "go.opentelemetry.io/otel/propagation"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
)

conn.Start(ctx,
    amqp.WithTracing(tp),
    amqp.WithPropagator(propagation.TraceContext{}),
    // ... publishers and consumers
)
```

### Metrics

Register Prometheus metrics once at startup with `InitMetrics`.

```go
import "github.com/prometheus/client_golang/prometheus"

err := amqp.InitMetrics(prometheus.DefaultRegisterer)
```

Use `WithRoutingKeyMapper` to normalize dynamic routing key segments and prevent unbounded label cardinality:

```go
amqp.InitMetrics(prometheus.DefaultRegisterer, amqp.WithRoutingKeyMapper(func(key string) string {
    // Redact UUIDs from routing keys
    return uuidRegex.ReplaceAllString(key, "<id>")
}))
```

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `amqp_events_received` | counter | `queue`, `routing_key` | Events received |
| `amqp_events_ack` | counter | `queue`, `routing_key` | Events acknowledged |
| `amqp_events_nack` | counter | `queue`, `routing_key` | Events rejected |
| `amqp_events_without_handler` | counter | `queue`, `routing_key` | Events with no registered handler |
| `amqp_events_not_parsable` | counter | `queue`, `routing_key` | Events that failed JSON parsing |
| `amqp_events_processed_duration` | histogram | `queue`, `routing_key`, `result` | Processing time (ms) |
| `amqp_events_publish_succeed` | counter | `exchange`, `routing_key` | Successful publishes |
| `amqp_events_publish_failed` | counter | `exchange`, `routing_key` | Failed publishes |
| `amqp_events_publish_duration` | histogram | `exchange`, `routing_key`, `result` | Publish time (ms) |

### Notifications

Monitor event processing outcomes through notification channels.

```go
notifyCh := make(chan spec.Notification, 100)
errorCh := make(chan spec.ErrorNotification, 100)

conn.Start(ctx,
    amqp.WithNotificationChannel(notifyCh),
    amqp.WithErrorChannel(errorCh),
    // ... publishers and consumers
)

go func() {
    for n := range notifyCh {
        log.Printf("processed %s in %dms", n.DeliveryInfo.Key, n.Duration)
    }
}()

go func() {
    for e := range errorCh {
        log.Printf("failed %s: %v", e.DeliveryInfo.Key, e.Error)
    }
}()
```

## Connection Lifecycle

### Close Listener

Monitor unexpected disconnects using `CloseListener`. The channel receives errors from both AMQP channel and connection close events.

```go
closeCh := make(chan error, 1)

conn.Start(ctx,
    amqp.CloseListener(closeCh),
    // ... publishers and consumers
)

go func() {
    if err := <-closeCh; err != nil {
        log.Printf("AMQP disconnected: %v", err)
        os.Exit(1)
    }
}()
```

### Graceful Shutdown

```go
conn.Close()
```

## Topology Export

After `Start()`, the connection exposes the declared topology for validation and visualization using the spec module.

```go
topology := conn.Topology()
// topology is a spec.Topology with Transport, ServiceName, and Endpoints

errors := spec.Validate(topology)
diagram := spec.Mermaid([]spec.Topology{topology})
```

## Development

```sh
# Start RabbitMQ
docker compose up -d

# Run tests
go test ./...

# Lint
go vet ./...
```

## TCK Adapter

The `cmd/tck-adapter/` directory contains a Technology Compatibility Kit adapter that proves this transport conforms to the gomessaging specification. It implements the [JSON-RPC subprocess protocol](https://github.com/sparetimecoders/messaging) used by the TCK runner.

```sh
go build -o tck-adapter ./cmd/tck-adapter/
```

## License

MIT
