package auth

import (
	"net/http"
	"fmt"
	"strings"
	"github.com/dgrijalva/jwt-go"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"github.com/golang/glog"
	"path"
	"gitlab.com/bytesized/bytesized-streaming/helpers"
	"io/ioutil"
	"time"
	"math/rand"
)

type UserClaims struct {
	Login string `json:"login"`
	Admin bool   `json:"admin"`
	jwt.StandardClaims
}

func AuthMiddleWare(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if db.UserCount() == 0 {
			glog.Warning("No users present, no auth required!")
			h.ServeHTTP(w, r)
		} else {
			glog.Info("Users present Auth required")
			var authHeader string
			authHeader = r.Header.Get("Authorization")
			if authHeader != "" {
				tokenStr := strings.Split(authHeader, " ")[1]
				token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
					secret, err := tokenSecret()
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

// TODO Maran: Rotate secrets
func tokenSecret() (string, error) {
	tokenPath := path.Join(helpers.GetHome(), ".config", "bss", "token.secret")
	if helpers.FileExists(tokenPath) {
		secret, err := ioutil.ReadFile(tokenPath)
		if err != nil {
			return "", err
		} else {
			return string(secret), nil
		}
	} else {
		secret := randString(32)
		ioutil.WriteFile(tokenPath, []byte(secret), 0700)
		return secret, nil
	}
}

// TODO Maran: Consider setting the jti if we want to increase security.
func createJWT(user *db.User) (string, error) {
	expiresAt := time.Now().Add(time.Hour * 24).Unix()

	claims := UserClaims{
		user.Login,
		user.Admin,
		jwt.StandardClaims{ExpiresAt: expiresAt, Issuer: "bss"},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := tokenSecret()
	if err != nil {
		return "", err
	}

	ss, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return ss, nil
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
