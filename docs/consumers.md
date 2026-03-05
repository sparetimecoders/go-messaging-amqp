# Consumers

All consumers are registered as `Setup` functions passed to `conn.Start()`. Each consumer setup declares an exchange, queue, and binding, then registers a handler function that is called for each matching message.

## Consumer Types

### Durable Event Stream Consumer

Subscribes to the default `events.topic.exchange`. The queue survives restarts and uses quorum replication.

```go
amqp.EventStreamConsumer("Order.Created",
    func(ctx context.Context, e spec.ConsumableEvent[OrderCreated]) error {
        return processOrder(e.Payload)
    })
```

### Transient Event Stream Consumer

Same exchange, but the queue is auto-deleted when the consumer disconnects. Useful for temporary subscriptions, live dashboards, or debugging.

```go
amqp.TransientEventStreamConsumer("Order.*",
    func(ctx context.Context, e spec.ConsumableEvent[any]) error {
        log.Println("event:", e.DeliveryInfo.Key)
        return nil
    })
```

Transient queues get a UUID suffix to avoid name collisions and have a 1-second TTL.

### Custom Stream Consumer

Subscribe to a named exchange instead of the default `events`:

```go
amqp.StreamConsumer("audit", "User.Login", handler)
amqp.TransientStreamConsumer("audit", "User.*", handler)
```

### Service Request/Response Consumers

See [Request-Response](request-response.md).

## Handler Contract

```go
type EventHandler[T any] func(ctx context.Context, event spec.ConsumableEvent[T]) error
```

The `ConsumableEvent[T]` contains:

| Field | Description |
|-------|-------------|
| `e.Payload` | Deserialized message body (type `T`) |
| `e.ID` | CloudEvents message ID |
| `e.Type` | Event type (routing key) |
| `e.Source` | Publishing service name |
| `e.Timestamp` | When the event was produced |
| `e.DeliveryInfo.Destination` | Queue name |
| `e.DeliveryInfo.Source` | Exchange name |
| `e.DeliveryInfo.Key` | Routing key |
| `e.DeliveryInfo.Headers` | All message headers |

### Acknowledgment

| Handler returns | Broker action |
|----------------|---------------|
| `nil` | Message **ACK** — removed from queue |
| non-nil `error` | Message **NACK with requeue** — returned to queue for retry |

### Deserialization Failures

If the message body cannot be unmarshalled to type `T`, the message is **rejected without requeue** (permanent failure). This prevents poison messages from blocking the queue.

## Consumer Options

Options are passed as variadic arguments after the handler:

```go
amqp.EventStreamConsumer("Order.Created", handler,
    amqp.WithDeadLetter("dlx"),
    amqp.DisableSingleActiveConsumer(),
)
```

### Queue Name Suffix

```go
amqp.AddQueueNameSuffix("priority")
```

Appends a suffix to the queue name. Use when the same service needs multiple consumers for the same routing key with different handlers (e.g., priority vs. batch processing).

Result: `events.topic.exchange.queue.order-service-priority`

### Single Active Consumer

By default, quorum queues enable [single active consumer](https://www.rabbitmq.com/docs/consumers#single-active-consumer) — only one consumer in a consumer group processes messages at a time. This guarantees ordering within the queue.

```go
amqp.DisableSingleActiveConsumer()
```

Disable for parallel processing across replicas when ordering is not required.

### Dead Letter Exchange

Route rejected or expired messages to a dead letter exchange for inspection:

```go
amqp.WithDeadLetter("my-dlx")
amqp.WithDeadLetterRoutingKey("failed.orders")
```

Messages that are:
- **NACKed without requeue** (deserialization failure)
- **Expired** (TTL exceeded)

...are routed to the specified dead letter exchange. You must declare the DLX and its bindings separately.

## Wildcard Routing

AMQP topic exchanges support wildcard routing keys:

| Pattern | Matches |
|---------|---------|
| `Order.Created` | Exactly `Order.Created` |
| `Order.*` | `Order.Created`, `Order.Updated` (one segment) |
| `Order.#` | `Order.Created`, `Order.Item.Added` (any depth) |

```go
amqp.EventStreamConsumer("Order.*", handler)  // all Order events (one level)
amqp.EventStreamConsumer("#", handler)         // everything
```

## Queue Defaults

All durable queues are created with these arguments:

| Argument | Value | Reason |
|----------|-------|--------|
| `x-queue-type` | `quorum` | Replicated, survives broker restarts |
| `x-single-active-consumer` | `true` | Ordered processing (can be disabled) |
| `x-message-ttl` | 432,000,000 ms (5 days) | Auto-cleanup of stale messages |

Transient queues use classic queues with auto-delete and 1-second TTL.

## Dynamic Type Mapping

For consumers that handle multiple message types on the same routing pattern:

```go
amqp.TypeMappingHandler(genericHandler, func(routingKey string) (reflect.Type, bool) {
    switch routingKey {
    case "Order.Created":
        return reflect.TypeOf(OrderCreated{}), true
    case "Order.Updated":
        return reflect.TypeOf(OrderUpdated{}), true
    default:
        return nil, false
    }
})
```

This deserializes messages to different types based on the routing key. If the mapper returns `false`, the message is rejected without requeue.
