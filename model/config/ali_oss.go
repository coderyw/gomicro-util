/**
 * @Author: yinwei
 * @Description:
 * @File: ali_oss
 * @Version: 1.0.0
 * @Date: 2023/10/28 15:12
 */
package config

// AliOssConfig 阿里oss配置
type AliOssConfig struct {
	AccessId  string `json:"access_id"`
	AccessKey string `json:"access_key"`
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
}
