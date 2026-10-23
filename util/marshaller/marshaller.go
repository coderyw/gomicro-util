// Package marshall
// @Author: yinwei
// @File: marshaller
// @Version: 1.0.0
// @Date: 2024/11/4 15:36

package marshaller

import (
	"encoding/json"
)

// MapToStruct
//
//	@Description: map转结构体，不使用官方包，使用json中间方法
//	@Author	yw 2024-11-04 15:37:22
//	@param	data
//	@param	v
//	@return	error
func MapToStruct(data map[string]interface{}, v interface{}) error {
	d, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(d, v)
}
