// Package etcd_config
// @Author: yinwei
// @File: task
// @Version: 1.0.0
// @Date: 2024/6/19 09:00

package config

type TaskInfo struct {
	Content     string `json:"content"`      //任务内容描述
	ServiceName string `json:"service_name"` //服务名
	Endpoint    string `json:"endpoint"`     //调用方法
	TaskTimeout int    `json:"task_timeout"` //任务执行超时时间,0不超时
	Schedule    Timer  `json:"schedule"`     //定时信息
}

type Timer struct {
	// 定时时间 支持linux cron定时格式
	T string `json:"t"` //定时时间
	//任务未完成是否可以执行下次定时任务 true=可以 false=不可以
	DoBeforeDone bool `json:"do_before_done"`
	// 任务未完成时延迟直到当前任务完成才执行下次任务，还是跳过下次任务
	// 只有在DoBeforeDone 为false才生效
	DelayBeforeDone bool `json:"delay_before_done"`
}
