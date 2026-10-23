// Package lark_message
// @Author: yinwei
// @File: share_chat_model
// @Version: 1.0.0
// @Date: 2025/6/4 11:02

package lark_message

type LarkMessageShareChatReqBuilder struct {
	ShareChatId string `json:"share_chat_id"`
}

func (b *LarkMessageShareChatReqBuilder) Build() LarkMessageReq {
	req := LarkMessageReq{
		MsgType: "share_chat",
	}
	req.Content.ShareChatId = b.ShareChatId
	return req
}
