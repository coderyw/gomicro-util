// Package json
// @Author: yinwei
// @File: decode_struct
// @Version: 1.0.0
// @Date: 2025/4/23 16:26

package json

//type StructDecoder struct{}
//
//func (d *StructDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
//	switch iter.WhatIsNext() {
//	case jsoniter.ArrayValue:
//		if !iter.ReadArray() {
//			// 是空数组 []
//			return
//		} else {
//			iter.ReportError("StructDecoder", "unsupported value type for struct")
//		}
//	case jsoniter.ObjectValue:
//		return
//	default:
//		iter.ReportError("StructDecoder", "unsupported value type for struct")
//	}
//}
