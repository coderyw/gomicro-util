// Package lark_message
// @Author: yinwei
// @File: post_model
// @Version: 1.0.0
// @Date: 2025/6/4 11:00

package lark_message

// 富文本

type LarkMessagePostReqBuilder struct {
	Post Post `json:"post"`
}

func (b *LarkMessagePostReqBuilder) Build() LarkMessageReq {
	req := LarkMessageReq{
		MsgType: "post",
	}
	req.Content.Post = b.Post
	return req
}

type PostContent struct {
	// text/a/at/img
	Tag string `json:"tag"`
	//文本内容。
	Text string `json:"text,omitempty"  binding:"required=true"`
	PostText
	PostA
	PostAt
	PostImg
}

// 文本标签：text
type PostText struct {

	//表示是否 unescape 解码。默认值为 false，未用到 unescape 时可以不填。
	UnEscape bool `json:"un_escape,omitempty"`
}

// 超链接标签：a
type PostA struct {
	//默认的链接地址，你需要确保链接地址的合法性，否则消息会发送失败。
	Href string `json:"href,omitempty"  binding:"required=true"`
}

type PostAt struct {
	// 用户的 Open ID。
	//- @ 单个用户时，user_id字段必须是有效值。
	//- @ 所有人时，填 all。
	UserID string `json:"user_id,omitempty" binding:"required=true"`
	//用户名称。
	UserName string `json:"user_name,omitempty"`
}

type PostImg struct {
	//图片的 key，图片的 key 通过上传图片接口获得。
	ImageKey string `json:"image_key,omitempty" binding:"required=true"`
}

// 是富文本的中、英文配置，富文本消息中至少需要包含一种语言的配置
type PostZhCn struct {
	//富文本消息的标题。
	Title string `json:"title,omitempty"`
	//富文本消息内容。由多个段落组成，每个段落为一个[]节点，其中包含若干个节点。
	Content [][]PostContent `json:"content,omitempty"`
}

// 富文本消息。
type Post struct {
	ZhCn PostZhCn `json:"zh_cn,omitempty"`
}
