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
	"testing"

	amqplib "github.com/rabbitmq/amqp091-go"
	"github.com/sparetimecoders/messaging/specification/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutboxRawPublisher(t *testing.T) {
	pub := NewPublisher()
	adapter := NewOutboxRawPublisher(pub)

	assert.NotNil(t, adapter)
	assert.Same(t, pub, adapter.publisher)
}

func TestOutboxRawPublisher_PublishRaw_NilChannel(t *testing.T) {
	pub := NewPublisher()
	adapter := NewOutboxRawPublisher(pub)

	err := adapter.PublishRaw(context.Background(), "test.key", []byte("payload"), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "publisher not initialized")
}

func TestOutboxRawPublisher_PublishRaw_Success(t *testing.T) {
	ch := NewMockAmqpChannel()
	pub := NewPublisher(WithoutPublisherConfirms())
	pub.channel = ch
	pub.exchange = "test-exchange"

	adapter := NewOutboxRawPublisher(pub)

	headers := map[string]string{
		"ce-id":     "event-123",
		"ce-source": "test-service",
		"custom":    "value",
	}

	err := adapter.PublishRaw(context.Background(), "test.key", []byte(`{"hello":"world"}`), headers)

	require.NoError(t, err)

	published := <-ch.Published
	assert.Equal(t, "test-exchange", published.exchange)
	assert.Equal(t, "test.key", published.key)
	assert.Equal(t, []byte(`{"hello":"world"}`), published.msg.Body)
	assert.Equal(t, contentType, published.msg.ContentType)
	assert.Equal(t, uint8(2), published.msg.DeliveryMode)

	// ce-* headers should be normalized to cloudEvents:* format
	assert.Equal(t, "event-123", published.msg.Headers[spec.AMQPCEHeaderKey("id")])
	assert.Equal(t, "test-service", published.msg.Headers[spec.AMQPCEHeaderKey("source")])
	// non-ce headers should pass through as-is
	assert.Equal(t, "value", published.msg.Headers["custom"])
}

func TestOutboxRawPublisher_PublishRaw_EmptyHeaders(t *testing.T) {
	ch := NewMockAmqpChannel()
	pub := NewPublisher(WithoutPublisherConfirms())
	pub.channel = ch
	pub.exchange = "test-exchange"

	adapter := NewOutboxRawPublisher(pub)

	err := adapter.PublishRaw(context.Background(), "test.key", []byte("data"), map[string]string{})

	require.NoError(t, err)

	published := <-ch.Published
	assert.Empty(t, published.msg.Headers)
}

func TestOutboxRawPublisher_PublishRaw_PublishError(t *testing.T) {
	ch := NewMockAmqpChannel()
	ch.publishFn = func(_ context.Context, _, _ string, _, _ bool, _ amqplib.Publishing) error {
		return fmt.Errorf("connection lost")
	}
	pub := NewPublisher(WithoutPublisherConfirms())
	pub.channel = ch
	pub.exchange = "my-exchange"

	adapter := NewOutboxRawPublisher(pub)

	err := adapter.PublishRaw(context.Background(), "rk", []byte("data"), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "amqp outbox: publish to my-exchange/rk")
	assert.Contains(t, err.Error(), "connection lost")
}

func TestOutboxRawPublisher_PublishRaw_WithConfirmAck(t *testing.T) {
	ch := NewMockAmqpChannel()
	confirmCh := make(chan amqplib.Confirmation, 1)
	pub := NewPublisher(WithoutPublisherConfirms())
	pub.channel = ch
	pub.exchange = "test-exchange"
	pub.confirmCh = confirmCh

	adapter := NewOutboxRawPublisher(pub)

	// Pre-load a successful confirmation
	go func() {
		<-ch.Published
		confirmCh <- amqplib.Confirmation{DeliveryTag: 1, Ack: true}
	}()

	err := adapter.PublishRaw(context.Background(), "rk", []byte("data"), nil)

	require.NoError(t, err)
}

func TestOutboxRawPublisher_PublishRaw_WithConfirmNack(t *testing.T) {
	ch := NewMockAmqpChannel()
	confirmCh := make(chan amqplib.Confirmation, 1)
	pub := NewPublisher(WithoutPublisherConfirms())
	pub.channel = ch
	pub.exchange = "test-exchange"
	pub.confirmCh = confirmCh

	adapter := NewOutboxRawPublisher(pub)

	go func() {
		<-ch.Published
		confirmCh <- amqplib.Confirmation{DeliveryTag: 1, Ack: false}
	}()

	err := adapter.PublishRaw(context.Background(), "rk", []byte("data"), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "broker nacked outbox publish")
}

func TestOutboxRawPublisher_PublishRaw_ContextCancelled(t *testing.T) {
	ch := NewMockAmqpChannel()
	confirmCh := make(chan amqplib.Confirmation) // unbuffered, will block
	pub := NewPublisher(WithoutPublisherConfirms())
	pub.channel = ch
	pub.exchange = "test-exchange"
	pub.confirmCh = confirmCh

	adapter := NewOutboxRawPublisher(pub)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-ch.Published
		cancel()
	}()

	err := adapter.PublishRaw(ctx, "rk", []byte("data"), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled waiting for outbox publish confirm")
}
