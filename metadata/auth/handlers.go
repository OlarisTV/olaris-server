// Package auth handles various authentication methods for the metadata server.
package auth

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"io/ioutil"
	"net/http"
	"time"
)

const DefaultLoginTokenValidity = 24 * time.Hour

type userRequest struct {
	Username string `json:"username"`
	Code     string `json:"code"`
	Password string `json:"password"`
}
type userRequestRes struct {
	HasError bool   `json:"has_error"`
	Message  string `json:"message"`
}

type tokenResponse struct {
	JWT string `json:"jwt"`
}

// ReadyForSetup checks whether the metadata has been through it's initial setup
// If this returns true a user can be created without an invite code.
func ReadyForSetup(w http.ResponseWriter, r *http.Request) {
	if db.UserCount() == 0 {
		w.Write([]byte("true"))
	} else {
		w.Write([]byte("false"))
	}
}

// UserHandler handles logging in existing users and returning a JWT.
func UserHandler(w http.ResponseWriter, r *http.Request) {
	ur := userRequest{}
	b, err := ioutil.ReadAll(r.Body)

	if err != nil {
		log.Warnln("Could not read incoming request body.")
		return
	}

	if err := json.Unmarshal(b, &ur); err != nil {
		writeError("Could not parse JSON object", w, http.StatusBadRequest)
		return
	}

	if ur.Username == "" {
		writeError("No username supplied", w, http.StatusBadRequest)
		return
	}

	if ur.Password == "" {
		writeError("No password supplied", w, http.StatusBadRequest)
		return
	}

	u := db.User{Username: ur.Username}

	if u.ValidPassword(ur.Password) == true {
		token, err := CreateMetadataJWT(&u, DefaultLoginTokenValidity)
		if err != nil {
			writeError(err.Error(), w, http.StatusUnauthorized)
		} else {
			tokenRes := tokenResponse{JWT: token}
			jtoken, err := json.Marshal(tokenRes)
			if err != nil {
				log.Warnln("Could not marshall JWT token:", err)
			}
			w.Write(jtoken)
		}
	} else {
		writeError("Invalid username or password", w, http.StatusUnauthorized)
	}
}

// CreateUserHandler handles the creation of users, either via invite code or the first admin user.
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	ur := userRequest{}
	b, err := ioutil.ReadAll(r.Body)

	if err != nil {
		log.Warnln("Could not read incoming request body.")
		return
	}

	if err := json.Unmarshal(b, &ur); err != nil {
		log.Warnln("Could not parse request:", err)
		return
	}

	user, err := db.CreateUserWithCode(ur.Username, ur.Password, ur.Code)
	if err != nil {
		writeError(err.Error(), w, http.StatusUnauthorized)
		return
	}

	jre, _ := json.Marshal(user)
	w.Write(jre)
}

func writeError(errStr string, w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	urr := userRequestRes{true, errStr}
	jres, err := json.Marshal(urr)
	if err != nil {
		log.Warnln("How is this for meta errors: There was an error while creating an error:", err)
	}
	w.Write(jres)
}
