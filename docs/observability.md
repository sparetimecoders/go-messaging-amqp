# Observability

go-messaging-amqp provides three observability layers: distributed tracing (OpenTelemetry), metrics (Prometheus), and event notifications.

## Tracing

### Setup

```go
import (
    "go.opentelemetry.io/otel/propagation"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))

conn.Start(ctx,
    amqp.WithTracing(tp),
    amqp.WithPropagator(propagation.TraceContext{}),
    // ... publishers and consumers
)
```

Default: `otel.GetTracerProvider()` and `otel.GetTextMapPropagator()`.

### Span Attributes

Every publish and consume operation creates a span with these attributes:

| Attribute | Value |
|-----------|-------|
| `messaging.system` | `rabbitmq` |
| `messaging.operation` | `publish` or `receive` |
| `messaging.destination.name` | Exchange name |
| `messaging.destination.queue` | Queue name (consume only) |
| `messaging.rabbitmq.routing_key` | Routing key |
| `messaging.message.id` | Message UUID |
| `messaging.message.body.size` | Payload size in bytes |
| `messaging.client_id` | Service name |

### Trace Context Propagation

Trace context is automatically injected into AMQP headers on publish and extracted on consume. This links producer and consumer spans into a single distributed trace.

### Custom Span Names

```go
// Consume spans
amqp.WithSpanNameFn(func(d spec.DeliveryInfo) string {
    return fmt.Sprintf("consume %s from %s", d.Key, d.Source)
})

// Publish spans
amqp.WithPublishSpanNameFn(func(exchange, routingKey string) string {
    return fmt.Sprintf("publish %s to %s", routingKey, exchange)
})
```

Default span names use the standard messaging semantic conventions.

## Metrics

### Setup

Register metrics once at application startup:

```go
import "github.com/prometheus/client_golang/prometheus"

err := amqp.InitMetrics(prometheus.DefaultRegisterer)
```

All publishers and consumers on this process will record metrics after registration.

### Available Metrics

**Counters:**

| Metric | Labels | Description |
|--------|--------|-------------|
| `amqp_events_received` | `queue`, `routing_key` | Messages received by consumers |
| `amqp_events_ack` | `queue`, `routing_key` | Messages acknowledged |
| `amqp_events_nack` | `queue`, `routing_key` | Messages rejected (handler error) |
| `amqp_events_without_handler` | `queue`, `routing_key` | Messages with no registered handler |
| `amqp_events_not_parsable` | `queue`, `routing_key` | Messages that failed JSON deserialization |
| `amqp_events_publish_succeed` | `exchange`, `routing_key` | Successful publishes |
| `amqp_events_publish_failed` | `exchange`, `routing_key` | Failed publishes |

**Histograms:**

| Metric | Labels | Description |
|--------|--------|-------------|
| `amqp_events_processed_duration` | `queue`, `routing_key`, `result` | Consumer processing time (ms) |
| `amqp_events_publish_duration` | `exchange`, `routing_key`, `result` | Publish time including confirm (ms) |

The `result` label is `"success"` or `"error"`.

### Routing Key Cardinality

Routing keys with dynamic segments (UUIDs, timestamps) cause unbounded label cardinality. Use `WithRoutingKeyMapper` to normalize:

```go
import "regexp"

var uuidPattern = regexp.MustCompile(
    `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

amqp.InitMetrics(prometheus.DefaultRegisterer,
    amqp.WithRoutingKeyMapper(func(key string) string {
        return uuidPattern.ReplaceAllString(key, "<id>")
    }),
)
```

This maps `Order.abc-123-def-456` → `Order.<id>`, keeping cardinality bounded.

## Notifications

Notifications provide a lightweight alternative to metrics for monitoring event processing outcomes.

### Success Notifications

```go
notifyCh := make(chan spec.Notification, 100)

conn.Start(ctx,
    amqp.WithNotificationChannel(notifyCh),
    // ...
)

go func() {
    for n := range notifyCh {
        log.Printf("processed %s in %dms (queue: %s)",
            n.DeliveryInfo.Key, n.Duration, n.DeliveryInfo.Destination)
    }
}()
```

### Error Notifications

```go
errorCh := make(chan spec.ErrorNotification, 100)

conn.Start(ctx,
    amqp.WithErrorChannel(errorCh),
    // ...
)

go func() {
    for e := range errorCh {
        log.Printf("failed %s: %v (queue: %s)",
            e.DeliveryInfo.Key, e.Error, e.DeliveryInfo.Destination)
    }
}()
```

### Notification Fields

| Field | Type | Description |
|-------|------|-------------|
| `DeliveryInfo.Destination` | string | Queue name |
| `DeliveryInfo.Source` | string | Exchange name |
| `DeliveryInfo.Key` | string | Routing key |
| `Duration` | int64 | Processing time in milliseconds |
| `Source` | NotificationSource | Always `CONSUMER` |
| `Error` | error | Error from handler (ErrorNotification only) |

## Combining All Three

A production service typically uses all three:

```go
// Metrics — registered once
amqp.InitMetrics(prometheus.DefaultRegisterer,
    amqp.WithRoutingKeyMapper(normalizeKey))

// Tracing — per connection
conn.Start(ctx,
    amqp.WithTracing(tp),
    amqp.WithPropagator(propagation.TraceContext{}),

    // Notifications — for alerting
    amqp.WithNotificationChannel(notifyCh),
    amqp.WithErrorChannel(errorCh),

    // ... publishers and consumers
)
```

- **Tracing** for request-level debugging and cross-service correlation
- **Metrics** for dashboards, SLOs, and alerting thresholds
- **Notifications** for in-process event-driven reactions (circuit breakers, adaptive backoff)
