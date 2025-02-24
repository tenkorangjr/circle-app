package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var tokenString = os.Getenv("JWT_KEY")

func GenerateJWT(userId int, email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email":   email,
		"user_id": userId,
		"exp":     time.Now().Add(time.Hour * 2).Unix(),
	})

	return token.SignedString([]byte(tokenString))
}
