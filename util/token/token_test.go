package token

import (
	"fmt"
	"os"
	"testing"
)

func TestParseToken(t *testing.T) {
	os.Setenv("ENVIRON", "server-test")
	str := "bet eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIyMDY3IiwiY3JlYXRlZCI6MTczMjg2NjEwNjYyNiwiaWF0IjoxNzMyODY2MTA2LCJleHAiOjE3MzMxMjUzMDZ9.kIk5BsBvpc3xekvKgovzVkqbvqacSaVAtLz2ScdhlMg"
	token, err := ParseToken(str, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf(token)
}

func TestMakeAesToken(t *testing.T) {
	str := "IAmToken"
	key := "1234567890123456"
	iv := "1234567890123456"
	token, err := MakeEnAesToken(str, key, iv)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(token)

	token, err = GetDeAesToken(token, key, iv)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(token)
}
