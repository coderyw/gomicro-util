// Package lark_message
// @Author: yinwei
// @File: text_model
// @Version: 1.0.0
// @Date: 2025/6/4 11:01

package lark_message

type LarkMessageTextReqBuilder struct {
	// 消息内容
	Text string `json:"text"`
}

func (b *LarkMessageTextReqBuilder) Build() LarkMessageReq {
	req := LarkMessageReq{
		MsgType: "text",
	}
	req.Content.Text = b.Text
	return req
}
