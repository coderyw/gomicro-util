package hash_circle

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

func TestSortConsistentHash_Add(t *testing.T) {
	my := uuid.NewString()
	s := NewSortHash()
	s.Add(my)
	work := []string{"0",
		"1",
		"2"}
	//"4",
	//"5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15",
	//"16", "17", "18", "19", "20", "21", "22", "23", "24", "25", "26",
	//"27", "28", "29", "30", "31", "32", "33", "34", "35", "36", "37", "38",
	//"39", "40", "41", "42", "43", "44", "45", "46", "47", "48", "49", "50", "51",
	//"52", "53", "54", "55", "56", "57", "58", "59", "60", "61", "62", "63", "64", "65",
	//"66", "67", "68", "69", "70", "71", "72", "73", "74", "75", "76", "77", "78", "79",
	//"80", "81", "82", "83", "84", "85", "86", "87", "88", "89", "90", "91", "92", "93", "94", "95"}
	for _ = range 3 {

		for i := range 50 {
			s.Add(fmt.Sprintf("%v-%v", uuid.NewString(), i))
		}
		w := s.GetAll(work)
		m1 := make(map[string]struct{})
		for k, v := range w {
			for _, vv := range v {
				if _, ok := m1[vv]; ok {
					log.Warn("chognfufeibn", zap.Any(k, vv))
				} else {
					m1[vv] = struct{}{}
				}
			}

			//log.Info("all", zap.Any(k, v))
		}
		fmt.Println(len(w))
	}

	//<-single.OsKillSingle()
}

func BenchmarkSortConsistentHash_GetAll(b *testing.B) {
	//my := uuid.NewString()
	//s := NewSortHash()
	//s.Add(my)
	//work := []string{"1",
	//	"2",
	//	"3",
	//	"4",
	//	"5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15",
	//	"16", "17", "18", "19", "20", "21", "22", "23", "24", "25", "26",
	//	"27", "28", "29", "30", "31", "32", "33", "34", "35", "36", "37", "38",
	//	"39", "40", "41", "42", "43", "44", "45", "46", "47", "48", "49", "50", "51",
	//	"52", "53", "54", "55", "56", "57", "58", "59", "60", "61", "62", "63", "64", "65",
	//	"66", "67", "68", "69", "70", "71", "72", "73", "74", "75", "76", "77", "78", "79",
	//	"80", "81", "82", "83", "84", "85", "86", "87", "88", "89", "90", "91", "92", "93", "94", "95"}
	//for i := range 50 {
	//	s.Add(fmt.Sprintf("%v-%v", uuid.NewString(), i))
	//}
}

func TestSort(t *testing.T) {
	type As struct {
		i int
		b string
	}
	arr := []As{
		{i: 10, b: "10"},
		{i: 1, b: "1"},

		{i: 5, b: "5"},
		{i: 2, b: "2"},
		{i: 7, b: "7"},
		{i: 2, b: "3"},
		{i: 6, b: "6"},
		{i: 8, b: "8"},
		{i: 4, b: "4"},
		{i: 9, b: "9"},
	}
	aa := make([]As, 0)
	for _, v := range arr {
		aa = append(aa, As{i: v.i, b: v.b})
		slices.SortFunc(aa, func(a, b As) int {
			if a.i == b.i {
				return strings.Compare(strings.ToLower(a.b), strings.ToLower(b.b))
			}
			return a.i - b.i
		})
		fmt.Println(aa)

	}

}
