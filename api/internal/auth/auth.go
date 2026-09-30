package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/samuelfabel/megumi-kura/api/internal/user"
)

const CookieName = "mk_session"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

type Service struct {
	Users        *user.Repository
	Secret       []byte
	SessionHours int
	CookieSecure bool
}

type SessionUser struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

func (s *Service) Login(c *gin.Context, username, password string) (SessionUser, error) {
	u, err := s.Users.FindByUsername(c.Request.Context(), username)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return SessionUser{}, ErrInvalidCredentials
		}
		return SessionUser{}, err
	}
	if !u.Active {
		return SessionUser{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return SessionUser{}, ErrInvalidCredentials
	}

	token, err := s.issueToken(u.ID)
	if err != nil {
		return SessionUser{}, err
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.CookieSecure,
		MaxAge:   s.SessionHours * 3600,
	})
	return SessionUser{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName}, nil
}

func (s *Service) Logout(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.CookieSecure,
		MaxAge:   -1,
	})
}

func (s *Service) Me(c *gin.Context) (SessionUser, error) {
	u, ok := CurrentUser(c)
	if !ok {
		return SessionUser{}, ErrUnauthorized
	}
	return u, nil
}

func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(CookieName)
		if err != nil || token == "" {
			// Also accept Authorization: Bearer
			authz := c.GetHeader("Authorization")
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
				token = strings.TrimSpace(authz[7:])
			}
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userID, err := s.parseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		u, err := s.Users.FindByID(c.Request.Context(), userID)
		if err != nil || !u.Active {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set("user", SessionUser{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName})
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (SessionUser, bool) {
	v, ok := c.Get("user")
	if !ok {
		return SessionUser{}, false
	}
	u, ok := v.(SessionUser)
	return u, ok
}

func (s *Service) issueToken(userID int64) (string, error) {
	// Signed session token (NOT JWT): base64url("id|exp") + "." + HMAC-SHA256
	exp := time.Now().UTC().Add(time.Duration(s.SessionHours) * time.Hour).Unix()
	payload := fmt.Sprintf("%d|%d", userID, exp)
	mac := hmac.New(sha256.New, s.Secret)
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig, nil
}

func (s *Service) parseToken(token string) (int64, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return 0, ErrUnauthorized
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, ErrUnauthorized
	}
	mac := hmac.New(sha256.New, s.Secret)
	_, _ = mac.Write(payloadBytes)
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return 0, ErrUnauthorized
	}
	fields := strings.Split(string(payloadBytes), "|")
	if len(fields) != 2 {
		return 0, ErrUnauthorized
	}
	exp, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || time.Now().UTC().Unix() > exp {
		return 0, ErrUnauthorized
	}
	id, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, ErrUnauthorized
	}
	return id, nil
}

// HashPassword is exported for tests / tooling. Never log the plaintext.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}
