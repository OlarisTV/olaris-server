package auth

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"time"
)

type StreamingClaims struct {
	UserID   uint
	FilePath string
	jwt.StandardClaims
}

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

	return fmt.Sprintf("/s/jwt/%s/hls-manifest.m3u8", ss), nil
}

func ValidateStreamingJWT(tokenStr string) (*StreamingClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &StreamingClaims{}, jwtSecretFunc)
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*StreamingClaims); ok && token.Valid {
		fmt.Printf("%v %v Expires at: %v\n", claims.FilePath, claims.UserID, claims.StandardClaims.ExpiresAt)
		return claims, nil
	} else {
		return nil, fmt.Errorf("Could not validate ticket.")
	}
}

func jwtSecretFunc(token *jwt.Token) (interface{}, error) {
	secret, err := tokenSecret()
	return []byte(secret), err
}
