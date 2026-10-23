package error_code

import (
	"errors"
	"testing"
)

var testBizCode = NewErrorCode(200, 600000001, "Biz error", "zh", "业务错误")

func TestNewErrorCodeRegistersForLookup(t *testing.T) {
	err := Code2Error(600000001, "detail")
	sc, ok, _ := MustUnwrap(err)
	if !ok || sc.Code() != testBizCode.Code() {
		t.Fatalf("Code2Error got %v, want code %d", err, testBizCode.Code())
	}
	if sc.GetMessage("zh") != "业务错误" {
		t.Fatalf("GetMessage(zh) = %q", sc.GetMessage("zh"))
	}
}

func TestNewErrorCodeDuplicatePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate code should panic")
		}
	}()
	NewErrorCode(200, INVALID_PARAM.Code(), "dup")
}

func TestUnknownCodeFallsBack(t *testing.T) {
	sc, _, _ := MustUnwrap(Code2Error(699999999))
	if sc.Code() != INTERNAL_SERVER_ERROR.Code() {
		t.Fatalf("unknown code should map to INTERNAL_SERVER_ERROR, got %d", sc.Code())
	}
	sc, _, _ = MustUnwrap(GrpcCode2Error(699999999, "remote msg", "ext"))
	if sc.Code() != 699999999 || sc.GetMessage() != "remote msg" {
		t.Fatalf("GrpcCode2Error keeps remote code/msg, got %d %q", sc.Code(), sc.GetMessage())
	}
}

func TestNewMsgKeepsPercentLiteral(t *testing.T) {
	_, err := Unwrap(INVALID_PARAM.NewMsg("100% invalid"))
	if err == nil || err.Error() != "100% invalid" {
		t.Fatalf("got %v", err)
	}
	if !INVALID_PARAM.Is(INVALID_PARAM.New(errors.New("x"))) {
		t.Fatal("Is should match same code")
	}
}
