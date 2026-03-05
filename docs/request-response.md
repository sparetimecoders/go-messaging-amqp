# Request-Response

The request-response pattern provides synchronous RPC between services over AMQP. A caller publishes a request to a service's direct exchange; the service processes it and publishes a response back via a headers exchange.

## Architecture

```
                     ┌─────────────────────────────────────────┐
order-service ─────> │ billing.direct.exchange.request         │ ──> billing-service
  (caller)           └─────────────────────────────────────────┘     (handler)
                                                                         │
                     ┌─────────────────────────────────────────┐         │
order-service <───── │ billing.headers.exchange.response        │ <───────┘
                     └─────────────────────────────────────────┘
```

The response exchange uses `headers` type, routing replies back to the correct caller based on a `service` header match.

## Handler Side

The service that handles requests uses `RequestResponseHandler`:

```go
type InvoiceRequest struct {
    OrderID string `json:"orderId"`
}

type InvoiceResponse struct {
    InvoiceID string `json:"invoiceId"`
    Total     int    `json:"total"`
}

billingConn.Start(ctx,
    amqp.RequestResponseHandler("Invoice.Create",
        func(ctx context.Context, e spec.ConsumableEvent[InvoiceRequest]) (InvoiceResponse, error) {
            invoice, err := createInvoice(e.Payload.OrderID)
            if err != nil {
                return InvoiceResponse{}, err
            }
            return InvoiceResponse{
                InvoiceID: invoice.ID,
                Total:     invoice.Total,
            }, nil
        }),
)
```

`RequestResponseHandler` combines:
1. A `ServiceRequestConsumer` — listens on `billing.direct.exchange.request`
2. Auto-publishing the return value to `billing.headers.exchange.response` with the caller's service name as the routing target

### Error Handling

| Handler returns | Behavior |
|----------------|----------|
| `(response, nil)` | Response published to caller, request ACKed |
| `(_, error)` | No response sent, request NACKed with requeue |

The caller must handle the case where no response arrives (timeout).

## Caller Side

The caller needs a publisher to send requests and a consumer to receive responses:

```go
pub := amqp.NewPublisher()

orderConn.Start(ctx,
    // Wire publisher to billing's request exchange
    amqp.ServicePublisher("billing-service", pub),

    // Listen for responses from billing
    amqp.ServiceResponseConsumer("billing-service", "Invoice.Create",
        func(ctx context.Context, e spec.ConsumableEvent[InvoiceResponse]) error {
            fmt.Printf("invoice %s, total %d\n", e.Payload.InvoiceID, e.Payload.Total)
            return nil
        }),
)

// Send request
pub.Publish(ctx, "Invoice.Create", InvoiceRequest{OrderID: "abc-123"})
```

## Low-Level Alternative

For more control, use `ServiceRequestConsumer` directly and publish responses manually:

```go
billingConn.Start(ctx,
    amqp.ServiceRequestConsumer("Invoice.Create",
        func(ctx context.Context, e spec.ConsumableEvent[InvoiceRequest]) error {
            invoice, err := createInvoice(e.Payload.OrderID)
            if err != nil {
                return err
            }
            // Manually publish response
            return billingConn.PublishServiceResponse(ctx,
                e.DeliveryInfo.Headers["service"].(string), // caller service name
                "Invoice.Create",
                InvoiceResponse{InvoiceID: invoice.ID},
            )
        }),
)
```

This is useful when you need conditional responses or want to respond to a different routing key.

## Topology Created

For a `billing-service` handling requests from `order-service`:

| Resource | Name | Type |
|----------|------|------|
| Request exchange | `billing-service.direct.exchange.request` | direct |
| Request queue | `billing-service.direct.exchange.request.queue` | quorum |
| Response exchange | `billing-service.headers.exchange.response` | headers |
| Response queue | `billing-service.headers.exchange.response.queue.order-service` | quorum |

The response queue is scoped to the caller — each calling service gets its own response queue.

## Extracting Response Types

Use `ServiceResponseConsumer` with a typed handler:

```go
amqp.ServiceResponseConsumer[InvoiceResponse]("billing-service", "Invoice.Create", handler)
```

The generic type parameter `InvoiceResponse` determines how the response body is deserialized.
