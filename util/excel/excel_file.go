// Package excel
// @Author: yinwei
// @File: excel_file
// @Version: 1.0.0
// @Date: 2025/1/22 14:10

package excel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
)

type Excel struct {
	path   string
	file   *excelize.File
	stream *excelize.StreamWriter
	sort   []string
	sync.Mutex
	index     int
	sheetName string
}

// Options
//
// 选项定义打开和读取电子表格的选项。
//
// MaxCalcIterations 指定迭代计算的最大迭代次数，默认值为0。
//
// Password 以明文形式指定电子表格的密码。
//
// RawCellValue 指定是否对单元格值应用数字格式或获取原始值。
//
// UnzipSizeLimit 指定打开电子表格时解压缩大小限制（以字节为单位），该值应大于或等于.
//
// UnzipXMLSizeLimit，默认大小限制为 16GB。
//
// UnzipXMLSizeLimit 指定解压工作表和共享字符串表的内存限制（以字节为单位），当文件大小超过此值时，工作表 XML 将被提取到系统临时目录，
// 此值应小于或等于
//
// UnzipSizeLimit，默认值为 16MB。
//
// ShortDatePattern 指定短日期数字格式代码。在电子表格应用程序中，日期格式将日期和时间序列号显示为日期值。
// 以星号 () 开头的日期格式响应为操作系统指定的区域日期和时间设置的更改。不带星号的格式不受操作系统设置的影响。
//
// ShortDatePattern 用于指定应用以星号开头的日期格式。
//
// LongDatePattern 指定长日期数字格式代码。
//
// LongTimePattern 指定长时间数字格式代码。
//
// CultureInfo 指定应用内置语言数字格式代码的国家/地区代码，这些代码受系统本地语言设置的影响。
//
// # SheetName sheet的名称，默认为Sheet1
//
// Sort 列名配置
type Options struct {
	MaxCalcIterations uint
	Password          string
	RawCellValue      bool
	UnzipSizeLimit    int64
	UnzipXMLSizeLimit int64
	ShortDatePattern  string
	LongDatePattern   string
	LongTimePattern   string
	CultureInfo       CultureName
	SheetName         string
	Sort              []string
	StreamStartIndex  int // 流方式写入的初始索引
}

type CultureName byte

const (
	CultureNameUnknown CultureName = iota
	CultureNameEnUS
	CultureNameZhCN
)

// NewExcelWriter 初始化excel writer。
//
// 需要传入写入的泛型
//
// Parameters:
//   - path: string 文件路径
//   - filename: string 文件名
//   - option: ...Options
//
// Returns:
//   - *Excel[T]
//   - error
func NewExcelWriter(path string, filename string, option ...Options) (*Excel, error) {
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	e := new(Excel)
	e.sheetName = "Sheet1"
	e.path = filepath.Join(path, filename)
	var opt []excelize.Options
	if len(option) > 0 {
		opt = []excelize.Options{
			{
				MaxCalcIterations: option[0].MaxCalcIterations,
				Password:          option[0].Password,
				RawCellValue:      option[0].RawCellValue,
				UnzipSizeLimit:    option[0].UnzipSizeLimit,
				UnzipXMLSizeLimit: option[0].UnzipXMLSizeLimit,
				ShortDatePattern:  option[0].ShortDatePattern,
				LongDatePattern:   option[0].LongDatePattern,
				LongTimePattern:   option[0].LongTimePattern,
				CultureInfo:       excelize.CultureName(option[0].CultureInfo),
			},
		}
		if len(option[0].SheetName) > 0 {
			e.sheetName = option[0].SheetName
		}
		e.sort = option[0].Sort
		e.index = option[0].StreamStartIndex
	}

	e.file = excelize.NewFile(opt...)
	stream, err := e.file.NewStreamWriter(e.sheetName)
	if err != nil {
		return nil, err
	}
	e.stream = stream
	if len(e.sort) != 0 { // 初始化列名
		// 遍历所有列名，设置列名
		rowName := make([]any, 0, len(e.sort))
		for _, k := range e.sort {
			rowName = append(rowName, k)
		}
		if err := e.stream.SetRow("A1", rowName); err != nil {
			return nil, err
		}
	}
	if e.index == 0 {
		e.index = 2 //首行默认标题
	}
	return e, nil
}

func (e *Excel) GetFileFullName() string {
	return e.path
}

// WriteArray 写入一个数组到Excel表格中。
//
// startRow: 开始写入的行号。
// list: 要写入的数组。
//
// 返回: 如果写入过程中发生错误，则返回错误。
func (e *Excel) WriteArray(list []any) error {
	// 如果数组为空，则不执行任何操作并返回。
	if len(list) == 0 {
		return nil
	}
	// 加锁以确保线程安全。
	e.Lock()
	defer e.Unlock()
	// 将数组写入到指定的行中。
	err := e.stream.SetRow(fmt.Sprintf("A%v", e.index), list)
	if err != nil {
		return err
	}
	e.index++
	return nil
}

// BatchWriteArray 批量写入二维数组到Excel文件中。
// 该方法主要用于高效地将内存中的二维数组数据写入到Excel文件中。
//
// 参数:
//
//	lists ([][]any): 一个二维数组，包含要写入Excel的多行数据。
//
// 返回值:
//
//	error: 如果写入过程中发生错误，则返回错误信息；否则返回nil。
func (e *Excel) BatchWriteArray(lists [][]any) error {
	// 加锁以确保并发安全
	e.Lock()
	defer e.Unlock()

	// 遍历二维数组中的每一行数据
	for _, list := range lists {
		// 跳过空行以避免不必要的写入操作
		if len(list) == 0 {
			continue
		}
		// 将一行数据写入到Excel中，起始位置为"A1"
		// 注意：这里假设调用者已经打开了正确的工作表和工作簿
		if err := e.stream.SetRow(fmt.Sprintf("A%v", e.index), list); err != nil {
			return err
		}
		e.index++
	}
	// 所有数据写入成功，返回nil
	return nil
}

// WriteMap 将map数据写入Excel文件。
// 该方法遍历map，将键值对作为行写入Excel，其中键代表行的标识，值代表行的数据。
//
// 参数:
//
//	m - 待写入Excel的map，键为字符串类型，值为任意类型。
//
// 返回值:
//
//	如果写入过程中发生错误，则返回错误。
func (e *Excel) WriteMap(m map[string]any) error {
	// 遍历map，将每个键值对写入Excel文件的一行中。
	for k, v := range m {
		// 使用stream的SetRow方法将当前键值对写入Excel。
		// 键k作为行的标识，值v作为行的数据，封装在一个切片中。
		if err := e.stream.SetRow(k, []any{v}); err != nil {
			return err
		}
	}
	return nil
}

// SaveAndClose 保存并关闭
//
// Parameters:
//
// Returns:
//   - error
func (e *Excel) SaveAndClose() error {
	if err := e.stream.Flush(); err != nil {
		return err
	}
	if err := e.file.SaveAs(e.path); err != nil {
		return err
	}
	return e.file.Close()
}

var colName = []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}

// i=0 A i=26 Z
// i=27 AA i=28 AB i=29 AC i=30 AD i=31 AE i=32 AF i=33 AG i=34 AH i=35 AI i=36 AJ i=37 AK i=38 AL i=39 AM i=40 AN i=41 AO i=42 AP i=43 AQ i=44 AR i=45 AS i=46 AT i=47 AU i=48 AV i=49 AW i=50 AX i=51 AY i=52 AZ i=53 BA i=54 BB i=55 BC i=56 BD i=57 BE i=58 BF i=59 BG i=60 BH i=61 BI i=62 BJ i=63 BK i=64 BL i=65 BM i=66BN i=67 BO i=68 BP i=69 BQ i

// GetExcelColumnName 根据excel第i列返回具体的excel的列名称
//
// Parameters:
//   - i: int 第几列，从1开始
//
// Returns:
//   - string 列名，比如 A B C AC
func GetExcelColumnName(i int) string {
	if i < 1 {
		return ""
	}
	var result strings.Builder
	// 预分配足够的空间，避免多次扩容
	result.Grow(10) // Excel列名通常不会太长，10足够
	for i > 0 {
		i-- // 因为A对应1，所以需要减1
		remainder := i % 26
		result.WriteByte('A' + byte(remainder))
		i = i / 26
	}
	// 由于我们是从低位到高位写入的，需要反转Builder中的内容
	runes := []byte(result.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
