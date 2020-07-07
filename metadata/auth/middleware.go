package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

type contextKey string

func (c contextKey) String() string {
	return "auth context key " + string(c)
}

var (
	contextKeyUserID  = contextKey("user_id")
	ContextKeyIsAdmin = contextKey("is_admin")
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

// ContextWithUserID returns a context with the given user ID
func ContextWithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, contextKeyUserID, userID)
}

// UserAdmin checks whether the JWT is authorised as admin.
func UserAdmin(ctx context.Context) (bool, bool) {
	isAdmin, ok := ctx.Value(ContextKeyIsAdmin).(bool)
	return isAdmin, ok
}

// MiddleWare checks for user authentication and prevents unauthorised access to the API.
func MiddleWare(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var authHeader, tokenStr string
		authHeader = r.Header.Get("Authorization")

		if authHeader == "" {
			q := r.URL.Query()
			tokenStr = q.Get("JWT")
		} else {
			// Split "Bearer <token>"
			splitHeader := strings.Split(authHeader, " ")
			if len(splitHeader) == 2 {
				tokenStr = splitHeader[1]
			}
		}

		if tokenStr != "" {
			token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, jwtSecretFunc)
			if err != nil {
				writeError(
					fmt.Sprintf("Unauthorized: %s", err.Error()),
					w,
					http.StatusUnauthorized)
				return
			}

			if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
				// Check if the user still exists
				_, err := db.FindUser(claims.UserID)
				if err != nil {
					writeError(
						fmt.Sprintf("Unauthorized: %s", err.Error()),
						w,
						http.StatusUnauthorized)
					return
				}

				log.WithFields(
					log.Fields{
						"username":  claims.Username,
						"userID":    claims.UserID,
						"expiresAt": claims.StandardClaims.ExpiresAt,
					},
				).Debugln("Authenticated with valid JWT")
				ctx := r.Context()
				ctx = context.WithValue(ctx, contextKeyUserID, claims.UserID)
				ctx = context.WithValue(ctx, ContextKeyIsAdmin, claims.Admin)
				h.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		log.Warnln("No authorization header presented.")
		writeError("Unauthorized", w, http.StatusUnauthorized)
		return
	})
}

// TODO Maran: Rotate secrets
func tokenSecret() (string, error) {
	tokenPath := path.Join(helpers.BaseConfigDir(), "token.secret")
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

// CreateMetadataJWT returns a string login JWT.
// TODO Maran: Consider setting the jti if we want to increase security.
func CreateMetadataJWT(user *db.User, validFor time.Duration) (string, error) {
	expiresAt := time.Now().Add(validFor).Unix()

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
