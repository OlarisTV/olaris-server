package auth

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"time"
)

type contextKey string

func (c contextKey) String() string {
	return "auth context key " + string(c)
}

var (
	contextKeyUserID  = contextKey("user_id")
	contextKeyIsAdmin = contextKey("is_admin")
)

// UserClaims defines our custom JWT.
type UserClaims struct {
	Username string `json:"username"`
	UserID   uint   `json:"user_id"`
	Admin    bool   `json:"admin"`
	jwt.StandardClaims
}

// UserID extracts user id from context.
func UserID(ctx context.Context) (uint, bool) {
	userID, ok := ctx.Value(contextKeyUserID).(uint)
	return userID, ok
}

// UserAdmin checks whether the JWT is authorised as admin.
func UserAdmin(ctx context.Context) (bool, bool) {
	isAdmin, ok := ctx.Value(contextKeyIsAdmin).(bool)
	return isAdmin, ok
}

// MiddleWare checks for user authentication and prevents unauthorised access to the API.
func MiddleWare(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if db.UserCount() == 0 {
			log.Warnln("No users present, no auth required!")
			h.ServeHTTP(w, r)
		} else {
			log.Debugln("Users present auth required from this point on.")
			var authHeader string
			authHeader = r.Header.Get("Authorization")
			if authHeader != "" {
				tokenStr := strings.Split(authHeader, " ")[1]
				token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, jwtSecretFunc)
				if err != nil {
					writeError(fmt.Sprintf("Unauthorized: %s", err.Error()), w, http.StatusUnauthorized)
				}

				if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
					log.Debugf("User: '%v' (%v). Expires at: %v.", claims.Username, claims.UserID, claims.StandardClaims.ExpiresAt)
					ctx := r.Context()
					ctx = context.WithValue(ctx, contextKeyUserID, claims.UserID)
					ctx = context.WithValue(ctx, contextKeyIsAdmin, claims.Admin)
					h.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				writeError("Unauthorized", w, http.StatusUnauthorized)
			}
		}
	})
}

// TODO Maran: Rotate secrets
func tokenSecret() (string, error) {
	tokenPath := path.Join(helpers.BaseConfigPath(), "token.secret")
	err := helpers.EnsurePath(path.Dir(tokenPath))
	if err != nil {
		return "", err
	}
	if helpers.FileExists(tokenPath) {
		secret, err := ioutil.ReadFile(tokenPath)
		if err != nil {
			return "", err
		}
		return string(secret), nil
	}

	secret := helpers.RandAlphaString(32)
	err = ioutil.WriteFile(tokenPath, []byte(secret), 0700)
	return secret, err
}

// TODO Maran: Consider setting the jti if we want to increase security.
func createJWT(user *db.User) (string, error) {
	expiresAt := time.Now().Add(time.Hour * 24).Unix()

	claims := UserClaims{
		user.Username,
		user.ID,
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
