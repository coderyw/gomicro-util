package common_result_proto

import (
	"errors"
	"testing"
)

func TestErrorCodeNewAndUnwrap(t *testing.T) {
	base := errors.New("boom")
	err := ERROR_CODE_INVALID_PARAM.New(base)
	e, ok := err.(*ERROR_CODEError)
	if !ok {
		t.Fatalf("want *ERROR_CODEError, got %T", err)
	}
	code, inner := e.UnwrapCode()
	if code != ERROR_CODE_INVALID_PARAM || !errors.Is(inner, base) {
		t.Fatalf("UnwrapCode = %v, %v", code, inner)
	}
	if !Is(err, base) {
		t.Fatal("Is should see wrapped error")
	}
	if ERROR_CODE_RECORD_NOT_FOUND.StringAlias() != "Record not found" || ERROR_CODE(42).StringAlias() != "42" {
		t.Fatal("StringAlias mismatch")
	}
}

func TestCommonResultNilSafe(t *testing.T) {
	var r *CommonResult
	if !r.IsNil() || r.GetErrorCode() != ERROR_CODE_SUCCESS {
		t.Fatal("nil CommonResult should be safe")
	}
	r = &CommonResult{}
	r.SetErrorCode_(ERROR_CODE_INVALID_REQUEST)
	if r.GetErrorCode() != ERROR_CODE_INVALID_REQUEST {
		t.Fatal("setter failed")
	}
}
