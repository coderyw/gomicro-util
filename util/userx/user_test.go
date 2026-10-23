package userx

import (
	"fmt"
	"testing"
)

func TestParseToken(t *testing.T) {
	userID := "PZ1234"
	userID = UserID2Username(userID)
	fmt.Println(userID)
	fmt.Println(Username2UserID(userID))
}
