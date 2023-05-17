package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	extraction "github.com/j-briant/extract-service/cmd/geometry"
	"github.com/j-briant/extract-service/cmd/web"
)

var secretKey = "supersecretkey"

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var users = make(map[string]User)
var blacklist = make(map[string]bool)

func main() {
	// Create a new Gin router
	router := gin.Default()

	// Define a GET endpoint for the root path
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})

	// Define the /home route
	router.GET("/home", func(c *gin.Context) {
		// Render the homepage HTML template
		c.JSON(http.StatusOK, gin.H{"message": "Welcome on /home."})
	})

	// Login user route
	router.POST("/login", web.SigninHandler)

	// Register user route
	router.POST("/register", web.SignupHandler)

	// Post parameters route
	router.POST("/extraction", extraction.VectorExtraction)

	// Add protected routes
	auth := router.Group("/")
	auth.Use(web.AuthMiddleware(secretKey))
	{
		auth.GET("/logout", web.LogoutHandler)
		auth.GET("/protected-route", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "You have access to a protected route."})
		})
	}

	// Start the server and listen for incoming HTTP requests
	if err := router.Run(":8080"); err != nil {
		panic(err)
	}
}
