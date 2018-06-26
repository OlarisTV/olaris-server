package auth

import (
	"fmt"
	"testing"
)

func TestStreamingTicket(t *testing.T) {
	path := "/users/maran/does/not/exist.mkv"
	token, err := CreateStreamingJWT(1, path)
	if err != nil {
		t.Errorf("Expected error to be nil, got error instead: %s", err.Error())
	}
	fmt.Println(token)
	claim, err := ValidateStreamingJWT(token)
	if err != nil {
		t.Errorf("Could not validate created token")
	}
	if claim.FilePath != path {
		t.Errorf("Filepath was not correct in token. Expected %s but got %s", path, claim.FilePath)
	}

	if claim.UserID != 1 {
		t.Errorf("User was not valid expected %d got %d", 1, claim.UserID)
	}

}
