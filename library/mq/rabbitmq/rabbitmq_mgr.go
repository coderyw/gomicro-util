package rabbitmq

import "github.com/coderyw/gomicro-util/model/config"

type RabbitMQManager struct {
	cfg     config.RabbitMqCfgs
	clients map[string]*Client
}

func NewRabbitMQManager(serverName string, cfg config.RabbitMqCfgs) (RabbitMQManager, error) {
	cs := make(map[string]*Client)
	for instance, v := range cfg {
		rmc := FromRabbitMqCfg(v)
		//初始化客户端
		c, err := NewClient(rmc)
		if err != nil {
			return RabbitMQManager{}, err
		}
		cs[instance] = c
	}

	return RabbitMQManager{
		clients: cs,
		cfg:     cfg,
	}, nil
}
