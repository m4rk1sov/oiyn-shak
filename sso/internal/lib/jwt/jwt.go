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
	ErrTokenExpired    = errors.New("token is expired")
	ErrClaimsInvalid   = errors.New("claims are invalid")
	ErrNotRefreshToken = errors.New("token is not refresh token")
	ErrNotAccessToken  = errors.New("token is not access token")
	ErrTokenMalformed  = errors.New("token is malformed")
	ErrInvalidIssuer   = errors.New("token issuer is invalid")
)

type TokenClaims struct {
	UserID      int64    `json:"user_id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Type        string   `json:"type"` // access or refresh
	ExpiresAt   int64    `json:"exp"`
	AppID       int32    `json:"app_id"`
	Permissions []string `json:"permissions,omitempty"`
	Issuer      string   `json:"iss"`
	IssuedAt    int64    `json:"iat"`
	jwt.RegisteredClaims
}

// creates new access JWT token for user and app
func NewToken(user models.User, app models.App, permissions []models.Permission, duration time.Duration) (string, error) {
	now := time.Now()
	
	permissionCodes := make([]string, len(permissions))
	for i, permission := range permissions {
		permissionCodes[i] = permission.Code
	}
	
	claims := TokenClaims{
		UserID:      user.ID,
		Email:       user.Email,
		Name:        user.Name,
		AppID:       app.ID,
		Type:        "access",
		Permissions: permissionCodes,
		Issuer:      fmt.Sprintf("sso-app-%d", app.ID),
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Add(duration).Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    fmt.Sprintf("sso-app-%d", app.ID),
			Subject:   fmt.Sprintf("user-%d", user.ID),
			ID:        fmt.Sprintf("%d-%d-%d", user.ID, app.ID, now.Unix()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	tokenString, err := token.SignedString([]byte(app.Secret))
	if err != nil {
		return "", err
	}
	
	return tokenString, nil
}

func NewRefreshToken(user models.User, app models.App, duration time.Duration) (string, error) {
	now := time.Now()
	
	claims := TokenClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AppID:     app.ID,
		Type:      "refresh",
		Issuer:    fmt.Sprintf("sso-app-%d", app.ID),
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(duration).Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    fmt.Sprintf("sso-app-%d", app.ID),
			Subject:   fmt.Sprintf("user-%d", user.ID),
			ID:        fmt.Sprintf("refresh-%d-%d-%d", user.ID, app.ID, now.Unix()),
		},
	}
	
	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	refreshString, err := refresh.SignedString([]byte(app.Secret))
	if err != nil {
		return "", err
	}
	
	return refreshString, nil
}

func ValidateToken(tokenString string, secret string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected token signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	
	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrClaimsInvalid
	}
	
	if claims.Type != "access" {
		return nil, ErrNotAccessToken
	}
	
	return claims, nil
}

func ValidateRefreshToken(refreshString string, secret string) (*TokenClaims, error) {
	claims, err := ValidateTokenBase(refreshString, secret)
	if err != nil {
		return nil, err
	}
	
	if claims.Type != "refresh" {
		return nil, ErrNotRefreshToken
	}
	
	return claims, nil
}

func ValidateTokenBase(tokenString string, secret string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected token signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	
	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrClaimsInvalid
	}
	
	return claims, nil
}

func DecodeWithoutValidation(tokenString string) (*TokenClaims, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, &TokenClaims{})
	if err != nil {
		return nil, ErrTokenMalformed
	}
	
	claims, ok := token.Claims.(*TokenClaims)
	if !ok {
		return nil, ErrClaimsInvalid
	}
	
	return claims, nil
}

// HasPermission checks for permission in token
func (c *TokenClaims) HasPermission(permissionCode string) bool {
	for _, permission := range c.Permissions {
		if permission == permissionCode {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if the token has any of the permissions
func (c *TokenClaims) HasAnyPermission(permissionCodes ...string) bool {
	for _, required := range permissionCodes {
		if c.HasPermission(required) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the token has all of the permissions
func (c *TokenClaims) HasAllPermissions(permissionCodes ...string) bool {
	for _, required := range permissionCodes {
		if !c.HasPermission(required) {
			return false
		}
	}
	return true
}
