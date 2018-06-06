package db

import "testing"

func TestSetPassword(t *testing.T) {
	user := User{Login: "animazing", Admin: true}
	r := user.SetPassword("test", "test")
	if r != "37268335dd6931045bdcdf92623ff819a64244b53d0e746d438797349d4da578" {
		t.Errorf("Expected salted password to be 37268335dd6931045bdcdf92623ff819a64244b53d0e746d438797349d4da578 got %s instead", r)
	}
	if user.PasswordHash == "" {
		t.Errorf("Password hash is not set on user")
	}
}
