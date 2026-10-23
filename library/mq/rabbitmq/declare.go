package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// 交换机类型常量
const (
	ExchangeDirect  = "direct"
	ExchangeTopic   = "topic"
	ExchangeFanout  = "fanout"
	ExchangeHeaders = "headers"
)

// QueueOption 队列声明参数封装。
type QueueOption struct {
	Name       string     // 队列名，空则由服务端生成
	Durable    bool       // 持久化，重启后保留
	AutoDelete bool       // 无消费者时自动删除
	Exclusive  bool       // 仅当前连接可访问
	NoWait     bool       // 不等待服务端确认
	Args       amqp.Table // 扩展参数，如 x-message-ttl、x-max-length
}

// DefaultQueueOption 返回常用默认：持久化、非独占、非自动删除。
func DefaultQueueOption(name string) QueueOption {
	return QueueOption{
		Name:       name,
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
		Args:       nil,
	}
}

// ExchangeOption 交换机声明参数封装。
type ExchangeOption struct {
	Name       string     // 交换机名
	Kind       string     // direct | topic | fanout | headers
	Durable    bool       // 持久化
	AutoDelete bool       // 无绑定队列时自动删除
	Internal   bool       // 仅内部使用，不接收客户端发布
	NoWait     bool       // 不等待服务端确认
	Args       amqp.Table // 扩展参数
}

// DefaultExchangeOption 返回常用默认：持久化、非内部。
func DefaultExchangeOption(name, kind string) ExchangeOption {
	if kind == "" {
		kind = ExchangeDirect
	}
	return ExchangeOption{
		Name:       name,
		Kind:       kind,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
		Args:       nil,
	}
}

// QueueBindOption 队列与交换机绑定参数。
type QueueBindOption struct {
	NoWait bool       // 不等待服务端确认
	Args   amqp.Table // 扩展参数，如 topic 的匹配参数
}

// QueueBindItem 描述一条队列与交换机的绑定，用于 Topology.Binds。
type QueueBindItem struct {
	QueueName  string // 队列名
	RoutingKey string // 路由键
	Exchange   string // 交换机名
	BindOption *QueueBindOption
}

// EnsureQueue 在给定 Channel 上声明队列，不存在则创建，已存在则校验参数一致。
func EnsureQueue(ch *amqp.Channel, opt QueueOption) (amqp.Queue, error) {
	return ch.QueueDeclare(opt.Name, opt.Durable, opt.AutoDelete, opt.Exclusive, opt.NoWait, opt.Args)
}

// EnsureQueuePassive 被动声明：仅校验队列是否存在，不存在则报错。
func EnsureQueuePassive(ch *amqp.Channel, name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return ch.QueueDeclarePassive(name, durable, autoDelete, exclusive, noWait, args)
}

// EnsureExchange 在给定 Channel 上声明交换机，不存在则创建，已存在则校验参数一致。
func EnsureExchange(ch *amqp.Channel, opt ExchangeOption) error {
	return ch.ExchangeDeclare(opt.Name, opt.Kind, opt.Durable, opt.AutoDelete, opt.Internal, opt.NoWait, opt.Args)
}

// EnsureQueueBind 在给定 Channel 上绑定队列与交换机；绑定已存在则忽略。
func EnsureQueueBind(ch *amqp.Channel, queueName, routingKey, exchange string, opt *QueueBindOption) error {
	noWait := false
	args := make(amqp.Table)
	if opt != nil {
		noWait = opt.NoWait
		if opt.Args != nil {
			args = opt.Args
		}
	}
	return ch.QueueBind(queueName, routingKey, exchange, noWait, args)
}

// QueueUnbind 解绑队列与交换机的指定 routing key。
func QueueUnbind(ch *amqp.Channel, queueName, routingKey, exchange string, args amqp.Table) error {
	return ch.QueueUnbind(queueName, routingKey, exchange, args)
}

// QueuePurge 清空队列中未被 ack 的消息，返回清除条数。
func QueuePurge(ch *amqp.Channel, name string, noWait bool) (int, error) {
	return ch.QueuePurge(name, noWait)
}

// QueueDelete 删除队列。ifUnused/ifEmpty 为 true 时在有用户或消息情况下不删除并报错。
func QueueDelete(ch *amqp.Channel, name string, ifUnused, ifEmpty, noWait bool) (int, error) {
	return ch.QueueDelete(name, ifUnused, ifEmpty, noWait)
}

// ExchangeDelete 删除交换机。ifUnused 为 true 时若仍有绑定则不删除并报错。
func ExchangeDelete(ch *amqp.Channel, name string, ifUnused, noWait bool) error {
	return ch.ExchangeDelete(name, ifUnused, noWait)
}

// Topology 描述一组需要声明的交换机、队列及绑定，便于一次性 Ensure。
type Topology struct {
	Exchanges []ExchangeOption
	Queues    []QueueOption
	Binds     []QueueBindItem
}

// EnsureTopology 在给定 Channel 上按顺序声明交换机、队列并建立绑定。
func EnsureTopology(ch *amqp.Channel, t Topology) error {
	for _, opt := range t.Exchanges {
		if err := EnsureExchange(ch, opt); err != nil {
			return err
		}
	}
	for _, opt := range t.Queues {
		if _, err := EnsureQueue(ch, opt); err != nil {
			return err
		}
	}
	for _, b := range t.Binds {
		if err := EnsureQueueBind(ch, b.QueueName, b.RoutingKey, b.Exchange, b.BindOption); err != nil {
			return err
		}
	}
	return nil
}

// EnsureTopologyWithContext 从 Client 取 Channel 并执行 EnsureTopology，用完后归还。
func (c *Client) EnsureTopologyWithContext(ctx context.Context, t Topology) error {
	ch, release, err := c.GetChannel(ctx)
	if err != nil {
		return err
	}
	defer release()
	return EnsureTopology(ch, t)
}

// EnsureQueueWithContext 从 Client 取 Channel 并声明队列，用完后归还。
func (c *Client) EnsureQueueWithContext(ctx context.Context, opt QueueOption) (amqp.Queue, error) {
	ch, release, err := c.GetChannel(ctx)
	if err != nil {
		return amqp.Queue{}, err
	}
	defer release()
	return EnsureQueue(ch, opt)
}

// EnsureExchangeWithContext 从 Client 取 Channel 并声明交换机，用完后归还。
func (c *Client) EnsureExchangeWithContext(ctx context.Context, opt ExchangeOption) error {
	ch, release, err := c.GetChannel(ctx)
	if err != nil {
		return err
	}
	defer release()
	return EnsureExchange(ch, opt)
}

// EnsureQueueBindWithContext 从 Client 取 Channel 并绑定队列与交换机，用完后归还。
func (c *Client) EnsureQueueBindWithContext(ctx context.Context, queueName, routingKey, exchange string, bindOpt *QueueBindOption) error {
	ch, release, err := c.GetChannel(ctx)
	if err != nil {
		return err
	}
	defer release()
	return EnsureQueueBind(ch, queueName, routingKey, exchange, bindOpt)
}
