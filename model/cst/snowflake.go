/**
 * @Author: yinwei
 * @Description: 雪花算法 区分key, 生成的id，在每个key下是全局唯一的。
 * @File: snowflake
 * @Version: 1.0.0
 * @Date: 2023/11/1 12:13
 */
package cst

const (
	SNOWFLAKE_KEY_TASK            = "task"             // 任务分布式id key
	SNOWFLAKE_KEY_USER_CLIENT     = "user_client"      // 用户推送信息分布式id key
	SNOWFLAKE_KEY_GPECHAT_MSG     = "gpt_chat_msg"     // gpt聊天内容表
	SNOWFLAKE_KEY_GPECHAT_SESSION = "gpt_chat_session" // gpt聊天session表（同一个聊天窗口）

)
