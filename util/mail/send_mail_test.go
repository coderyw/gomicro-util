// Package mail
// @Author: yinwei
// @File: send_mail_test.go
// @Version: 1.0.0
// @Date: 2024/12/9 18:24

package mail

import (
	"context"
	"fmt"
	"github.com/coderyw/gomicro-util/library/trace"
	"math/rand"
	"testing"
)

func TestSendMail(t *testing.T) {
	cfg := Config{
		MailHost:     "smtp.larksuite.com",
		MailPort:     465,
		MailUserName: "xavier@mastertech.one",
		MailPassword: "BNflNNlwE9ViskGW",
	}
	type args struct {
		ctx        context.Context
		from       string
		toMailAddr string
		title      string
		content    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "测试推送邮件", args: args{
			ctx:        trace.GetTraceCtx(context.Background()),
			from:       "xavier@mastertech.one",
			toMailAddr: "francis@mastertech.one",
			title:      "你好啊",
			content:    "测试推送邮件",
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SendMail(tt.args.ctx, cfg, tt.args.from, tt.args.toMailAddr, tt.args.title, tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("SendMail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSendMailToAddrs(t *testing.T) {
	cfg := Config{
		MailHost:     "smtp.larksuite.com",
		MailPort:     465,
		MailUserName: "xavier@mastertech.one",
		MailPassword: "BNflNNlwE9ViskGW",
	}
	type args struct {
		ctx        context.Context
		from       string
		toMailAddr []string
		title      string
		content    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "测试推送邮件", args: args{
			ctx:        trace.GetTraceCtx(context.Background()),
			from:       "xavier@mastertech.one",
			toMailAddr: []string{"francis@mastertech.one", "gasol.wang@nustargame.com.ph"},
			title:      "你好啊",
			content:    "测试推送邮件",
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SendMailToAddrs(tt.args.ctx, cfg, tt.args.from, tt.args.toMailAddr, tt.args.title, tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("SendMail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRound(t *testing.T) {
	for _ = range 10 {
		fmt.Println(rand.Int())
	}
}
