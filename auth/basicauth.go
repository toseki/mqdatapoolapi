package auth

import (
	"crypto/sha256"
	"fmt"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	log "github.com/sirupsen/logrus"
	"github.com/toseki/mqttdatapoolapi/storage"
)

// BasicAuth username password check
func BasicAuth(sqlh storage.MysqlHandler) echo.MiddlewareFunc {
	return middleware.BasicAuth(func(username, password string) bool {
		var u storage.UserInfo
		u, err := sqlh.UserAuth(username)
		if err != nil {
			log.Error("auth/BasicAuth: sql UserAuth error:", err)
			return false
		}

		authchkhex := fmt.Sprintf("%x", sha256.Sum256([]byte(u.Salt+password)))

		return username == u.Username && u.Password == authchkhex
	})
}
