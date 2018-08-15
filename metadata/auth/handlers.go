package auth

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"io/ioutil"
	"net/http"
)

type UserRequest struct {
	Username string `json:"username"`
	Code     string `json:"code"`
	Password string `json:"password"`
}
type UserRequestRes struct {
	HasError bool   `json:"has_error"`
	Message  string `json:"message"`
}

type TokenResponse struct {
	JWT string `json:"jwt"`
}

func ReadyForSetup(w http.ResponseWriter, r *http.Request) {
	if db.UserCount() == 0 {
		w.Write([]byte("true"))
	} else {
		w.Write([]byte("false"))
	}
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	ur := UserRequest{}
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
		token, err := createJWT(&u)
		if err != nil {
			writeError(err.Error(), w, http.StatusUnauthorized)
		} else {
			tokenRes := TokenResponse{JWT: token}
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

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	ur := UserRequest{}
	b, err := ioutil.ReadAll(r.Body)

	if err != nil {
		log.Warnln("Could not read incoming request body.")
		return
	}

	if err := json.Unmarshal(b, &ur); err != nil {
		log.Warnln("Could not parse request:", err)
		return
	}

	if ur.Code == "" {
		writeError("No invite code supplied", w, http.StatusBadRequest)
		return
	}

	user, err := db.CreateUser(ur.Username, ur.Password, ur.Code)
	if err != nil {
		writeError(err.Error(), w, http.StatusUnauthorized)
		return
	} else {
		jre, _ := json.Marshal(user)
		w.Write(jre)

	}
}

func writeError(errStr string, w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	urr := UserRequestRes{true, errStr}
	jres, err := json.Marshal(urr)
	if err != nil {
		log.Warnln("How is this for meta errors: There was an error while creating an error:", err)
	}
	w.Write(jres)
}
