// Package lark_message
// @Author: yinwei
// @File: model
// @Version: 1.0.0
// @Date: 2025/6/4 10:56

package lark_message

type LarkMessageBuilder interface {
	Build() LarkMessageReq
}

type Sign struct {
	Timestamp string `json:"timestamp"`
	Sign      string `json:"sign"`
}

type LarkMessageReq struct {
	Sign
	MsgType string `json:"msg_type"`

	Card struct {
		Elements []CardElements `json:"elements"`
		Header   CardHeader     `json:"header"`
	} `json:"card,omitempty"`

	Content struct {
		Post        Post   `json:"post,omitempty"`
		ImageKey    string `json:"image_key,omitempty"`
		ShareChatId string `json:"share_chat_id,omitempty"`
		Text        string `json:"text,omitempty"`
	} `json:"content,omitempty"`
}

type LarkMessageResp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}
