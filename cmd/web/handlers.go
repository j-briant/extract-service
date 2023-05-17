package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

type User struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Email      string `json:"email" binding:"email"`
	FullName   string `json:"fullname"`
	CreateDate string `json:"createdate"`
}

// This is just an example. You should store user data in a database in a real application.
var (
	users     = make(map[string]User)
	blacklist = make(map[string]bool)
	jwtKey    = []byte("my_secret_key")
)

// Register user handler.
func SignupHandler(c *gin.Context) {
	var user User
	err := c.BindJSON(&user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, exists := users[user.Username]; exists {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "User already exists"})
		return
	}

	// TODO: Implement code to save user to a database or file system
	users[user.Username] = user

	c.JSON(200, gin.H{"message": fmt.Sprintf("User %s registered successfully!", user.Username)})
}

// Login user handler.
func SigninHandler(c *gin.Context) {
	var user User
	err := c.BindJSON(&user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Implement code to check if user exists in the database and if the
	//       password matches
	// For now just checks if received user matches a registered user
	if users[user.Username] != user {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Login or password incorrect"})
		return
	}

	claims := jwt.MapClaims{}
	claims["username"] = user.Username
	claims["exp"] = time.Now().Add(time.Hour * 1).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// Logout user handler.
func LogoutHandler(c *gin.Context) {
	// Get the JWT token from the Authorization header
	tokenString := c.Request.Header.Get("Authorization")[7:]

	// Parse the token
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify that the token was signed with the expected algorithm and key
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}

	// Check if the token is in the blacklist
	if _, ok := blacklist[tokenString]; ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token has already been revoked"})
		return
	}

	// Add the token to the blacklist
	blacklist[tokenString] = true

	c.JSON(http.StatusOK, gin.H{"message": "User logged out successfully"})
}
