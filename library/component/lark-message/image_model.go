// Package lark_message
// @Author: yinwei
// @File: image_model
// @Version: 1.0.0
// @Date: 2025/6/4 11:03

package lark_message

// 发送图片

type LarkMessageImageReqBuilder struct {
	ImageKey string `json:"image_key"`
}

func (b *LarkMessageImageReqBuilder) Build() LarkMessageReq {
	req := LarkMessageReq{
		MsgType: "image",
	}
	req.Content.ImageKey = b.ImageKey
	return req
}
