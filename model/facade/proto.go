/**
 * @Author: yinwei
 * @Description:
 * @File: proto
 * @Version: 1.0.0
 * @Date: 2023/10/28 13:38
 */
package facade

import common_result_proto "github.com/coderyw/gomicro-util/model/common-result"

type RespSet[T any] interface {
	ProtoRespFacade[T]
	CommRespFacade
}

// ProtoRespFacade grpc proto 消息回复统计接口，不包含数据
type ProtoRespFacade[T any] interface {
	GetData() T

	SetData_(val T)
}

type CommRespFacade interface {
	GetErrorCode() common_result_proto.ERROR_CODE

	GetErrorMsg() string

	GetExtraMessage() string

	SetErrorCode_(val common_result_proto.ERROR_CODE)

	SetErrorMsg_(val string)

	SetExtraMessage_(val string)

	IsNil() bool
}
