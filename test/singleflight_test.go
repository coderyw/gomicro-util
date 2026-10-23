package test

import (
	"fmt"
	"golang.org/x/sync/singleflight"
	"testing"
)

func TestSingleflight(t *testing.T) {
	a := singleflight.Group{}

	count := 0

	//wg := sync.WaitGroup{}
	//for i := range 10 {
	//	wg.Add(1)
	//	time.Sleep(10 * time.Millisecond)
	//	go func() {
	//		defer wg.Done()
	//		val, _, _ := a.Do("test", func() (interface{}, error) {
	//			fmt.Printf("%v:got\n", i)
	//			if count == 0 {
	//				count++
	//				return 1, nil
	//			}
	//			return 2, nil
	//
	//		})
	//		fmt.Printf("%v:%v\n", i, val)
	//	}()
	//}
	//wg.Wait()
	val, _, _ := a.Do("test", func() (interface{}, error) {
		fmt.Printf("%v:got\n", 99999)
		if count == 0 {
			return 1, nil
		}
		return 2, nil

	})
	fmt.Printf("%v:%v\n", 99999, val)

}
