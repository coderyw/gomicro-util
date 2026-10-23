/**
 * @Author: yinwei
 * @Description:
 * @File: error_test
 * @Version: 1.0.0
 * @Date: 2023/10/26 13:21
 */
package e_error

import (
	"fmt"
	"github.com/pkg/errors"
	commonResult "github.com/coderyw/gomicro-util/model/common-result"
	"github.com/coderyw/gomicro-util/library/db/cache/redis"
	"github.com/coderyw/gomicro-util/library/log"
	"testing"
)

var notFoundErr = fmt.Errorf("record not found")
var redisNil = redis.RedisNil

func TestEError_Is(t *testing.T) {

	t.Run("eErr=error", func(t *testing.T) {
		err := commonResult.ERROR_CODE_INVALID_PARAM.New(notFoundErr)
		fmt.Println(Is(err, notFoundErr))
	})
	t.Run("eErr=eErr", func(t *testing.T) {
		err1 := commonResult.ERROR_CODE_INVALID_PARAM.New(notFoundErr)
		err2 := commonResult.ERROR_CODE_INVALID_PARAM.New(notFoundErr)
		fmt.Println(Is(err1, err2))
	})
	t.Run("error=eErr", func(t *testing.T) {
		err := commonResult.ERROR_CODE_INVALID_PARAM.New(notFoundErr)
		fmt.Println(Is(notFoundErr, err))
	})
	t.Run("error=error", func(t *testing.T) {
		fmt.Println(Is(notFoundErr, notFoundErr))
	})
	t.Run("error!=error", func(t *testing.T) {
		fmt.Println(Is(notFoundErr, redisNil))
	})
	t.Run("error!=eErr", func(t *testing.T) {
		err := commonResult.ERROR_CODE_INVALID_PARAM.New(notFoundErr)
		fmt.Println(Is(redisNil, err))
	})
	t.Run("redisNil!=protoNil", func(t *testing.T) {
		fmt.Println(Is(redisNil, redis.RedisNil))
	})
}

func TestErrors(t *testing.T) {

	err := errors.New("my error")
	fmt.Printf("%+v\n", err)
	log.Info("errpr")

	err = errors.WithMessage(fmt.Errorf("hhh"), "23")
	fmt.Printf("%+v\n", err)

}
