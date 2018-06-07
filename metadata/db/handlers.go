package db

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
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
	r.ParseForm()

	u := User{Login: r.Form["login"][0]}

	if u.ValidPassword(r.Form["password"][0]) == true {
		token, err := u.CreateJWT()
		if err != nil {
			w.Write([]byte(err.Error()))
		} else {
			w.Write([]byte(token))
		}
	}
}
