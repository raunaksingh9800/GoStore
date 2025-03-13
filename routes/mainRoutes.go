package routes

import (
	admindb "GoStore/admin"
	"GoStore/client"
	user "GoStore/client"
	l "GoStore/log"
	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type Credentials_user struct {
	Username string `json:"username"`
}

type NewFolderData struct {
	Name      string `json:"name"`
	Parent_id string `json:"parent"`
	User_uid  string `json:"user_uid"`
}

func CORSMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allow all origins (change if needed)
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "usr", "token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	})
}

// Middleware to check if the user is an admin
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		usr := c.GetHeader("usr")
		token := c.GetHeader("token")

		isAdmin, err := admindb.IsItAdmin(usr, token)
		if err != nil || !isAdmin {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func UserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		usr := c.GetHeader("usr")
		token := c.GetHeader("token")

		isUser, err := client.IsItUser(usr, token)
		if err != nil || !isUser {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func StartServer() {
	r := gin.Default()
	r.Use(CORSMiddleware())
	r.LoadHTMLFiles("templates/index.html")
	r.Static("/static", "./static")

	r.GET("/", func(ctx *gin.Context) {
		ctx.HTML(200, "index.html", gin.H{
			"title": "Welcome to GoStore",
		})
	})

	adminRoutes := r.Group("/admin")
	{
		adminRoutes.POST("/", func(c *gin.Context) {
			var creds Credentials

			// Bind JSON to struct
			if err := c.ShouldBindJSON(&creds); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			status, DEFAULT_CRED, token := admindb.AdminLoginCred(creds.Username, creds.Password)
			if DEFAULT_CRED {
				l.LogMessage(l.INFO, "true")
			} else {
				l.LogMessage(l.INFO, "false")
			}
			switch {
			case !status:
				c.JSON(http.StatusBadRequest, gin.H{"error": "CREDS ERROR"})
			case status && DEFAULT_CRED:
				c.JSON(http.StatusOK, gin.H{"action": "/newcred"})
			case status && !DEFAULT_CRED:
				c.JSON(http.StatusOK, gin.H{"action": "CRED CORRT", "token": token})
			}
		})

		adminRoutes.POST("/newcred", func(c *gin.Context) {
			var creds Credentials

			// Bind JSON to struct
			if err := c.ShouldBindJSON(&creds); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			code, err := admindb.IfFirstLogin(creds.Username, creds.Password)

			if err != nil {
				l.LogMessage(l.ERROR, err.Error())
			}

			if code == 500 {
				c.JSON(http.StatusInternalServerError, gin.H{"action": "err"})
			} else if code == 409 || code == 404 {
				c.JSON(http.StatusUnauthorized, gin.H{"action": "err"})
			} else if code == 200 {
				c.JSON(http.StatusOK, gin.H{"action": "/admin"})
			}
		})

		adminRoutes.POST("/newuser", AdminMiddleware(), func(c *gin.Context) {
			var creds Credentials
			if err := c.ShouldBindJSON(&creds); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			code, err := admindb.AddUser(creds.Username, creds.Password)

			if err != nil {
				l.LogMessage(l.ERROR, "mainRoutes :"+err.Error())
			}
			if code == 200 {
				c.JSON(http.StatusOK, gin.H{"action": "added"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"action": "err"})
			}
		})

		adminRoutes.DELETE("/deluser", AdminMiddleware(), func(c *gin.Context) {
			var creds Credentials_user
			if err := c.ShouldBindJSON(&creds); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			code, err := admindb.DelUser(creds.Username)

			if err != nil {
				l.LogMessage(l.ERROR, "mainRoutes :"+err.Error())
			}
			if code == 200 {
				c.JSON(http.StatusOK, gin.H{"action": "deleted"})
			} else if code == 404 {
				c.JSON(http.StatusNotFound, gin.H{"action": "not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"action": "err"})
			}
		})

		adminRoutes.GET("/dashboard", AdminMiddleware(), func(c *gin.Context) {
			users, err := admindb.ListUsers()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"users": users})
		})
	}

	adminClient := r.Group("/client")
	{
		adminClient.POST("/login", func(c *gin.Context) {
			var creds Credentials

			// Bind JSON to struct
			if err := c.ShouldBindJSON(&creds); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			token, UID, err := user.Login(creds.Username, creds.Password)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"token": token, "UID": UID})
		})
		adminClient.POST("/newfolder", UserMiddleware(), func(c *gin.Context) {
			var folder_data NewFolderData

			if err := c.ShouldBindJSON(&folder_data); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			status, err := client.NewFolder(folder_data.User_uid, folder_data.Name, folder_data.Parent_id)
			if err != nil {
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusCreated, gin.H{"message": "Folder created successfully"})
		})

		adminClient.POST("/newfile", UserMiddleware(), func(ctx *gin.Context) {
			file, err := ctx.FormFile("file")
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "Failed to retrieve file"})
				return
			}

			hashedName := uuid.New().String()
			dst := fmt.Sprintf("uploads/%s", hashedName)

			if err := ctx.SaveUploadedFile(file, dst); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
				return
			}

			folderID := ctx.PostForm("folder_id")
			userUID := ctx.PostForm("user_uid")
			customFileName := ctx.PostForm("filename")
			if customFileName == "" {
				customFileName = file.Filename // Default to the original filename
			}
			code, err := client.SaveFileMetadata(userUID, folderID, customFileName, hashedName)
			if err != nil {
				if code == http.StatusForbidden {
					ctx.JSON(http.StatusForbidden, gin.H{"error": "Folder not found or unauthorized access"})
				} else {
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file metadata"})
				}
				return
			}

			ctx.JSON(http.StatusCreated, gin.H{"message": "File uploaded successfully", "hashed_name": hashedName})
		})

		adminClient.GET("/file/view/:fileID", UserMiddleware(), func(c *gin.Context) {
			client.ViewFile(c) // Pass the Gin context
		})
		adminClient.DELETE("/file/delete/:fileID", UserMiddleware(), func(c *gin.Context) {
			client.DeleteFile(c) // Pass the Gin context
		})
	}

	r.Run(":8080")
}
