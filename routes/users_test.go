package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tenkorangjr/circle-app/db"
	"github.com/tenkorangjr/circle-app/models"
	"github.com/tenkorangjr/circle-app/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func SetupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}
	db.AutoMigrate(&models.User{})
	return db
}

func TestSignUpRoute(t *testing.T) {
	db.DB = SetupTestDB()
	server := gin.Default()
	RegisterRoutes(server)

	user := models.NewUser("michael@tenkorang.com", "admin")
	userBytes, _ := json.Marshal(user)
	req, _ := http.NewRequest("POST", "/signup", bytes.NewBuffer(userBytes))
	req.Header.Set("Content-Type", "application/json")

	// Record the response body
	responseWriter := httptest.NewRecorder()
	server.ServeHTTP(responseWriter, req)

	assert.Equal(t, http.StatusCreated, responseWriter.Code)

	var createdUser models.User
	json.Unmarshal(responseWriter.Body.Bytes(), &createdUser)

	assert.Equal(t, "michael@tenkorang.com", createdUser.Email)
	checkPassword := utils.ValidatePassword(createdUser.Password, "admin")
	assert.True(t, checkPassword) // Ensure the password is not returned in plain text
}

func TestSignInRoute(t *testing.T) {

	db.DB = SetupTestDB()
	server := gin.Default()
	RegisterRoutes(server)

	// Mock signing up a user
	user := models.NewUser("michael@tenkorang.com", "admin")
	userBytes, _ := json.Marshal(user)
	req, _ := http.NewRequest("POST", "/signup", bytes.NewBuffer(userBytes))
	req.Header.Set("Content-Type", "application/json")
	responseWriter := httptest.NewRecorder()
	server.ServeHTTP(responseWriter, req)

	assert.Equal(t, http.StatusCreated, responseWriter.Code)

	// Testing signing in user
	signInData := map[string]string{
		"email":    "michael@tenkorang.com",
		"password": "admin",
	}
	signInBytes, _ := json.Marshal(signInData)
	signInReq, _ := http.NewRequest("POST", "/signin", bytes.NewBuffer(signInBytes))
	signInReq.Header.Set("Content-Type", "application/json")
	signInResponseWriter := httptest.NewRecorder()
	server.ServeHTTP(signInResponseWriter, signInReq)

	assert.Equal(t, http.StatusOK, signInResponseWriter.Code)

	var signInResponse map[string]interface{}
	json.Unmarshal(signInResponseWriter.Body.Bytes(), &signInResponse)

	assert.NotEmpty(t, signInResponse["token"]) // Assuming the response contains a token
}
