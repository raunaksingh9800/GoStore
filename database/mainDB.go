package database

import (
	l "GoStore/log"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type Database struct {
	DB *sql.DB
}

func NewDatabase(db *sql.DB) *Database {
	return &Database{DB: db}
}

func updateEnv(key, value string) error {
	// Load existing environment variables
	envMap, err := godotenv.Read(".env")
	if err != nil {
		l.LogMessage(l.ERROR, err.Error())
		return err
	}

	// Update or add the key-value pair
	envMap[key] = value

	// Convert map to string format
	var envData []string
	for k, v := range envMap {
		envData = append(envData, fmt.Sprintf("%s=%s", k, v))
	}

	// Write back to .env file
	return os.WriteFile(".env", []byte(strings.Join(envData, "\n")), 0644)
}

func (d *Database) createUserTable() {

	createTableSQL := `CREATE TABLE IF NOT EXISTS users (
		UID TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		pwd TEXT NOT NULL
	);`

	_, err := d.DB.Exec(createTableSQL)

	if err != nil {
		l.LogMessage(l.ERROR, err.Error())
		return
	}
	l.LogMessage(l.SUCS, "User Table created")
}

func (d *Database) createfoldersTable() {

	createTableSQL := `CREATE TABLE IF NOT EXISTS folders (
		UID TEXT PRIMARY KEY,
		user_UID TEXT NOT NULL,
		name TEXT NOT NULL,
		parent_id TEXT NULL,
		FOREIGN KEY (user_UID) REFERENCES users(UID),
		FOREIGN KEY (parent_id) REFERENCES folders(UID)
	);`

	_, err := d.DB.Exec(createTableSQL)

	if err != nil {
		l.LogMessage(l.ERROR, err.Error())
		return
	}
	l.LogMessage(l.SUCS, "Folder Table created")
}

func (d *Database) createfilesTable() {

	createTableSQL := `CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		folder_id TEXT NOT NULL,
		name TEXT NOT NULL,
		hashed_name TEXT NOT NULL,
		FOREIGN KEY (folder_id) REFERENCES folders(UID)
	);`

	_, err := d.DB.Exec(createTableSQL)

	if err != nil {
		l.LogMessage(l.ERROR, err.Error())
		return
	}
	l.LogMessage(l.SUCS, "Files Table created")
}

//  -------------ADMIN start----------------

func (d *Database) createAdminDetails() {
	createTableSQL := `CREATE TABLE IF NOT EXISTS ADMIN (
		user TEXT NOT NULL PRIMARY KEY,
		pwd TEXT NOT NULL,
		default_cred BOOLEAN NOT NULL
	);`

	_, err := d.DB.Exec(createTableSQL)
	if err != nil {
		l.LogMessage(l.ERROR, "Failed to create admin table: "+err.Error())
		return
	}
	l.LogMessage(l.SUCS, "Admin Table created")

	// Hash the password
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		l.LogMessage(l.ERROR, "Password hashing failed: "+err.Error())
		return
	}

	l.LogMessage(l.INFO, "ENV:FIRST_START = "+os.Getenv("FIRST_START"))

	if os.Getenv("FIRST_START") == "true" {
		insertSQL := `INSERT OR REPLACE INTO ADMIN (user, pwd, default_cred) 
			VALUES (?, ?, ?);`

		l.LogMessage(l.INFO, "ENV:DEFAULT_CRED = "+os.Getenv("DEFAULT_CRED"))

		defaultCred := 0 // Default to false
		if os.Getenv("DEFAULT_CRED") == "true" {
			defaultCred = 1
		}

		// Generate a dummy access token (in a real-world scenario, generate a secure token)

		_, err = d.DB.Exec(insertSQL, "admin", string(hashedPwd), defaultCred)
		if err != nil {
			l.LogMessage(l.ERROR, "Failed to insert admin details: "+err.Error())
			return
		}

		updateEnv("FIRST_START", "false")
	}

	l.LogMessage(l.SUCS, "Admin details added")
}

//  -------------ADMIN end----------------

// func (d *Database) createSessionTable() {

// 	createTableSQL := `CREATE TABLE IF NOT EXISTS SESSIONS (
//     user TEXT NOT NULL,
//     token TEXT NOT NULL,
//     expires_at DATETIME NOT NULL,
//     FOREIGN KEY (user) REFERENCES ADMIN(user) ON DELETE CASCADE
// );`

// 	_, err := d.DB.Exec(createTableSQL)

// 	if err != nil {
// 		l.LogMessage(l.ERROR, err.Error())
// 		return
// 	}
// 	l.LogMessage(l.SUCS, "SESSIONS Table created")
// }

func (db *Database) EnsureRootFolder(userUID string) error {
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM folders WHERE user_UID = ? AND parent_id IS NULL", userUID).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		rootUID := uuid.New().String()
		_, err = db.DB.Exec("INSERT INTO folders (UID, user_UID, name, parent_id) VALUES (?, ?, 'root', NULL)", rootUID, userUID)
		if err != nil {
			return err
		}
		log.Println("Root folder created for user:", userUID)
	}
	return nil
}

func InitDB() {
	err := godotenv.Load()
	if err != nil {
		l.LogMessage(l.ERROR, "Error loading .env file")
	}
	// Open database connection
	db, err := sql.Open("sqlite3", "main.db") // This creates or opens 'example.db'
	if err != nil {
		l.LogMessage(l.ERROR, err.Error())
	}
	// defer db.Close() // Do not close the database connection here

	database := NewDatabase(db)

	database.createUserTable()
	database.createfoldersTable()
	database.createfilesTable()
	database.createAdminDetails()

	database.DB.Close()
}
