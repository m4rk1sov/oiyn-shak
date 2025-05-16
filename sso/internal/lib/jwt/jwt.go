package jwt

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"sso/internal/domain/models"
	"time"
)

var (
	ErrTokenInvalid    = errors.New("token is invalid")
	ErrClaimsInvalid   = errors.New("claims are invalid")
	ErrNotRefreshToken = errors.New("token is not refresh")
	//ErrTokenMalformed  = errors.New("token is malformed")
	//jwtSecretKey       = []byte(os.Getenv("SECRET_KEY")) // Secure this properly
)

// creates new JWT token for user and app
func NewToken(user models.User, app models.App, duration time.Duration) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	// info inside token
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = user.ID
	claims["email"] = user.Email
	claims["type"] = "access"
	claims["exp"] = time.Now().Add(duration).Unix()
	claims["app_id"] = app.ID

	tokenString, err := token.SignedString([]byte(app.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func NewRefreshToken(user models.User, app models.App, duration time.Duration) (string, error) {
	refresh := jwt.New(jwt.SigningMethodHS256)

	claims := refresh.Claims.(jwt.MapClaims)
	claims["user_id"] = user.ID
	claims["email"] = user.Email
	claims["type"] = "refresh"
	claims["exp"] = time.Now().Add(duration).Unix()
	claims["app_id"] = app.ID

	refreshString, err := refresh.SignedString([]byte(app.Secret))
	if err != nil {
		return "", err
	}

	return refreshString, nil
}

func ValidateToken(tokenString string, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected token signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrTokenInvalid
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrClaimsInvalid
	}

	return claims, nil
}

func ValidateRefreshToken(refreshString string, secret string) (jwt.MapClaims, error) {
	claims, err := ValidateToken(refreshString, secret)
	if err != nil {
		return nil, err
	}

	if claims["type"] != "refresh" {
		return nil, ErrNotRefreshToken
	}

	return claims, nil
}
