/**
 * @Author: yinwei
 * @Description:
 * @File: beanstalk
 * @Version: 1.0.0
 * @Date: 2023/10/31 17:54
 */
package config

type BeanstalkCfg struct {
	Hosts       []string `json:"hosts"`        //主机地址
	ConsumerNum int      `json:"consumer_num"` //启动消费者数量 每个tube启动的消费者数量
}
