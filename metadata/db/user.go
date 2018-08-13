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
	User   *User
}

func CreateInvite() Invite {
	invite := Invite{Code: helpers.RandAlphaString(24)}
	env.Db.Save(&invite)

	return invite
}

func AllInvites() (invites []Invite) {
	env.Db.Find(&invites)
	return invites
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
func CreateUser(login string, password string, code string) (User, error) {
	if len(login) < 3 {
		return User{}, fmt.Errorf("Login should be at least 3 characters")
	}

	if len(password) < 8 {
		return User{}, fmt.Errorf("Password should be at least 8 characters.")
	}

	count := 0
	env.Db.Table("users").Count(&count)

	invite := Invite{}
	admin := false

	// Not the first user, checking invite.
	if count > 0 {
		env.Db.Where("code = ?", code).First(&invite)

		if (invite.Code == "") || (invite.UserID != 0) {
			fmt.Println("Not a valid code or already used.")
			return User{}, fmt.Errorf("Invite code invalid.")
		}
	} else {
		admin = true
	}

	user := User{Login: login, Admin: admin}
	user.SetPassword(password, helpers.RandAlphaString(24))
	dbobj := env.Db.Create(&user)

	if count > 0 {
		if !env.Db.NewRecord(&user) {
			invite.UserID = user.ID
			env.Db.Save(&invite)
		}
	}
	return user, dbobj.Error
}

func AllUsers() (users []User) {
	env.Db.Find(&users)
	return users
}

func FindUser(id uint) (user User) {
	env.Db.Find(&user, id)
	return user
}

func UserCount() int {
	count := 0
	env.Db.Find(&User{}).Count(&count)
	return count
}

func DeleteUser(id int) (User, error) {
	user := User{}
	env.Db.Find(&user, id)

	if user.ID != 0 {
		obj := env.Db.Unscoped().Delete(&user)
		return user, obj.Error
	}
	return user, fmt.Errorf("User could not be found, not deleted.")
}
