# Publishers

Publishers send messages to AMQP exchanges. Each publisher is created independently and wired to an exchange during `Start()`.

## Creating a Publisher

```go
pub := amqp.NewPublisher()
```

A publisher can only be used after it has been wired to an exchange via a setup function.

## Wiring to Exchanges

### Event Stream Publisher

Publishes to the default `events.topic.exchange`:

```go
conn.Start(ctx,
    amqp.EventStreamPublisher(pub),
)
```

### Custom Stream Publisher

Publishes to a named topic exchange:

```go
conn.Start(ctx,
    amqp.StreamPublisher("audit", pub),
)
```

### Service Request Publisher

Publishes to a service's request exchange (`{service}.direct.exchange.request`):

```go
conn.Start(ctx,
    amqp.ServicePublisher("billing-service", pub),
)
```

### Queue Publisher

Publishes directly to a named queue (default exchange):

```go
conn.Start(ctx,
    amqp.QueuePublisher(pub, "email-tasks"),
)
```

## Publishing Messages

```go
err := pub.Publish(ctx, "Order.Created", OrderCreated{
    OrderID: "abc-123",
    Amount:  42,
})
```

| Parameter | Description |
|-----------|-------------|
| `ctx` | Context for tracing and cancellation |
| `routingKey` | Routing key for topic exchange matching |
| `msg` | Payload — must be JSON-serializable |

### What Publish Does

1. Serializes `msg` to JSON
2. Generates a UUID message ID
3. Sets CloudEvents headers (`cloudEvents:specversion`, `cloudEvents:type`, `cloudEvents:source`, `cloudEvents:id`, `cloudEvents:time`, `cloudEvents:datacontenttype`)
4. Injects trace context into AMQP headers
5. Publishes with delivery mode = persistent
6. Waits for publisher confirm (unless disabled)

### Custom Headers

Append headers to any published message:

```go
pub.Publish(ctx, "Order.Created", payload,
    amqp.Header{Key: "priority", Value: "high"},
    amqp.Header{Key: "region", Value: "eu-west-1"},
)
```

The reserved header key `"service"` is used internally for request-response routing — do not use it.

## Publisher Confirms

By default, `Publish()` blocks until the broker confirms the message was persisted. This guarantees at-least-once delivery at the cost of latency.

### Disable Confirms

For high-throughput scenarios where occasional message loss is acceptable:

```go
pub := amqp.NewPublisher(amqp.WithoutPublisherConfirms())
```

### Custom Confirm Channel

Monitor confirms asynchronously:

```go
confirmCh := make(chan amqplib.Confirmation, 100)
pub := amqp.NewPublisher(amqp.WithConfirm(confirmCh))

go func() {
    for c := range confirmCh {
        if !c.Ack {
            log.Printf("message %d was not confirmed", c.DeliveryTag)
        }
    }
}()
```

## Per-Publisher Channels

Each publisher gets its own AMQP channel. This means:
- Publishers are safe to use from multiple goroutines
- One slow publisher doesn't block others
- Channel-level errors are isolated

## Wire Format

Every published message has these AMQP properties:

| Property | Value |
|----------|-------|
| `delivery_mode` | 2 (persistent) |
| `content_type` | `application/json` |
| `message_id` | UUID v4 |
| `timestamp` | Current UTC time |

And these application headers (CloudEvents binary content mode):

| Header | Value |
|--------|-------|
| `cloudEvents:specversion` | `1.0` |
| `cloudEvents:type` | routing key |
| `cloudEvents:source` | exchange name |
| `cloudEvents:id` | message ID |
| `cloudEvents:time` | RFC 3339 UTC |
| `cloudEvents:datacontenttype` | `application/json` |
