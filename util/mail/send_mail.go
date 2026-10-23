// Package mail
// @Author: yinwei
// @File: send_mail
// @Version: 1.0.0
// @Date: 2024/12/9 18:12

package mail

import (
	"context"
	"github.com/wneessen/go-mail"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"sync"
)

type Config struct {
	//
	MailHost     string
	MailPort     int
	MailUserName string
	MailPassword string
}

var (
	clients       sync.Map
	defaultClient *mail.Client
)

func initCfg(cfg Config) (*mail.Client, error) {

	//_, ok := clients.Load(fmt.Sprintf("%v:%v", cfg.MailHost, cfg.MailUserName))
	//if ok {
	//	return nil
	//}

	// 创建邮件客户端
	client, err := mail.NewClient(cfg.MailHost,
		mail.WithPort(cfg.MailPort),           // SMTP 端口
		mail.WithSMTPAuth(mail.SMTPAuthPlain), // 使用 PLAIN 认证
		mail.WithUsername(cfg.MailUserName),   // SMTP 用户名
		mail.WithPassword(cfg.MailPassword),   // SMTP 密码
		mail.WithSSL(),
		mail.WithLogger(&log.GoMailLog{}),
	)
	if err != nil {
		return nil, err
	}
	//clients.Store(fmt.Sprintf("%v:%v", cfg.MailHost, cfg.MailUserName), client)
	//if defaultClient == nil {
	//	defaultClient = client
	//}
	//client.Reset()
	return client, nil
}

// SendMail 通过默认客户端发送邮件
//
// 默认客户端必须通过 Init 函数进行初始化, 否则将返回错误.
//
// Parameters:
//   - ctx: 上下文
//   - cfg: 邮件配置
//   - fromName: 发件人名称
//   - toMailAddr: 收件人邮箱地址
//   - title: 邮件标题
//   - content: 邮件正文
//   - attachFilePath: 可变参数附件路径，如果传了多个，只会取第一个。
//
// Returns:
//   - error: 发送邮件期间的错误
func SendMail(ctx context.Context, cfg Config, fromName, toMailAddr string, title, content string, attachFilePath ...string) error {
	cli, err := initCfg(cfg)
	if err != nil {
		return err
	}
	defer cli.Close()
	// 创建邮件
	msg := mail.NewMsg()
	// 发件人
	if err := msg.From(fromName); err != nil {
		return err
	}
	// 收件人
	if err := msg.To(toMailAddr); err != nil {
		return err
	}
	msg.Subject(title)                             // 邮件主题
	msg.SetBodyString(mail.TypeTextPlain, content) // 邮件正文
	if len(attachFilePath) > 0 {
		msg.AttachFile(attachFilePath[0])
	}

	// 发送邮件
	if err := cli.DialAndSend(msg); err != nil {
		log.ErrorCtx(ctx, "发送邮件错误", zap.Error(err))
		return err
	}
	return nil
}

// SendMailToAddrs 通过默认客户端发送邮件给多个收件人
//
// 默认客户端必须通过 Init 函数进行初始化, 否则将返回错误.
//
// Parameters:
//   - ctx: 上下文
//   - cfg: 邮件配置
//   - fromName: 发件人名称
//   - toMailAddr: []收件人邮箱地址
//   - title: 邮件标题
//   - content: 邮件正文
//   - attachFilePath: 可变参数附件路径，如果传了多个，只会取第一个。
//
// Returns:
//   - error: 发送邮件期间的错误
func SendMailToAddrs(ctx context.Context, cfg Config, fromName string, toMailAddr []string, title, content string, attachFilePath ...string) error {
	cli, err := initCfg(cfg)
	if err != nil {
		return err
	}
	defer cli.Close()
	// 创建邮件
	msg := mail.NewMsg()
	// 发件人
	if err := msg.From(fromName); err != nil {
		return err
	}
	// 收件人
	if err := msg.To(toMailAddr...); err != nil {
		return err
	}
	msg.Subject(title)                             // 邮件主题
	msg.SetBodyString(mail.TypeTextPlain, content) // 邮件正文
	if len(attachFilePath) > 0 {
		msg.AttachFile(attachFilePath[0])
	}

	// 发送邮件
	if err := cli.DialAndSend(msg); err != nil {
		log.ErrorCtx(ctx, "发送邮件错误", zap.Error(err))
		return err
	}
	return nil
}
