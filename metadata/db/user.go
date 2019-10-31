package db

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"time"
)

// CommonModelFields is a list of all fields that should be present on all models.
type CommonModelFields struct {
	ID        uint       `gorm:"primary_key" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

// User defines a user model.
type User struct {
	UUIDable
	CommonModelFields
	Username     string `gorm:"not null;unique" json:"username"`
	Admin        bool   `gorm:"not null" json:"admin"`
	PasswordHash string `gorm:"not null" json:"-"`
	Salt         string `gorm:"not null" json:"-"`
}

// Invite is a model used to invite users to your server.
type Invite struct {
	gorm.Model
	Code   string
	UserID uint
	User   *User
}

// CreateInvite creates an invite code that can be redeemed by new users.
func CreateInvite() Invite {
	invite := Invite{Code: helpers.RandAlphaString(24)}
	db.Save(&invite)

	return invite
}

// AllInvites returns all invites from the db.
func AllInvites() (invites []Invite) {
	db.Find(&invites)
	return invites
}

// ValidPassword checks if the given password is valid for the user.
func (user *User) ValidPassword(password string) bool {
	db.Where("username = ?", user.Username).Find(user)
	if user.hashPassword(password, user.Salt) == user.PasswordHash {
		return true
	}
	return false
}

// SetPassword sets a (new) password for the given user.
func (user *User) SetPassword(password string, salt string) string {
	user.Salt = salt
	user.PasswordHash = user.hashPassword(password, user.Salt)

	return user.PasswordHash
}

func (user *User) hashPassword(password string, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(password))
	hashedStr := hex.EncodeToString(h.Sum(nil))
	return hashedStr
}

// CreateUserWithCode creates a new user. The invite code will be ignored if no other users exist yet.
func CreateUserWithCode(username string, password string, code string) (User, error) {
	count := 0
	db.Table("users").Count(&count)

	invite := Invite{}
	admin := false

	// Not the first user, checking invite.
	if count > 0 {
		db.Where("code = ?", code).First(&invite)

		if (invite.Code == "") || (invite.UserID != 0) {
			log.Warnln("Not a valid code or already used.")
			return User{}, fmt.Errorf("invite code invalid")
		}
	} else {
		admin = true
	}

	user, err := CreateUser(username, password, admin)

	if count > 0 {
		if !db.NewRecord(&user) {
			invite.UserID = user.ID
			db.Save(&invite)
		}
	}
	return user, err
}

// CreateUser creates a new (admin) user to allow access via the web-interface
func CreateUser(username string, password string, admin bool) (User, error) {
	// TODO Maran: Create a way to return all errors at once
	if len(username) < 3 {
		return User{}, fmt.Errorf("username should be at least 3 characters")
	}

	if len(password) < 8 {
		return User{}, fmt.Errorf("password should be at least 8 characters")
	}

	user := User{Username: username, Admin: admin}
	user.SetPassword(password, helpers.RandAlphaString(24))
	dbobj := db.Create(&user)

	return user, dbobj.Error
}

// AllUsers return all users from thedb.
func AllUsers() (users []User) {
	db.Find(&users)
	return users
}

// FindUser returns a specific user.
func FindUser(id uint) (*User, error) {
	var user User
	if err := db.Take(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindUserByUsername returns a specific user.
func FindUserByUsername(username string) (*User, error) {
	var user User
	if err := db.Take(&user, "username = ?", username).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UserCount counts the amount of users in the db.
func UserCount() int {
	count := 0
	db.Find(&User{}).Count(&count)
	return count
}

// DeleteUser deletes the given user.
func DeleteUser(id uint) (User, error) {
	user := User{}
	db.Find(&user, id)

	if user.ID != 0 {
		db.Unscoped().Where("user_id = ?", user.ID).Delete(Invite{})
		obj := db.Unscoped().Delete(&user)
		return user, obj.Error
	}

	return user, fmt.Errorf("user could not be found, not deleted")
}
