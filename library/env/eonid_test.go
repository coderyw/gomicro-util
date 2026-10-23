package env

import (
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/model/cst"
)

func TestGetDisableEonidHashTag(t *testing.T) {
	original := disableEonidHashTag
	t.Cleanup(func() { disableEonidHashTag = original })
	for _, tc := range []struct {
		value   string
		want    bool
		wantErr bool
	}{
		{"", false, false},
		{"0", false, false},
		{"1", true, false},
		{"2", false, true},
		{"true", false, true},
		{"false", false, true},
		{" 1 ", false, true},
	} {
		t.Run("value="+tc.value, func(t *testing.T) {
			disableEonidHashTag = tc.value
			got, err := GetDisableEonidHashTag()
			if got != tc.want || (err != nil) != tc.wantErr {
				t.Fatalf("got (%v, %v), want (%v, error=%v)", got, err, tc.want, tc.wantErr)
			}
			if tc.wantErr {
				if err := checkEnv(); err == nil || !strings.Contains(err.Error(), cst.ENV_DISABLE_EONID_HASHTAG) {
					t.Fatalf("startup must reject invalid setting: %v", err)
				}
			}
		})
	}
}
