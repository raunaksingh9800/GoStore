package client

import (
	"GoStore/auth"
	"GoStore/log"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func Login(usr, pwd string) (string, string, error) {
	log.LogMessage(log.SUCS, "Working client")

	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		log.LogMessage(log.ERROR, "Database connection failed: "+err.Error())
		return "", "", err
	}
	defer db.Close()

	var storedPwd, userUID string
	err = db.QueryRow("SELECT UID, pwd FROM users WHERE username = ?", usr).Scan(&userUID, &storedPwd)
	if err != nil {
		if err == sql.ErrNoRows {
			log.LogMessage(log.ERROR, "User not found")
			return "", "", fmt.Errorf("user not found")
		}
		log.LogMessage(log.ERROR, "Query failed: "+err.Error())
		return "", "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPwd), []byte(pwd))
	if err != nil {
		log.LogMessage(log.ERROR, "Password mismatch: "+err.Error())
		return "", "", fmt.Errorf("invalid credentials")
	}

	// Ensure root folder exists for the user
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM folders WHERE user_UID = ? AND parent_id IS NULL", userUID).Scan(&count)

	if err != nil {
		log.LogMessage(log.ERROR, "Failed to create root folder: "+err.Error())
		return "", "", err
	}

	if count == 0 {
		rootUID := uuid.New().String()
		_, err = db.Exec("INSERT INTO folders (UID, user_UID, name, parent_id) VALUES (?, ?, 'root', NULL)", rootUID, userUID)
		if err != nil {
			return "", "", err
		}
		log.LogMessage(log.SUCS, "Root folder created for user:")
	}

	token, err := auth.GenerateTokenJWT(usr, pwd)
	if err != nil {
		log.LogMessage(log.ERROR, "Token generation failed: "+err.Error())
		return "", "", err
	}

	return token, userUID, nil
}
