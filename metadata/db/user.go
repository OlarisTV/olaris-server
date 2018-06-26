package db

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/jinzhu/gorm"
	"gitlab.com/bytesized/bytesized-streaming/helpers"
	"time"
)

type CommonModelFields struct {
	ID        uint       `gorm:"primary_key" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type User struct {
	UUIDable
	CommonModelFields
	Login        string `gorm:"not null;unique" json:"login"`
	Admin        bool   `gorm:"not null" json:"admin"`
	PasswordHash string `gorm:"not null" json:"-"`
	Salt         string `gorm:"not null" json:"-"`
}

type Invite struct {
	gorm.Model
	Code   string
	UserID uint
	User   User
}

func (self *User) ValidPassword(password string) bool {
	env.Db.Where("login = ?", self.Login).Find(self)
	if self.HashPassword(password, self.Salt) == self.PasswordHash {
		return true
	} else {
		return false
	}
}

func (self *User) SetPassword(password string, salt string) string {
	self.Salt = salt
	self.PasswordHash = self.HashPassword(password, self.Salt)

	return self.PasswordHash
}

func (self *User) HashPassword(password string, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(password))
	hashedStr := hex.EncodeToString(h.Sum(nil))
	return hashedStr
}

// TODO Maran: Create a way to return all errors at once
func CreateUser(login string, password string, admin bool, code string) (User, error) {
	invite := Invite{}
	if len(login) < 3 {
		return User{}, fmt.Errorf("Login should be at least 3 characters")
	}

	if len(password) < 8 {
		return User{}, fmt.Errorf("Password should be at least 8 characters")
	}

	if code != "" {
		count := 0
		env.Db.Where("code = ? and user_id IS NULL", code).Find(&invite).Count(&count)
		if count != 0 {
			fmt.Println("Valid and unused code, creating account")
		} else {
			fmt.Println("Not a valid code or already used")
			return User{}, fmt.Errorf("Invite code invalid")
		}
	}

	user := User{Login: login, Admin: admin}
	user.SetPassword(password, helpers.RandAlphaString(24))
	dbobj := env.Db.Create(&user)
	if !env.Db.NewRecord(&user) {
		invite.UserID = user.ID
		env.Db.Save(&invite)
	}
	return user, dbobj.Error
}

func AllUsers() (users []User) {
	env.Db.Find(&users)
	return users
}

func UserCount() int {
	count := 0
	env.Db.Find(&User{}).Count(&count)
	return count
}
