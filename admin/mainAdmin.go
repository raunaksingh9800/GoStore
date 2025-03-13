package admin

import (
	"GoStore/auth"
	l "GoStore/log"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv" // Import SQLite driver
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func updateEnv(key, value string) error {
	envMap, err := godotenv.Read(".env")
	if err != nil {
		l.LogMessage(l.ERROR, "Failed to read .env: "+err.Error())
		return err
	}
	envMap[key] = value

	var envData []string
	for k, v := range envMap {
		envData = append(envData, fmt.Sprintf("%s=%s", k, v))
	}

	return os.WriteFile(".env", []byte(strings.Join(envData, "\n")), 0644)
}

func AdminLoginCred(usr, pwd string) (bool, bool, string) {
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "Database connection failed: "+err.Error())
		return false, false, ""
	}
	defer db.Close()

	var storedPwd string
	var firstLogin bool

	query := `SELECT pwd, default_cred FROM ADMIN WHERE user = ?`
	err = db.QueryRow(query, usr).Scan(&storedPwd, &firstLogin)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, false, ""
		}
		l.LogMessage(l.ERROR, "Query error: "+err.Error())
		return false, false, ""
	}

	if err = bcrypt.CompareHashAndPassword([]byte(storedPwd), []byte(pwd)); err != nil {
		return false, false, ""
	}

	if firstLogin {
		// If it's the first login, do not generate a session token
		return true, true, ""
	}

	JWTToken, err := auth.GenerateTokenJWT(usr, pwd)
	if err != nil {
		l.LogMessage(l.ERROR, "GenerateToken failed: "+err.Error())
		return false, false, ""
	}

	return true, false, JWTToken
}

func IfFirstLogin(newUSR, newPWD string) (int, error) {
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "Database connection failed: "+err.Error())
		return 500, err
	}
	defer db.Close()

	var defaultCred bool
	err = db.QueryRow(`SELECT default_cred FROM ADMIN WHERE user = ?`, "admin").Scan(&defaultCred)
	if err != nil {
		if err == sql.ErrNoRows {
			return 404, fmt.Errorf("admin user not found")
		}
		l.LogMessage(l.ERROR, "Query error: "+err.Error())
		return 500, err
	}

	if !defaultCred {
		return 409, fmt.Errorf("default credentials already changed")
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(newPWD), bcrypt.DefaultCost)
	if err != nil {
		l.LogMessage(l.ERROR, "Password hashing failed: "+err.Error())
		return 500, err
	}

	stmt, err := db.Prepare(`UPDATE ADMIN SET user = ?, pwd = ?, default_cred = 0 WHERE user = ?`)
	if err != nil {
		l.LogMessage(l.ERROR, "Statement preparation failed: "+err.Error())
		return 500, err
	}
	defer stmt.Close()

	_, err = stmt.Exec(newUSR, string(hashedPwd), "admin")
	if err != nil {
		l.LogMessage(l.ERROR, "Update query failed: "+err.Error())
		return 500, err
	}

	updateEnv("DEFAULT_CRED", "false")

	return 200, nil
}
func DelUser(usr string) (int, error) {
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "Database connection failed: "+err.Error())
		return 500, err
	}
	defer db.Close()

	stmt, err := db.Prepare(`DELETE FROM users WHERE username = ?`)
	if err != nil {
		l.LogMessage(l.ERROR, "Statement preparation failed: "+err.Error())
		return 500, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(usr)
	if err != nil {
		l.LogMessage(l.ERROR, "Delete query failed: "+err.Error())
		return 500, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		l.LogMessage(l.ERROR, "Failed to get affected rows: "+err.Error())
		return 500, err
	}

	if rowsAffected == 0 {
		return 404, fmt.Errorf("user not found")
	}

	return 200, nil
}
func AddUser(usr, pwd string) (int, error) {

	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "Database connection failed: "+err.Error())
		return 500, err
	}
	defer db.Close()

	// Create table if not exists

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		l.LogMessage(l.ERROR, "Password hashing failed: "+err.Error())
		return 500, err
	}

	// Generate UID using username without special characters
	uidBytes, err := bcrypt.GenerateFromPassword(append([]byte(usr), []byte(pwd)...), bcrypt.MinCost)
	if err != nil {
		l.LogMessage(l.ERROR, "UID generation failed: "+err.Error())
		return 500, err
	}
	uid := fmt.Sprintf("%x", uidBytes)

	stmt, err := db.Prepare(`INSERT INTO users (UID, username, pwd) VALUES (?, ?, ?)`)
	if err != nil {
		l.LogMessage(l.ERROR, "Statement preparation failed: "+err.Error())
		return 500, err
	}
	defer stmt.Close()

	_, err = stmt.Exec(uid, usr, string(hashedPwd))
	if err != nil {
		l.LogMessage(l.ERROR, "Insert query failed: "+err.Error())
		return 500, err
	}

	return 200, nil
}

func ListUsers() ([]string, error) {
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "Database connection failed: "+err.Error())
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT username FROM users")
	if err != nil {
		l.LogMessage(l.ERROR, "Query error: "+err.Error())
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			l.LogMessage(l.ERROR, "Row scan failed: "+err.Error())
			return nil, err
		}
		users = append(users, username)
	}

	if err = rows.Err(); err != nil {
		l.LogMessage(l.ERROR, "Rows iteration error: "+err.Error())
		return nil, err
	}

	return users, nil
}

func IsItAdmin(usr, token string) (bool, error) {
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "Database connection failed: "+err.Error())
		return false, err
	}
	defer db.Close()

	// Authenticate the token
	isValid, claims := auth.AuthenticateTokenJWT(token)
	if !isValid {
		return false, fmt.Errorf("invalid token")
	}

	// Check if the user in the token matches the provided user
	if claims.Username != usr {
		return false, fmt.Errorf("token does not match the user")
	}

	// Check if the user has admin privileges
	var isAdmin bool
	query := `SELECT EXISTS(SELECT 1 FROM ADMIN WHERE user = ?)`
	err = db.QueryRow(query, usr).Scan(&isAdmin)
	if err != nil {
		l.LogMessage(l.ERROR, "Query error: "+err.Error())
		return false, err
	}

	return isAdmin, nil
}
