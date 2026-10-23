// Package lark_message
// @Author: yinwei
// @File: card_model
// @Version: 1.0.0
// @Date: 2025/6/4 10:57

package lark_message

// 发送消息卡片
type LarkMessageCardReqBuilder struct {
	Elements []CardElements `json:"elements"`
	Header   CardHeader     `json:"header"`
}

func (b *LarkMessageCardReqBuilder) Build() LarkMessageReq {
	req := LarkMessageReq{
		MsgType: "interactive",
	}
	req.Card.Elements = b.Elements
	req.Card.Header = b.Header
	return req
}

type CardText struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}

type CardActions struct {
	Tag   string                 `json:"tag"`
	Text  CardText               `json:"text"`
	URL   string                 `json:"url"`
	Type  string                 `json:"type"`
	Value map[string]interface{} `json:"value"`
}
type CardElements struct {
	Tag     string        `json:"tag"`
	Text    CardText      `json:"text,omitempty"`
	Actions []CardActions `json:"actions,omitempty"`
}
type CardTitle struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}
type CardHeader struct {
	Title CardTitle `json:"title"`
}
