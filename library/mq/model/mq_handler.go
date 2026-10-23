package model

import (
	"context"
)

type MQHandler interface {
	Do(ctx context.Context, topic string, msg []byte) error
}

type Handler struct {
	InstanceName string
	GroupId      string
	HandlerFunc  HandlerFunc
	HandlerFuncs HandlerFuncs
	Topics       string
	BeforeStop   func()
}

type HandlerFunc func(ctx context.Context, msg KafkaConsumerMessage) error

type HandlerFuncs func(ctx context.Context, msgs []KafkaConsumerMessage) (int, error)
