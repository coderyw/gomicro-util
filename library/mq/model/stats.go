package model

type ProducerStats struct {
	Instance    string
	PushCount   int64
	AvaPushTime string
	Topic       string
}

const MQ_HEADER_TRACEID = "trace_id"
