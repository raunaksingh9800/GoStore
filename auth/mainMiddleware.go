package auth

import (
	l "GoStore/log"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func AuthMiddleWare(usr, token string) bool {
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "database connection failed:"+err.Error())
		return false
	}
	defer db.Close()

	var expiresAt time.Time
	var tokenI string

	q := `SELECT token FROM SESSIONS WHERE user = ?;`
	err = db.QueryRow(q, usr).Scan(&tokenI)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		l.LogMessage(l.ERROR, "No User found: "+err.Error())
		return false
	}
	if token == tokenI {
		query := `SELECT expires_at FROM SESSIONS WHERE user = ? `
		err = db.QueryRow(query, usr).Scan(&expiresAt)
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Println("PRB")
				return false
			}
			l.LogMessage(l.ERROR, "No User found: "+err.Error())
			return false
		}
	}

	return !time.Now().After(expiresAt)
}
