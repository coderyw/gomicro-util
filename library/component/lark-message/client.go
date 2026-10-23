// Package lark_message
// @Author: yinwei
// @File: client
// @Version: 1.0.0
// @Date: 2025/6/4 09:22

package lark_message

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/coderyw/gomicro-util/library/http/http_req"
	"github.com/coderyw/gomicro-util/library/http/net_http"
	srv_config "github.com/coderyw/gomicro-util/model/config"
)

// SendLarkMessage 发送 lark 消息
//
// Parameters:
//   - ctx context.Context 上下文
//   - groupId string 群组 id
//   - msg LarkMessageBuilder 消息builder
//
// Returns：
//   - error 发送错误信息
func SendLarkMessage(ctx context.Context, groupId string, msg LarkMessageBuilder) error {
	req := msg.Build()
	tm := strconv.FormatInt(time.Now().Unix(), 10)
	// 获取配置
	all := srv_config.LarkConfig.Groups
	if all == nil {
		return errors.Errorf("larkCfg is nil")
	}
	cfg, ok := all[groupId]
	if !ok {
		return errors.Errorf("larkCfg is nil: %v", groupId)
	}
	if cfg.Enabled != nil && *cfg.Enabled == false {
		return nil
	}
	sign, err := GenSign(cfg.Sign, tm)
	if err != nil {
		return errors.Wrap(err, "error to gen sign")
	}
	req.Sign.Sign = sign
	req.Sign.Timestamp = tm
	httpReq := http_req.NewRequest[*LarkMessageResp](req)
	resp, _, err := net_http.Post(ctx, cfg.WebHook, *httpReq)
	if err != nil {
		return errors.Wrap(err, "error to send lark message")
	}
	if resp.Code != 0 {
		return errors.Errorf("error to send lark message: %v", resp.Msg)
	}
	return nil
}

func GenSign(secret string, timestamp string) (string, error) {
	//timestamp + key 做sha256, 再进行base64 encode
	stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + secret

	var data []byte
	h := hmac.New(sha256.New, []byte(stringToSign))
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}

	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}
