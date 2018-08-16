package auth

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
	"time"
)

// StreamingClaims is a custom JWT that allows filesystem access to files for a certain timespan.
type StreamingClaims struct {
	UserID   uint
	FilePath string
	jwt.StandardClaims
}

// CreateStreamingJWT creates a new JWT that will give permission to stream certain media for a certain timespan.
func CreateStreamingJWT(userID uint, filePath string) (string, error) {
	expiresAt := time.Now().Add(time.Hour * 8).Unix()

	claims := StreamingClaims{
		userID,
		filePath,
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

// ValidateStreamingJWT validates whether a JWT is still valid and allows access to the requested file.
func ValidateStreamingJWT(tokenStr string) (*StreamingClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &StreamingClaims{}, jwtSecretFunc)
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*StreamingClaims); ok && token.Valid {
		log.Debugf("Incoming streaming ticket for '%v' (User %v). Expires at: %v", claims.FilePath, claims.UserID, claims.StandardClaims.ExpiresAt)
		return claims, nil
	}

	return nil, fmt.Errorf("could not validate ticket")
}

func jwtSecretFunc(token *jwt.Token) (interface{}, error) {
	secret, err := tokenSecret()
	return []byte(secret), err
}
