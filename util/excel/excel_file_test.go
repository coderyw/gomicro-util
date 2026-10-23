// Package excel
// @Author: yinwei
// @File: excel_file_test.go
// @Version: 1.0.0
// @Date: 2025/1/22 14:52

package excel

import (
	"sync"
	"testing"
)

func TestGetExcelColumnName(t *testing.T) {
	type args struct {
		i int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "A", args: args{1}, want: "A"},
		{name: "Z", args: args{26}, want: "Z"},
		{name: "AA", args: args{27}, want: "AA"},
		{name: "AZ", args: args{52}, want: "AZ"},
		{name: "BA", args: args{53}, want: "BA"},
		{name: "AAA", args: args{703}, want: "AAA"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetExcelColumnName(tt.args.i); got != tt.want {
				t.Errorf("GetExcelColumnName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	ex, err := NewExcelWriter("./", "test.xlsx", Options{
		Sort: []string{"name", "age"},
	})
	if err != nil {
		t.Errorf("NewExcelWriter err=%v", err)
	}

	for i := 0; i < 100; i++ {
		if err := ex.WriteArray([]any{i, "name"}); err != nil {
			t.Errorf("Write err=%v", err)
		}
	}
	if err := ex.SaveAndClose(); err != nil {
		t.Errorf("Write err=%v", err)
	}
}

func TestBatchWriteArray(t *testing.T) {
	ex, err := NewExcelWriter("./", "test_batchwrite.xlsx", Options{
		Sort: []string{"name", "age"},
	})
	if err != nil {
		t.Errorf("NewExcelWriter err=%v", err)
	}
	arr := [][]any{}
	for i := 0; i < 100; i++ {
		arr = append(arr, []any{i, "name"})
	}
	if err := ex.BatchWriteArray(arr); err != nil {
		t.Errorf("Write err=%v", err)
	}
	if err := ex.SaveAndClose(); err != nil {
		t.Errorf("Write err=%v", err)
	}
}

func TestWriteArrayGoroutine(t *testing.T) {
	ex, err := NewExcelWriter("./", "test_goroutine_write.xlsx", Options{
		Sort: []string{"name", "age"},
	})
	if err != nil {
		t.Errorf("NewExcelWriter err=%v", err)
	}
	wg := sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if err := ex.WriteArray([]any{i, "name"}); err != nil {
					t.Errorf("Write err=%v", err)
				}
			}
		}()

	}
	wg.Wait()
	if err := ex.SaveAndClose(); err != nil {
		t.Errorf("Write err=%v", err)
	}

}

func TestWriteMap(t *testing.T) {
	ex, err := NewExcelWriter("./", "test_writemap.xlsx", Options{
		Sort: []string{"name", "age"},
	})
	if err != nil {
		t.Errorf("NewExcelWriter err=%v", err)
	}

	if err := ex.WriteMap(map[string]any{
		"A1": 23,
		"B2": "张三",
	}); err != nil {
		t.Errorf("Write err=%v", err)
	}
	if err := ex.SaveAndClose(); err != nil {
		t.Errorf("Write err=%v", err)
	}
}
