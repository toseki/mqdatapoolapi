package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

type MysqlHandler interface {
	UserAuth(username string) (UserInfo, error)
	CloseSQLHandler()
}

type SQLHandler struct {
	sql       *sql.DB
	usertable string
	authcache *cache.Cache
}

type UserInfo struct {
	Username string
	Salt     string
	Password string
}

func NewSQLHandler(mysqlparam string, usertable string) MysqlHandler {
	db, err := sql.Open("mysql", mysqlparam)
	if err != nil {
		panic(err.Error())
	}
	c := cache.New(5*time.Minute, 10*time.Minute)

	h := SQLHandler{
		sql:       db,
		usertable: usertable,
		authcache: c,
	}
	return &h
}

func (h *SQLHandler) CloseSQLHandler() {
	h.sql.Close()
}

func (h *SQLHandler) UserAuth(username string) (UserInfo, error) {
	var u UserInfo

	cachedata, found := h.authcache.Get(username)
	if found {
		u = cachedata.(UserInfo)
		log.Debug("found in cache:", username)
		return u, nil
	}

	dbquery, err := h.sql.Prepare(fmt.Sprintf("SELECT username,salt,password FROM %s WHERE username=? LIMIT 1", h.usertable))
	if err != nil {
		log.Error("storage/sql: Query Prepare error:", err)
	}

	err = dbquery.QueryRow(username).Scan(&u.Username, &u.Salt, &u.Password)
	if err == sql.ErrNoRows {
		log.Error("storage/sql: Query ErrNoRows:", err)
		return u, err
	}
	if err != nil {
		log.Error("storage/sql: Query Error:", err)
		return u, err
	}

	h.authcache.Set(username, u, cache.DefaultExpiration)
	return u, nil
}
