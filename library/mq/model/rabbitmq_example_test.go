package model

import (
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ExampleRabbitMqPublishing 展示如何使用 RabbitMQ 推送消息。
func ExampleRabbitMqPublishing() {
	_ = RabbitMqPublishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		MessageId:    uuid.New().String(),
		AppId:        "order-service",
		Type:         "order.created",
		Headers: amqp.Table{
			"tenant_id": "tenantID",
			"trace_id":  "traceID",
			"retry":     0,
		},
		Body: []byte{},
	}
	// Output:
}
