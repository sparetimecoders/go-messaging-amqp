// MIT License
//
// Copyright (c) 2026 sparetimecoders
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package amqp

import (
	"context"
	"fmt"
	"strings"

	amqplib "github.com/rabbitmq/amqp091-go"
	"github.com/sparetimecoders/messaging/specification/spec"
)

// OutboxRawPublisher wraps an AMQP Publisher to satisfy the outbox.RawPublisher
// interface. Headers are received as ce-* prefixed keys and normalized to the
// AMQP cloudEvents:* wire format before publishing.
type OutboxRawPublisher struct {
	publisher *Publisher
}

// NewOutboxRawPublisher creates an adapter that wraps publisher for use with
// the outbox relay. The publisher must already be wired to a connection.
func NewOutboxRawPublisher(publisher *Publisher) *OutboxRawPublisher {
	return &OutboxRawPublisher{publisher: publisher}
}

// PublishRaw publishes a pre-serialized message to AMQP with the given headers.
// ce-* headers are normalized to cloudEvents:* per the AMQP binding spec.
func (a *OutboxRawPublisher) PublishRaw(ctx context.Context, routingKey string, payload []byte, headers map[string]string) error {
	if a.publisher.channel == nil {
		return fmt.Errorf("amqp outbox: publisher not initialized")
	}

	table := amqplib.Table{}

	// Normalize ce-* headers to cloudEvents:* for AMQP wire format
	for k, v := range headers {
		if strings.HasPrefix(k, "ce-") {
			amqpKey := spec.AMQPCEHeaderKey(strings.TrimPrefix(k, "ce-"))
			table[amqpKey] = v
		} else {
			table[k] = v
		}
	}

	publishing := amqplib.Publishing{
		Body:         payload,
		ContentType:  contentType,
		DeliveryMode: 2,
		Headers:      table,
	}

	err := a.publisher.channel.PublishWithContext(ctx, a.publisher.exchange, routingKey, false, false, publishing)
	if err != nil {
		return fmt.Errorf("amqp outbox: publish to %s/%s: %w", a.publisher.exchange, routingKey, err)
	}

	if a.publisher.confirmCh != nil {
		select {
		case confirm := <-a.publisher.confirmCh:
			if !confirm.Ack {
				return fmt.Errorf("broker nacked outbox publish to %s/%s", a.publisher.exchange, routingKey)
			}
		case <-ctx.Done():
			return fmt.Errorf("context cancelled waiting for outbox publish confirm: %w", ctx.Err())
		}
	}

	return nil
}
