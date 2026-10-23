/**
 * @Author: yinwei
 * @Description:
 * @File: context
 * @Version: 1.0.0
 * @Date: 2023/10/26 09:32
 */
package cst

// context 存储的元数据
const (
	CONTEXT_METADATA_USER_ID          = "user_id"          //元数据 用户id
	CONTEXT_METADATA_MUST_USER_ID     = "must_user_id"     //元数据 不会存储用户id用户id
	CONTEXT_METADATA_CLIENT_TYPE      = "client_type"      //元数据 客户端类型
	CONTEXT_METADATA_MUST_CLIENT_TYPE = "must_client_type" //元数据 客户端类型
	CONTEXT_METADATA_REQ_IP           = "request_id"       //元数据 请求端ip
	CONTEXT_METADATA_USER_SESSION     = "user_session"     //元数据 请求端ip
)
