/**
 * @Author: yinwei
 * @Description:
 * @File: grpc_error
 * @Version: 1.0.0
 * @Date: 2023/10/28 13:42
 */
package e_error

import (
	"fmt"
	commonResult_proto "github.com/coderyw/gomicro-util/model/common-result"
	"github.com/coderyw/gomicro-util/model/facade"
	"reflect"
)

var defaultProto = commonResult_proto.CommonResult{}

// Deprecated: 请使用 common-model
// Unwrap2Error[T any]
//
//	 @Description: 解析带有data的rpc返回数据，返回data数据和code报错信息。
//		如果code不为0，则返回error
//	 @Author yw 2024-07-03 13:24:49
//	 @param obj
//	 @param err
//	 @return T
//	 @return error
func Unwrap2Error[T any](obj facade.RespSet[T], err error) (T, error) {
	return obj.GetData(), UnwrapCom2Error(obj, err)
}

// Deprecated: 请使用 common-model
// UnwrapCom2Error
//
//	@Description: 解析commresult到error，如果errorcode=0.则返回空错误
//	@Author yw 2024-07-03 13:24:23
//	@param obj
//	@param err
//	@return error
func UnwrapCom2Error(obj facade.CommRespFacade, err error) error {
	if err != nil {
		if _, ok := err.(*commonResult_proto.ERROR_CODEError); ok {
			return err
		}
		return commonResult_proto.ERROR_CODE_INTERNAL_SERVER_ERROR.New(err)
	}
	if obj != nil && obj.GetErrorCode() != 0 {
		return obj.GetErrorCode().New(fmt.Errorf(obj.GetExtraMessage()))
	}
	return nil
}

// Deprecated: 请使用 common-model
// Unwrap2ProtoResp 解析err到返回数据
// 如果result中已经包含code，会覆盖
// func Unwrap2ProtoResp(result facade.ProtoRespFacade, err error) facade.ProtoRespFacade {
//
//		if err != nil {
//			code, err := Unwrap(err)
//			a := defaultProto
//			result = &a
//			result.SetCode_(code)
//			result.SetExtraMessage_(err.Error())
//		}
//		result.SetMessage_(result.GetCode().StringAlias())
//		return result
//	}
//

// Deprecated: 请使用 common-model
// result必须是facade.ProtoRespFacade 类型，否则会返回ProtoRespFacade类型
func Unwrap2ProtoResp(result facade.CommRespFacade, err error) (facade.CommRespFacade, error) {
	if result == nil || result.IsNil() {
		a := defaultProto
		result = &a
	}
	var code commonResult_proto.ERROR_CODE
	if err != nil {
		code, err = Unwrap(err)

		result.SetErrorCode_(code)
		result.SetExtraMessage_(err.Error())
	}
	result.SetErrorMsg_(result.GetErrorCode().StringAlias())
	return result, err
}

// Deprecated: 请使用 common-model
func IsNil(input interface{}) bool {
	if input == nil {
		return true
	}
	if reflect.TypeOf(input).Kind() == reflect.Ptr && reflect.ValueOf(input).IsNil() {
		return true
	}
	return false
}
