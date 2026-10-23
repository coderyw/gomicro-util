/**
 * @Author: yinwei
 * @Description:
 * @File: ali_sms
 * @Version: 1.0.0
 * @Date: 2023/10/25 15:52
 */
package ali_sms

import (
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v3/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/coderyw/gomicro-util/library/site-var/etcd"
	config2 "github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/cst"
)

func createClient(accessKeyId *string, accessKeySecret *string) (_result *dysmsapi20170525.Client, _err error) {
	config := &openapi.Config{
		// 必填，您的 AccessKey ID
		AccessKeyId: accessKeyId,
		// 必填，您的 AccessKey Secret
		AccessKeySecret: accessKeySecret,
	}
	// Endpoint 请参考 https://api.aliyun.com/product/Dysmsapi
	config.Endpoint = tea.String("dysmsapi.aliyuncs.com")
	_result = &dysmsapi20170525.Client{}
	_result, _err = dysmsapi20170525.NewClient(config)
	return _result, _err
}

// 发送阿里短信验证码
func SendAliVerCodeSms(phone, verCode string) error {
	cfg := new(config2.AliSmsCfg)
	err := etcd.GetEtcdCfgData(etcd.GetConfigKey(cst.ETCD_ALI_SMS_VER_KEY), cfg)
	if err != nil {
		return err
	}
	client, err := createClient(tea.String(cfg.AccessKey), tea.String(cfg.AccessSecret))
	if err != nil {
		return err
	}

	sendSmsRequest := &dysmsapi20170525.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String("linkysw"),
		TemplateCode:  tea.String("SMS_268605666"),
		TemplateParam: tea.String(fmt.Sprintf("{\"code\":\"%v\"}", verCode)),
	}
	runtime := &util.RuntimeOptions{}
	tryErr := func() (_e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				_e = r
			}
		}()
		// 复制代码运行请自行打印 API 的返回值
		resp, err := client.SendSmsWithOptions(sendSmsRequest, runtime)
		if err != nil {
			return err
		}
		if resp.Body != nil {
			if tea.ToString(resp.Body.Code) != "OK" {
				return fmt.Errorf(tea.ToString(resp.Body.Message))
			}
		}

		return nil
	}()

	if tryErr != nil {
		var error = &tea.SDKError{}
		if _t, ok := tryErr.(*tea.SDKError); ok {
			error = _t
		} else {
			error.Message = tea.String(tryErr.Error())
		}
		// 如有需要，请打印 error
		_, err = util.AssertAsString(error.Message)
		if err != nil {
			return err
		}
	}
	return err
}
