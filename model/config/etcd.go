/**
 * @Author: yinwei
 * @Description: 配置相关配置结构
 * @File: etcd
 * @Version: 1.0.0
 * @Date: 2023/10/24 16:12
 */
package config

var DefaulConfig = EtcdConfig{}

// 默认server配置，配置必须是这个结构起始
type EtcdConfig struct {
	ConfigName string      `json:"config_name"` //配置名称
	Desc       string      `json:"desc"`        //配置描述
	Server     []string    `json:"server"`      //配置支持服务名，不填默认支持全部
	Data       interface{} `json:"data"`        //具体配置数据
}

/**
{
"config_name":"redis配置",
"desc":"",
"server":[],
"data":"{}"
}
*/
