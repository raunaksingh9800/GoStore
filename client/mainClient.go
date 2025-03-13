package client

import (
	"GoStore/auth"
	l "GoStore/log"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// IsItUser checks if the provided token belongs to the given user.
func IsItUser(usr, token string) (bool, error) {
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

	// Check if the user exists
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)`
	err = db.QueryRow(query, usr).Scan(&exists)
	if err != nil {
		l.LogMessage(l.ERROR, "Query error: "+err.Error())
		return false, err
	}

	return exists, nil
}

// NewFolder creates a new folder for a user.
func NewFolder(usr, name, parent string) (int, error) {
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "Database connection failed: "+err.Error())
		return http.StatusInternalServerError, err
	}
	defer db.Close()

	// Check if the user exists
	var userExists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE UID = ?)`
	err = db.QueryRow(query, usr).Scan(&userExists)
	if err != nil {
		l.LogMessage(l.ERROR, "User check failed: "+err.Error())
		return http.StatusInternalServerError, err
	}
	if !userExists {
		return http.StatusNotFound, fmt.Errorf("user not found")
	}

	// Check if parent folder exists (if provided)
	if parent != "" {
		var parentExists bool
		query = `SELECT EXISTS(SELECT 1 FROM folders WHERE UID = ?)`
		err = db.QueryRow(query, parent).Scan(&parentExists)
		if err != nil {
			l.LogMessage(l.ERROR, "Parent folder check failed: "+err.Error())
			return http.StatusInternalServerError, err
		}
		if !parentExists {
			return http.StatusBadRequest, fmt.Errorf("parent folder not found")
		}
	}

	// Generate a new UID for the folder
	folderUID := uuid.New().String()

	// Insert the new folder
	insertQuery := `INSERT INTO folders (UID, user_UID, name, parent_id) VALUES (?, ?, ?, ?)`
	_, err = db.Exec(insertQuery, folderUID, usr, name, parent)
	if err != nil {
		l.LogMessage(l.ERROR, "Folder creation failed: "+err.Error())
		return http.StatusInternalServerError, err
	}

	return http.StatusCreated, nil
}

// SaveFileMetadata stores file metadata in the database.
func SaveFileMetadata(usr, folderID, fileName, hashedName string) (int, error) {
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "Database connection failed: "+err.Error())
		return http.StatusInternalServerError, err
	}
	defer db.Close()

	// Check if the folder exists and belongs to the user
	var folderExists bool
	query := `SELECT EXISTS(SELECT 1 FROM folders WHERE UID = ? AND user_UID = ?);`
	l.LogMessage(l.INFO, "USER_UID: "+usr)
	l.LogMessage(l.INFO, "folderID: "+folderID)
	err = db.QueryRow(query, folderID, usr).Scan(&folderExists)
	l.LogMessage(l.INFO, folderExists)
	if err != nil {
		l.LogMessage(l.ERROR, "Folder existence check failed: "+err.Error())
		return http.StatusInternalServerError, err
	}
	if !folderExists {
		return http.StatusForbidden, fmt.Errorf("folder not found or unauthorized access")
	}

	// Insert file metadata
	insertQuery := `INSERT INTO files (folder_id, name, hashed_name) VALUES (?, ?, ?)`
	_, err = db.Exec(insertQuery, folderID, fileName, hashedName)
	if err != nil {
		l.LogMessage(l.ERROR, "File metadata insertion failed: "+err.Error())
		return http.StatusInternalServerError, err
	}

	l.LogMessage(l.SUCS, "File metadata saved successfully")
	return http.StatusCreated, nil
}

func ViewFile(c *gin.Context) {
	// Extract user token from headers
	token := c.GetHeader("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
		return
	}

	// Authenticate the token
	isValid, claims := auth.AuthenticateTokenJWT(token)
	if !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Extract file ID from URL
	fileID := c.Param("fileID")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID is required"})
		return
	}

	// Connect to the SQLite database
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "Database connection failed: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	defer db.Close()

	// Fetch file details from the database
	var fileName, hashedName, folderOwner string
	query := `SELECT f.name, f.hashed_name, u.username
		FROM files f
		JOIN folders fo ON f.folder_id = fo.UID
		JOIN users u ON fo.user_UID = u.UID
		WHERE f.hashed_name = ?`
	err = db.QueryRow(query, fileID).Scan(&fileName, &hashedName, &folderOwner)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		} else {
			l.LogMessage(l.ERROR, "Database query failed: "+err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	// Ensure the requesting user is the owner of the file
	if claims.Username != folderOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access"})
		return
	}

	// Construct the file path
	filePath := filepath.Join("uploads", hashedName)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found on disk"})
		return
	}

	// Serve the file securely with the original filename
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	c.File(filePath)

	l.LogMessage(l.INFO, "File served: "+fileName)
}

func DeleteFile(c *gin.Context) {
	// Extract user token
	token := c.GetHeader("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
		return
	}

	// Authenticate the token
	isValid, claims := auth.AuthenticateTokenJWT(token)
	if !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Extract file ID from URL
	fileID := c.Param("fileID")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID is required"})
		return
	}

	// Connect to the database
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		l.LogMessage(l.ERROR, "Database connection failed: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	defer db.Close()

	// Retrieve file details
	var fileName, hashedName, folderOwner string
	query := `SELECT f.name, f.hashed_name, u.username
		FROM files f
		JOIN folders fo ON f.folder_id = fo.UID
		JOIN users u ON fo.user_UID = u.UID
		WHERE f.hashed_name = ?`
	err = db.QueryRow(query, fileID).Scan(&fileName, &hashedName, &folderOwner)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		} else {
			l.LogMessage(l.ERROR, "Database query failed: "+err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	// Ensure the requesting user is the owner of the file
	if claims.Username != folderOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access"})
		return
	}

	// Construct the file path
	filePath := filepath.Join("uploads", hashedName)

	// Delete the file from disk
	if err := os.Remove(filePath); err != nil {
		l.LogMessage(l.ERROR, "Failed to delete file from disk: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	// Delete file record from the database
	deleteQuery := `DELETE FROM files WHERE hashed_name = ?`
	_, err = db.Exec(deleteQuery, fileID)
	if err != nil {
		l.LogMessage(l.ERROR, "Failed to delete file record: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file record"})
		return
	}

	// Respond to client
	l.LogMessage(l.INFO, "File deleted: "+fileName)
	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}
