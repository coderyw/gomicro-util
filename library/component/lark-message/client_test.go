// Package lark_message
// @Author: yinwei
// @File: client_test.go
// @Version: 1.0.0
// @Date: 2025/6/4 11:34

package lark_message

import (
	"context"
	srv_config "github.com/coderyw/gomicro-util/model/config"
	"testing"
	"time"
)

func Init() {
	srv_config.LarkConfig = srv_config.LarkCfg{
		Groups: map[string]srv_config.LarkGroup{
			"test": {
				WebHook: "https://open.larksuite.com/open-apis/bot/v2/hook/9543e7c0-f684-4d14-b29c-fbdb98797778",
				Sign:    "NwXrdsho8ofin7rCYQb0Cc",
			},
		},
	}
}

func TestSendLarkMessage(t *testing.T) {
	Init()
	type args struct {
		ctx     context.Context
		groupId string
		msg     LarkMessageBuilder
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "text", args: args{
			ctx:     context.Background(),
			groupId: "test",
			msg: &LarkMessageTextReqBuilder{
				Text: "test message",
			},
		}, wantErr: false},
		{name: "post", args: args{
			ctx:     context.Background(),
			groupId: "test",
			msg: &LarkMessagePostReqBuilder{
				Post: Post{
					ZhCn: PostZhCn{
						Title: "test title",
						Content: [][]PostContent{
							{
								{
									Tag:  "text",
									Text: "项目有更新: ",
								},
								{
									Tag:  "a",
									Text: "请查看",
									PostA: PostA{
										Href: "https://www.baidu.com",
									},
								},
								{
									Tag: "at",
									PostAt: PostAt{
										UserID: "ou_7d8a6e6df7621556ce0d21922b676706ccs",
									},
								},
							},
						},
					},
				},
			},
		}, wantErr: false},
		{name: "post", args: args{
			ctx:     context.Background(),
			groupId: "test",
			msg: &LarkMessagePostReqBuilder{
				Post: Post{
					ZhCn: PostZhCn{
						Title: "充值/提现预警",
						Content: [][]PostContent{
							{
								{
									Tag:  "text",
									Text: "时间: " + time.Now().Format(time.DateTime),
								},
							},
							{
								{
									Tag:  "text",
									Text: "充值成功订单金额: 1,000,000",
								},
							},
							{
								{
									Tag:  "text",
									Text: "GCash: 400,000",
								},
							},
							{
								{
									Tag:  "text",
									Text: "Maya: 4600,000",
								},
							},
						},
					},
				},
			},
		}, wantErr: false},
		{
			name: "card",
			args: args{
				ctx:     context.Background(),
				groupId: "test",
				msg: &LarkMessageCardReqBuilder{
					Header: CardHeader{
						Title: CardTitle{
							Content: "今日旅游推荐",
							Tag:     "plain_text",
						},
					},
					Elements: []CardElements{
						{
							Tag: "div",
							Text: CardText{
								Content: "**西湖**，位于浙江省杭州市西湖区龙井路1号，杭州市区西部，景区总面积49平方千米，汇水面积为21.22平方千米，湖面面积为6.38平方千米。",
								Tag:     "lark_md",
							},
						},
						{
							Actions: []CardActions{
								{
									Tag: "button",
									Text: CardText{
										Content: "更多景点介绍 :玫瑰:",
										Tag:     "lark_md",
									},
									URL:  "https://www.example.com",
									Type: "default",
									Value: map[string]interface{}{
										"key": "value",
									},
								},
							},
							Tag: "action",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SendLarkMessage(tt.args.ctx, tt.args.groupId, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("SendLarkMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
