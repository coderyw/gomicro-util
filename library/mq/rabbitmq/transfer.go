package rabbitmq

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/coderyw/gomicro-util/library/mq/model"
	"github.com/coderyw/gomicro-util/library/trace"
)

func RabbitMqPublishing2AmqpPublishing(ctx context.Context, mqPublishing model.RabbitMqPublishing) amqp.Publishing {

	p := amqp.Publishing{
		Headers:         mqPublishing.Headers,
		ContentType:     mqPublishing.ContentType,
		ContentEncoding: mqPublishing.ContentEncoding,
		DeliveryMode:    mqPublishing.DeliveryMode,
		Priority:        mqPublishing.Priority,
		CorrelationId:   mqPublishing.CorrelationId,
		ReplyTo:         mqPublishing.ReplyTo,
		Expiration:      mqPublishing.Expiration,
		MessageId:       mqPublishing.MessageId,
		Timestamp:       time.Now(),
		Type:            mqPublishing.Type,
		UserId:          mqPublishing.UserId,
		AppId:           mqPublishing.AppId,
		Body:            mqPublishing.Body,
	}
	if p.ContentType == "" {
		p.ContentType = "application/json"
	}
	if p.DeliveryMode == 0 {
		p.DeliveryMode = amqp.Persistent
	}
	traceId, _ := trace.GetTraceId(ctx)
	if p.CorrelationId == "" {
		p.CorrelationId = traceId
	}
	if p.MessageId == "" {
		p.MessageId = traceId
	}
	if p.AppId == "" {
		// todo servername 赋值
	}
	if p.Headers == nil {
		p.Headers = make(amqp.Table)
	}
	// traceId 透传
	if _, ok := p.Headers[model.MQ_HEADER_TRACEID]; !ok {
		p.Headers[model.MQ_HEADER_TRACEID] = traceId
	}
	return p
}
