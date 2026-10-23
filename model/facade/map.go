// Package facade
// @Author: yinwei
// @File: map
// @Version: 1.0.0
// @Date: 2023/11/13 13:28

package facade

//type MapMarshal interface {
//	MarshalMap() (map[string]interface{}, error)
//	UnMarshalMap(m map[string]string) error
//	UnMarshalMapInterface(m map[string]interface{}) error
//	CheckField(m map[string]string) error
//}

type HMField interface {
	CheckField(m map[string]string) error
}
