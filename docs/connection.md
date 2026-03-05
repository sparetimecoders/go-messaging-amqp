# Connection Lifecycle

## Creating a Connection

A connection requires a **service name** and an **AMQP URL**. The service name drives all exchange and queue naming — it must be consistent across deployments of the same service.

```go
conn, err := amqp.NewFromURL("order-service", "amqp://guest:guest@localhost:5672/")
if err != nil {
    log.Fatal(err)
}
```

The URL is parsed into an `amqp.URI` (accessible via `conn.URI()`). The connection is not yet established — that happens in `Start()`.

## Starting

`Start()` connects to the broker, declares all topology (exchanges, queues, bindings), and starts consumers. Setup functions are executed in order.

```go
err := conn.Start(ctx,
    amqp.WithLogger(logger),
    amqp.WithTracing(tp),
    amqp.EventStreamPublisher(pub),
    amqp.EventStreamConsumer("Order.Created", handler),
)
```

`Start()` can only be called once. Calling it again returns `ErrAlreadyStarted`.

### What Start Does

1. Dials the AMQP broker with a heartbeat of 10s
2. Sets the client connection name to `{serviceName}#{version}#@{hostname}` (visible in RabbitMQ Management)
3. Creates a setup channel for declaring topology
4. Runs each `Setup` function — these declare exchanges, queues, bindings, and register consumer handlers
5. Closes the setup channel
6. Creates one channel per consumer queue, with the configured prefetch limit
7. Starts all consumer goroutines

## Configuration

All configuration is passed as `Setup` functions to `Start()`. Order matters only when setups depend on each other (e.g., `WithLogger` should come before setups that log).

### Logger

```go
amqp.WithLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
```

Default: `slog.Default()`. The logger is enriched with `"service"` attribute automatically.

### Prefetch

```go
amqp.WithPrefetchLimit(1)    // strict round-robin
amqp.WithPrefetchLimit(100)  // high throughput
```

Default: 20. Controls how many unacknowledged messages each consumer channel can hold. Lower values give fairer distribution across consumers; higher values improve throughput.

### Legacy Support

```go
amqp.WithLegacySupport()
```

When enabled, incoming messages without CloudEvents headers are enriched with synthetic metadata (ID, timestamp, type from routing key, source from exchange). This allows gradual migration from systems that don't set CE headers.

## Monitoring Disconnects

Register a close listener to detect unexpected broker disconnects:

```go
closeCh := make(chan error, 1)

conn.Start(ctx,
    amqp.CloseListener(closeCh),
    // ...
)

go func() {
    if err := <-closeCh; err != nil {
        log.Printf("AMQP disconnected: %v", err)
        // Trigger reconnection, alert, or shutdown
    }
}()
```

The channel receives errors from both AMQP connection close events and channel close events. This is the primary mechanism for detecting broker failures.

## Graceful Shutdown

```go
err := conn.Close()
```

`Close()` closes the underlying AMQP connection, which implicitly closes all channels and stops all consumers. It is safe to call on a connection that was never started.

## Connection Properties

RabbitMQ Management shows each connection with a client-provided name:

```
order-service#v0.3.1#@ip-10-0-1-42
```

This includes the service name, module version (from Go build info), and hostname — useful for identifying which service instance owns a connection.
