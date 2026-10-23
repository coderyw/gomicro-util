// Package excel
// @Author: yinwei
// @File: excel_inter
// @Version: 1.0.0
// @Date: 2025/1/22 14:13

package excel

type ExcelType interface {
	// Map 返回列名与数据的映射，列名需要约 Cell 中返回的一致
	//
	// Returns:
	//  - map[string]interface{} 列名与数据的映射
	Map() map[string]any
	// Cell 返回单元格数据 通过数组形式返回，数据会按照列名顺序排放入表格内
	//
	// Returns:
	//  - []string 单元格数据
	Cell() []string
}
