// Package cron
// @Author: yinwei
// @File: robfig_cron
// @Version: 1.0.0
// @Date: 2025/6/13 13:25

package cron

import (
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/coderyw/gomicro-util/library/log"
	cron2 "github.com/coderyw/gomicro-util/model/cron"
	"go.uber.org/zap"
)

var (
	c    *cron.Cron
	lock sync.RWMutex
	jobs = sync.Map{}
	once sync.Once
)

func init() {
	Init()
}

// AddCron 添加定时任务
//
// Parameters:
//   - cronJob: cron2.CronJob
//
// Returns:
func AddCron(cronJob cron2.CronJob) {
	lock.RLock()
	defer lock.RUnlock()
	if c == nil {
		return
	}
	if _, ok := jobs.Load(cronJob.Name); ok {
		return
	}
	var chain cron.Chain
	// 中间件
	if cronJob.DoBeforeDone { //任务没完成立即执行
		chain = cron.NewChain(BeforeRun())
	} else if cronJob.DelayBeforeDone { //延迟下次执行
		chain = cron.NewChain(cron.DelayIfStillRunning(log.GetCronLogger()), BeforeRun())
	} else if !cronJob.DelayBeforeDone { //跳过下次执行
		chain = cron.NewChain(cron.SkipIfStillRunning(log.GetCronLogger()), BeforeRun())
	}
	job := chain.Then(cron.FuncJob(cronJob.DoJob))

	entryId, err := c.AddJob(cronJob.CronSpc, job)
	if err != nil {
		log.Error("添加定时任务错误", zap.Any("schedule", job), zap.Error(err))
	} else {
		cronJob.EntryId = entryId
		jobs.Store(cronJob.Name, cronJob)
	}
}

func Init() {
	lock.Lock()
	defer lock.Unlock()
	once.Do(func() {
		if c == nil {
			c = cron.New(cron.WithLogger(log.GetCronLogger()), cron.WithSeconds())
		}
		c.Start()
	})
}

// Stop 停止定时器，如果不传任务名则不处理
//
// Parameters:
//   - name: ...string 任务名称
func Stop(name ...string) {
	lock.RLock()
	defer lock.RUnlock()
	if c != nil && len(name) > 0 { //清除所有对应任务
		for _, v := range name {
			if vv, ok := jobs.Load(v); ok {
				job := vv.(cron2.CronJob)
				c.Remove(job.EntryId)
				jobs.Delete(v)
			}
		}
	}
	log.Info("[Cron] stop job", zap.Strings("name", name))
}

func CloseCron() {
	lock.Lock()
	defer lock.Unlock()
	if c == nil {
		return
	}
	ctx := c.Stop()
	<-ctx.Done()
	jobs.Clear()
	c = nil
	log.Info("定时任务已停止")
}

func BeforeRun() cron.JobWrapper {
	return func(j cron.Job) cron.Job {

		return cron.FuncJob(func() {
			//log.Info("do before run")
			j.Run()
		})
	}
}
