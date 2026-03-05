# go-messaging-amqp Documentation

Detailed guides for the AMQP/RabbitMQ transport. For a quick overview, see the [main README](../README.md).

| Guide | Description |
|-------|-------------|
| [Connection Lifecycle](connection.md) | Creating, starting, monitoring, and closing connections |
| [Consumers](consumers.md) | Durable and transient consumers, error handling, dead letters |
| [Publishers](publishers.md) | Publishing messages, publisher confirms, custom headers |
| [Request-Response](request-response.md) | Synchronous RPC between services |
| [Observability](observability.md) | OpenTelemetry tracing, Prometheus metrics, notifications |
| [Topology & Naming](topology.md) | How patterns map to AMQP primitives, queue arguments |

## Related

- [gomessaging specification](https://github.com/sparetimecoders/messaging) — shared spec, TCK, validation, visualization
- [gomessaging/nats](https://github.com/sparetimecoders/go-messaging-nats) — NATS/JetStream transport
