package main

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/j-briant/extract-service/authentication"
	"github.com/j-briant/extract-service/extraction"
)

var secretKey = []byte("supersecretkey")

func main() {
	// Create a new Gin router
	router := gin.Default()

	// Define a GET endpoint for the root path
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})

	// Load the registration form HTML template
	htmlTemplate := template.Must(template.ParseFiles("html/register.html"))

	// Add a registration route
	router.GET("/register", func(c *gin.Context) {
		htmlTemplate.Execute(c.Writer, nil)
	})

	router.POST("/register", authentication.SubmitRegistrationHandler)

	// Add a protected route
	authorized := router.Group("/extract-vector")
	authorized.Use(authentication.AuthMiddleware(string(secretKey)))
	authorized.GET("/", extraction.VectorExtraction)

	// Start the server and listen for incoming HTTP requests
	if err := router.Run(":8080"); err != nil {
		panic(err)
	}
}
