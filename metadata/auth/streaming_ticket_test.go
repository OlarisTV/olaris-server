package auth

import (
	"fmt"
	"testing"
)

func TestStreamingTicket(t *testing.T) {
	path := "/users/maran/does/not/exist.mkv"
	secret, err := tokenSecret()
	if err != nil {
		t.Errorf("No secret could be generated: %s", err)
	}
	fmt.Println("Secret:", secret)
	token, err := CreateStreamingJWT(1, path)
	if err != nil {
		t.Errorf("Expected error to be nil, got error instead: %s", err)
	}
	newsecret, err := tokenSecret()
	if err != nil {
		t.Errorf("No secret could be generated: %s", err)
	}
	if newsecret != secret {
		t.Errorf("JWT Secret somehow changed, something is wrong! Secret %s and %s", newsecret, secret)
	}

	claim, err := ValidateStreamingJWT(token)
	if err != nil {
		t.Errorf("Could not validate created token: %s", err)
	}
	if claim.FilePath != path {
		t.Errorf("Filepath was not correct in token. Expected %s but got %s", path, claim.FilePath)
	}

	if claim.UserID != 1 {
		t.Errorf("User was not valid expected %d got %d", 1, claim.UserID)
	}

}
