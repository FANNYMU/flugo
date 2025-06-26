package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"flugo.com/config"
	"flugo.com/logger"
	"flugo.com/router"
)

type Claims struct {
	UserID   int                    `json:"user_id"`
	Username string                 `json:"username"`
	Email    string                 `json:"email"`
	Roles    []string               `json:"roles"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
	Exp      int64                  `json:"exp"`
	Iat      int64                  `json:"iat"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type AuthService struct {
	secretKey   []byte
	expTime     time.Duration
	refreshTime time.Duration
}

func NewAuthService(cfg *config.JWTConfig) *AuthService {
	return &AuthService{
		secretKey:   []byte(cfg.Secret),
		expTime:     time.Duration(cfg.ExpirationTime) * time.Second,
		refreshTime: time.Duration(cfg.RefreshTime) * time.Second,
	}
}

var DefaultAuthService *AuthService

func Init(cfg *config.JWTConfig) {
	DefaultAuthService = NewAuthService(cfg)
}

func (a *AuthService) GenerateToken(claims Claims) (*Token, error) {
	now := time.Now()
	claims.Iat = now.Unix()
	claims.Exp = now.Add(a.expTime).Unix()

	accessToken, err := a.createJWT(claims)
	if err != nil {
		return nil, err
	}

	refreshClaims := Claims{
		UserID: claims.UserID,
		Exp:    now.Add(a.refreshTime).Unix(),
		Iat:    now.Unix(),
	}

	refreshToken, err := a.createJWT(refreshClaims)
	if err != nil {
		return nil, err
	}

	return &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(a.expTime.Seconds()),
	}, nil
}

func (a *AuthService) createJWT(claims Claims) (string, error) {
	header := map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, _ := json.Marshal(header)
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerJSON)

	claimsJSON, _ := json.Marshal(claims)
	claimsEncoded := base64.RawURLEncoding.EncodeToString(claimsJSON)

	message := headerEncoded + "." + claimsEncoded
	signature := a.sign(message)

	return message + "." + signature, nil
}

func (a *AuthService) sign(message string) string {
	h := hmac.New(sha256.New, a.secretKey)
	h.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func (a *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	message := parts[0] + "." + parts[1]
	expectedSignature := a.sign(message)

	if parts[2] != expectedSignature {
		return nil, fmt.Errorf("invalid token signature")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid token claims")
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("invalid token claims")
	}

	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token has expired")
	}

	return &claims, nil
}

func (a *AuthService) RefreshToken(refreshTokenString string) (*Token, error) {
	claims, err := a.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	newClaims := Claims{
		UserID: claims.UserID,
	}

	return a.GenerateToken(newClaims)
}

func RequireAuth() router.MiddlewareFunc {
	return func(next router.HandlerFunc) router.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				http.Error(w, "Authorization token required", http.StatusUnauthorized)
				return
			}

			claims, err := DefaultAuthService.ValidateToken(token)
			if err != nil {
				logger.Warn("Invalid token: %v", err)
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			SetCurrentUser(r, claims)
			next(w, r)
		}
	}
}

func RequireRoles(roles ...string) router.MiddlewareFunc {
	return func(next router.HandlerFunc) router.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user := GetCurrentUser(r)
			if user == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			if !hasAnyRole(user.Roles, roles) {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next(w, r)
		}
	}
}

func OptionalAuth() router.MiddlewareFunc {
	return func(next router.HandlerFunc) router.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token != "" {
				if claims, err := DefaultAuthService.ValidateToken(token); err == nil {
					SetCurrentUser(r, claims)
				}
			}
			next(w, r)
		}
	}
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

func hasAnyRole(userRoles, requiredRoles []string) bool {
	for _, required := range requiredRoles {
		for _, userRole := range userRoles {
			if userRole == required {
				return true
			}
		}
	}
	return false
}

type contextKey string

const userContextKey contextKey = "current_user"

func SetCurrentUser(r *http.Request, claims *Claims) {
	ctx := r.Context()
	*r = *r.WithContext(ctx)
	r.Header.Set("X-Current-User", fmt.Sprintf("%d", claims.UserID))
}

func GetCurrentUser(r *http.Request) *Claims {
	userID := r.Header.Get("X-Current-User")
	if userID == "" {
		return nil
	}

	token := extractToken(r)
	if token == "" {
		return nil
	}

	claims, err := DefaultAuthService.ValidateToken(token)
	if err != nil {
		return nil
	}

	return claims
}

func GetCurrentUserID(r *http.Request) int {
	user := GetCurrentUser(r)
	if user == nil {
		return 0
	}
	return user.UserID
}

func GenerateToken(claims Claims) (*Token, error) {
	if DefaultAuthService == nil {
		return nil, fmt.Errorf("auth service not initialized")
	}
	return DefaultAuthService.GenerateToken(claims)
}

func ValidateToken(token string) (*Claims, error) {
	if DefaultAuthService == nil {
		return nil, fmt.Errorf("auth service not initialized")
	}
	return DefaultAuthService.ValidateToken(token)
}

func RefreshToken(refreshToken string) (*Token, error) {
	if DefaultAuthService == nil {
		return nil, fmt.Errorf("auth service not initialized")
	}
	return DefaultAuthService.RefreshToken(refreshToken)
}

// JWTConfig is an alias for config.JWTConfig for backward compatibility
type JWTConfig struct {
	Secret         string
	ExpirationTime int
	RefreshTime    int
}
