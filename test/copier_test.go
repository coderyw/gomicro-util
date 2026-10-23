// Package test
// @Author: yinwei
// @File: copier_test
// @Version: 1.0.0
// @Date: 2024/10/23 16:06

package test

import (
	"fmt"
	"github.com/jinzhu/copier"
	"github.com/coderyw/gomicro-util/test/model"
	"testing"
)

func TestMarshalMap(t *testing.T) {
	s := model.Source{
		A: 1,
		B: 2,
		C: []string{"234", "fwfe"},
		D: model.Source1{
			D: "source",
		},
	}

	data, _ := s.MarshalJSON()
	target := model.Target{}
	target.UnmarshalJSON(data)
	fmt.Printf("%+v", target)

}

func BenchmarkCopier(b *testing.B) {
	s := model.Source{
		A: 1,
		B: 2,
		C: []string{"234", "fwfe"},
		D: model.Source1{
			D: "source",
		},
	}
	b.Run("copier", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			t := model.Target{}

			copier.Copy(&t, &s)
		}
	})

	b.Run("hand", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = model.Target{
				A: s.A,
				B: s.B,
				C: s.C,
				D: model.Target1{
					D: s.D.D,
				},
			}

		}
	})
	b.Run("easyjson", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, _ := s.MarshalJSON()
			t := model.Target{}
			t.UnmarshalJSON(data)
		}
	})
	b.Run("easymap", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, _ := s.MarshalMap()
			t := model.Target{}
			t.UnMarshalMapInterface(data)
		}
	})
}
