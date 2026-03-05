# Topology & Naming

This transport maps gomessaging's five messaging patterns to AMQP 0-9-1 primitives (exchanges, queues, bindings) using deterministic naming conventions from the [specification](https://github.com/sparetimecoders/messaging).

## Pattern-to-AMQP Mapping

### Event Stream

```
Publisher ──> [events.topic.exchange] ──routing key──> [events.topic.exchange.queue.{service}]
```

| Resource | Name | Type |
|----------|------|------|
| Exchange | `events.topic.exchange` | topic |
| Queue | `events.topic.exchange.queue.{service}` | quorum |
| Binding | routing key (e.g., `Order.Created`) | — |

### Custom Stream

Same structure as event stream but with a named exchange:

| Resource | Name | Type |
|----------|------|------|
| Exchange | `{name}.topic.exchange` | topic |
| Queue | `{name}.topic.exchange.queue.{service}` | quorum |

### Service Request

```
Caller ──> [billing.direct.exchange.request] ──> [billing.direct.exchange.request.queue]
```

| Resource | Name | Type |
|----------|------|------|
| Exchange | `{service}.direct.exchange.request` | direct |
| Queue | `{service}.direct.exchange.request.queue` | quorum |

### Service Response

```
Handler ──> [billing.headers.exchange.response] ──header match──> [billing.headers.exchange.response.queue.{caller}]
```

| Resource | Name | Type |
|----------|------|------|
| Exchange | `{service}.headers.exchange.response` | headers |
| Queue | `{service}.headers.exchange.response.queue.{caller}` | quorum |

Routing uses `x-match: any` with a `service` header to route responses back to the correct caller.

### Queue Publish

```
Publisher ──> [default exchange] ──> [{queueName}]
```

Publishes directly to a named queue via the AMQP default exchange (routing key = queue name).

## Queue Arguments

### Durable Queues

All durable queues are created with:

```json
{
  "x-queue-type": "quorum",
  "x-single-active-consumer": true,
  "x-message-ttl": 432000000
}
```

| Argument | Value | Purpose |
|----------|-------|---------|
| `x-queue-type` | `quorum` | Replicated across brokers, survives restarts |
| `x-single-active-consumer` | `true` | Only one consumer active at a time (ordered processing) |
| `x-message-ttl` | 432,000,000 ms (5 days) | Auto-cleanup of unconsumed messages |

### Transient Queues

Transient (ephemeral) consumers use classic queues:

```json
{
  "x-message-ttl": 1000
}
```

- Auto-delete: enabled (queue removed when consumer disconnects)
- Queue name: `{base-name}-{uuid}` (unique per instance)
- TTL: 1 second

## Exchange Declarations

All exchanges are declared as:
- **Durable**: survives broker restart
- **Not auto-delete**: persists after last binding is removed
- **Not internal**: accepts publishes from clients

## Topology Export

### From a Running Connection

After `Start()`, the connection exposes its declared topology:

```go
topology := conn.Topology()
// spec.Topology{Transport: "amqp", ServiceName: "order-service", Endpoints: [...]}
```

### Without Connecting

Use `CollectTopology` to inspect topology statically (no broker connection needed):

```go
topology, err := amqp.CollectTopology("order-service",
    amqp.EventStreamPublisher(pub),
    amqp.EventStreamConsumer("Order.Created", handler),
    amqp.StreamConsumer("audit", "User.Login", auditHandler),
)
```

This runs setup functions in a dry-run mode to collect endpoint declarations without connecting to a broker.

### Validation and Visualization

Use the spec module to validate and visualize exported topologies:

```go
import "github.com/sparetimecoders/messaging/specification/spec"

// Validate one service
errors := spec.Validate(topology)

// Cross-validate multiple services
errors := spec.ValidateTopologies([]spec.Topology{orders, billing, notifications})

// Generate Mermaid diagram
diagram := spec.Mermaid([]spec.Topology{orders, billing, notifications})
```

See the [spec documentation](https://github.com/sparetimecoders/messaging/blob/main/docs/topology.md) for details.

## Inspecting Topology in RabbitMQ Management

All gomessaging resources follow predictable naming. In the RabbitMQ Management UI:

**Exchanges tab:**
- `events.topic.exchange` — default event stream
- `{name}.topic.exchange` — custom streams
- `{service}.direct.exchange.request` — request exchanges
- `{service}.headers.exchange.response` — response exchanges

**Queues tab:**
- `events.topic.exchange.queue.{service}` — event consumers
- `{service}.direct.exchange.request.queue` — request handlers
- `{service}.headers.exchange.response.queue.{caller}` — response consumers
- `*-{uuid}` suffix — transient consumers
