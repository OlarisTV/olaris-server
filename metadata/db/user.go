package db

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/jinzhu/gorm"
	"math/rand"
)

type User struct {
	UUIDable
	gorm.Model
	Login        string `gorm:"not null;unique"`
	Admin        bool   `gorm:"not null"`
	PasswordHash string `gorm:"not null"`
	Salt         string `gorm:"not null"`
}

func (self *User) SetPassword(password string, salt string) string {
	self.Salt = salt
	h := sha256.New()
	h.Write([]byte(self.Salt))
	h.Write([]byte(password))
	self.PasswordHash = hex.EncodeToString(h.Sum(nil))

	return self.PasswordHash
}

// TODO Maran: Create a way to return all errors at once
func CreateUser(login string, password string, admin bool) (User, error) {
	if len(login) < 3 {
		return User{}, fmt.Errorf("Login should be at least 3 characters")
	}

	if len(password) < 8 {
		return User{}, fmt.Errorf("Password should be at least 8 characters")
	}

	user := User{Login: login, Admin: admin}
	user.SetPassword(password, randString(24))
	dbobj := ctx.Db.Create(&user)
	return user, dbobj.Error
}

func AllUsers() (users []User) {
	ctx.Db.Find(&users)
	return users
}

// This is so we can invite users later
type InviteLink struct {
	Code string
}

// Plucked from https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
)

func randString(n int) string {
	b := make([]byte, n)
	for i := 0; i < n; {
		if idx := int(rand.Int63() & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i++
		}
	}
	return string(b)
}
