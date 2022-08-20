package common

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func NewJWT(userId int, ttl time.Duration, key string) (jwtToken string, err error) {
	// Creating a new token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(ttl)},
		Subject:   strconv.Itoa(userId),
	})

	// Token signing
	return token.SignedString([]byte(key))
}

func VerifyToken(tokenString, key string) int {
	// Token decryption
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Header["alg"])
		}

		return []byte(key), nil
	})
	if err != nil {
		return 0
	}

	// Retrieving data from a token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0
	}
	
	userId, err := strconv.Atoi(claims["sub"].(string))
	if err != nil {
		return 0
	}
	
	return userId
}