package auth

import (
	l "GoStore/log"
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func GenerateToken(usr, pwd string) (string, error) {
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "Database connection failed: "+err.Error())
		return "", err
	}
	defer db.Close()

	sessionToken, err := bcrypt.GenerateFromPassword([]byte(usr+pwd), bcrypt.DefaultCost)
	if err != nil {
		l.LogMessage(l.ERROR, "Session token generation failed: "+err.Error())
		return "", err
	}

	// Store session token with expiration time
	expirationTime := time.Now().Add(2 * time.Hour) // Set expiration time to 2 hours
	stmt, err := db.Prepare(`INSERT INTO SESSIONS (user, token, expires_at) VALUES (?, ?, ?)`)
	if err != nil {
		l.LogMessage(l.ERROR, "Statement preparation failed: "+err.Error())
		return "", err
	}
	defer stmt.Close()

	_, err = stmt.Exec(usr, sessionToken, expirationTime)
	if err != nil {
		l.LogMessage(l.ERROR, "Insert query failed: "+err.Error())
		return "", err
	}

	return string(sessionToken), nil
}
