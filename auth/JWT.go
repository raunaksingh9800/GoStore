package auth

import (
	l "GoStore/log"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var jwtKey = []byte("my_secret_key")

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func GenerateTokenJWT(usr, pwd string) (string, error) {
	expirationTime := time.Now().Add(100 * time.Hour)
	claims := &Claims{
		Username: usr,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		l.LogMessage(l.ERROR, "Token signing failed: "+err.Error())
		return "", err
	}

	return tokenString, nil
}

func AuthenticateTokenJWT(tokenString string) (bool, *Claims) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			l.LogMessage(l.ERROR, "Invalid token signature: "+err.Error())
			return false, nil
		}
		l.LogMessage(l.ERROR, "Token parsing failed: "+err.Error())
		l.LogMessage(l.INFO, tokenString)
		return false, nil
	}

	if !token.Valid {
		l.LogMessage(l.ERROR, "Invalid token")
		return false, nil
	}

	return true, claims
}
