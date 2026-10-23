/**
 * @Author: yinwei
 * @Description: 延迟队列分类
 * @File: delay_tube
 * @Version: 1.0.0
 * @Date: 2023/10/31 12:37
 */
package cst

//延迟job类型

type DelayTubeType string

const (
	DELAY_TUBE_NAME_TASK DelayTubeType = "delay_task" //延迟任务
)
