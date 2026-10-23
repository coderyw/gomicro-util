/**
 * @Author: yinwei
 * @Description:
 * @File: yaml
 * @Version: 1.0.0
 * @Date: 2023/10/25 18:20
 */
package yaml

import (
	"gopkg.in/yaml.v3"
	"os"
)

func YamlLoad(path string, obj interface{}) error {

	//// 设置配置文件的名字
	//viper.SetConfigName("test")
	//// 设置配置文件的类型
	//viper.SetConfigType("yaml")
	//// 添加配置文件的路径，指定 config 目录下寻找
	//viper.AddConfigPath("./config")

	file, err := os.ReadFile(path)
	if err != nil {

		return err
	}

	err = yaml.Unmarshal(file, obj)
	return err
}
