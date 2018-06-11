package db

import (
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"net/http"
	"strings"
)

func AuthMiddleWare(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := 0
		ctx.Db.Find(&User{}).Count(&count)
		if count == 0 {
			fmt.Println("No users present, no auth required")
			h.ServeHTTP(w, r)
		} else {
			fmt.Println("Users present Auth required")
			var authHeader string
			authHeader = r.Header.Get("Authorization")
			if authHeader != "" {
				tokenStr := strings.Split(authHeader, " ")[1]
				token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
					secret, err := TokenSecret()
					return []byte(secret), err
				})
				if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
					fmt.Printf("%v %v", claims.Login, claims.StandardClaims.ExpiresAt)
					h.ServeHTTP(w, r)
					return
				} else {
					fmt.Println(err)
				}
			}
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
		}
	})
}

func AuthHandler(w http.ResponseWriter, r *http.Request) {
	ur := UserRequest{}
	b, err := ioutil.ReadAll(r.Body)

	if err != nil {
		fmt.Println("Could not read request body")
		return
	}

	if err := json.Unmarshal(b, &ur); err != nil {
		WriteError("Could not parse JSON object", w, http.StatusBadRequest)
		return
	}

	if ur.Login == "" {
		WriteError("No login supplied", w, http.StatusBadRequest)
		return
	}

	if ur.Password == "" {
		WriteError("No password supplied", w, http.StatusBadRequest)
		return
	}

	u := User{Login: ur.Login}

	if u.ValidPassword(ur.Password) == true {
		token, err := u.CreateJWT()
		if err != nil {
			WriteError(err.Error(), w, http.StatusUnauthorized)
		} else {
			tokenRes := TokenResponse{JWT: token}
			jtoken, err := json.Marshal(tokenRes)
			if err != nil {
				fmt.Println("error during token creation josn :p")
			}
			w.Write(jtoken)
		}
	} else {
		WriteError("Invalid username or password", w, http.StatusUnauthorized)
	}
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	ur := UserRequest{}
	b, err := ioutil.ReadAll(r.Body)

	if err != nil {
		fmt.Println("Could not read request body")
		return
	}

	if err := json.Unmarshal(b, &ur); err != nil {
		fmt.Println("Could not parse request")
		return
	}

	if ur.Code == "" {
		WriteError("No invite code supplied", w, http.StatusBadRequest)
		return
	}

	user, err := CreateUser(ur.Login, ur.Password, ur.Admin, ur.Code)
	if err != nil {
		WriteError(err.Error(), w, http.StatusUnauthorized)
		return
	} else {
		jre, _ := json.Marshal(user)
		w.Write(jre)

	}
}

type UserRequest struct {
	Login    string `json:"login"`
	Admin    bool   `json:"admin"`
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

func WriteError(errStr string, w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	urr := UserRequestRes{true, errStr}
	jres, err := json.Marshal(urr)
	if err != nil {
		fmt.Println("error during error creation :p")
	}
	w.Write(jres)
}
