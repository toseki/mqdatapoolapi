package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

type MysqlHandler interface {
	UserAuth(username string) (UserInfo, error)
	CloseSQLHandler()
}

type SQLHandler struct {
	sql       *sql.DB
	usertable string
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
	h := SQLHandler{
		sql:       db,
		usertable: usertable,
	}
	return &h
}

func (h *SQLHandler) CloseSQLHandler() {
	h.sql.Close()
}

func (h *SQLHandler) UserAuth(username string) (UserInfo, error) {
	var u UserInfo

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
	return u, nil
}
