package authentication

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type CustomClaims struct {
	Username string
	jwt.StandardClaims
}

func GenerateJWT(username string, secret string) (string, error) {
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &CustomClaims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ValidateJWT(tokenString string, secret string) (bool, error) {
	// Parse and validate the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return false, err
	}

	// Check if the token is valid
	if !token.Valid {
		return false, jwt.NewValidationError("Error: JWT not valid", 1)
	}

	return true, nil
}
