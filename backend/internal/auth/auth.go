// Package auth 提供 JWT 签发与校验。
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	jwtlib "github.com/golang-jwt/jwt/v5"

	"gost-panel/internal/resp"
)

// Manager 持有 JWT 密钥与过期时间。
type Manager struct {
	Secret      string
	ExpireHours int
}

// LoadSecret 读取 JWT 密钥：JWT_SECRET 环境变量优先，
// 否则读 data/jwt_secret，不存在则生成并持久化（重启后 token 不失效）。
func LoadSecret(dataDir string) string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	secretFile := filepath.Join(dataDir, "jwt_secret")
	if b, err := os.ReadFile(secretFile); err == nil {
		if s := strings.TrimSpace(string(b)); s != "" {
			return s
		}
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("生成随机密钥失败: %v", err)
	}
	s := hex.EncodeToString(b)
	_ = os.MkdirAll(dataDir, 0o700)
	if err := os.WriteFile(secretFile, []byte(s), 0o600); err != nil {
		log.Printf("写入 jwt_secret 失败: %v", err)
	}
	return s
}

// CreateToken 为用户签发 HS256 JWT。
func (m *Manager) CreateToken(username string) (string, error) {
	claims := jwtlib.MapClaims{
		"sub": username,
		"exp": time.Now().Add(time.Duration(m.ExpireHours) * time.Hour).Unix(),
	}
	return jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte(m.Secret))
}

// Middleware 返回 Gin 认证中间件，通过后把用户名放入 c.Get("user")。
func (m *Manager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			abort(c, "无效凭证")
			return
		}
		tokenStr := strings.TrimPrefix(h, "Bearer ")
		tok, err := jwtlib.Parse(tokenStr, func(t *jwtlib.Token) (any, error) {
			return []byte(m.Secret), nil
		}, jwtlib.WithValidMethods([]string{"HS256"}))
		if err != nil || !tok.Valid {
			if errors.Is(err, jwtlib.ErrTokenExpired) {
				abort(c, "登录已过期")
			} else {
				abort(c, "无效凭证")
			}
			return
		}
		claims, ok := tok.Claims.(jwtlib.MapClaims)
		if !ok {
			abort(c, "无效凭证")
			return
		}
		sub, _ := claims.GetSubject()
		c.Set("user", sub)
		c.Next()
	}
}

func abort(c *gin.Context, msg string) {
	c.Abort()
	resp.ErrUnauthorized(c, msg)
}
