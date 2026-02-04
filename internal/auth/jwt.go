package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	claimSubject     = "sub"
	claimUserID      = "user_id"
	claimType        = "typ"
	claimBotID       = "bot_id"
	claimPlatform    = "platform"
	claimReplyTarget = "reply_target"
	claimSessionID   = "session_id"
	claimContactID   = "contact_id"
	sessionTokenType = "channel_session"
)

// JWTMiddleware returns a JWT auth middleware configured for HS256 tokens.
func JWTMiddleware(secret string, skipper middleware.Skipper) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey:    []byte(secret),
		SigningMethod: "HS256",
		TokenLookup:   "header:Authorization:Bearer ",
		Skipper:       skipper,
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return jwt.MapClaims{}
		},
	})
}

// UserIDFromContext extracts the user id from JWT claims.
func UserIDFromContext(c echo.Context) (string, error) {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok || token == nil || !token.Valid {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "invalid token claims")
	}
	if userID := claimString(claims, claimUserID); userID != "" {
		return userID, nil
	}
	if userID := claimString(claims, claimSubject); userID != "" {
		return userID, nil
	}
	return "", echo.NewHTTPError(http.StatusUnauthorized, "user id missing")
}

// GenerateToken creates a signed JWT for the user.
func GenerateToken(userID, secret string, expiresIn time.Duration) (string, time.Time, error) {
	if strings.TrimSpace(userID) == "" {
		return "", time.Time{}, fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(secret) == "" {
		return "", time.Time{}, fmt.Errorf("jwt secret is required")
	}
	if expiresIn <= 0 {
		return "", time.Time{}, fmt.Errorf("jwt expires in must be positive")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(expiresIn)
	claims := jwt.MapClaims{
		claimSubject: userID,
		claimUserID:  userID,
		"iat":        now.Unix(),
		"exp":        expiresAt.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

type SessionToken struct {
	BotID       string
	Platform    string
	ReplyTarget string
	SessionID   string
	ContactID   string
}

// GenerateSessionToken creates a signed JWT for channel session reply.
func GenerateSessionToken(info SessionToken, secret string, expiresIn time.Duration) (string, time.Time, error) {
	if strings.TrimSpace(info.BotID) == "" {
		return "", time.Time{}, fmt.Errorf("bot id is required")
	}
	if strings.TrimSpace(info.Platform) == "" {
		return "", time.Time{}, fmt.Errorf("platform is required")
	}
	if strings.TrimSpace(info.ReplyTarget) == "" {
		return "", time.Time{}, fmt.Errorf("reply target is required")
	}
	if strings.TrimSpace(secret) == "" {
		return "", time.Time{}, fmt.Errorf("jwt secret is required")
	}
	if expiresIn <= 0 {
		return "", time.Time{}, fmt.Errorf("jwt expires in must be positive")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(expiresIn)
	claims := jwt.MapClaims{
		claimType:        sessionTokenType,
		claimBotID:       info.BotID,
		claimPlatform:    info.Platform,
		claimReplyTarget: info.ReplyTarget,
		claimSessionID:   info.SessionID,
		claimContactID:   info.ContactID,
		"iat":            now.Unix(),
		"exp":            expiresAt.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// SessionTokenFromContext extracts the session token claims from context.
func SessionTokenFromContext(c echo.Context) (SessionToken, error) {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok || token == nil || !token.Valid {
		return SessionToken{}, echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return SessionToken{}, echo.NewHTTPError(http.StatusUnauthorized, "invalid token claims")
	}
	if claimString(claims, claimType) != sessionTokenType {
		return SessionToken{}, echo.NewHTTPError(http.StatusUnauthorized, "invalid session token")
	}
	return SessionToken{
		BotID:       claimString(claims, claimBotID),
		Platform:    claimString(claims, claimPlatform),
		ReplyTarget: claimString(claims, claimReplyTarget),
		SessionID:   claimString(claims, claimSessionID),
		ContactID:   claimString(claims, claimContactID),
	}, nil
}

func claimString(claims jwt.MapClaims, key string) string {
	raw, ok := claims[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprint(raw)
	}
}
