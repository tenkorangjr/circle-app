package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tenkorangjr/circle-app/db"
	"github.com/tenkorangjr/circle-app/models"
	"github.com/tenkorangjr/circle-app/utils"
)

func signUp(context *gin.Context) {

	var user models.User
	err := context.ShouldBindJSON(&user)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Could not create user model"})
		return
	}

	err = user.Save(db.DB)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Unable to save user to db", "error": err})
		return
	}

	context.JSON(http.StatusCreated, user)
}

// Sign in the user by checking if the user exists using the email and then validating the password
func signIn(context *gin.Context) {

	var user models.User
	err := context.ShouldBindJSON(&user)

	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Incorrect fields"})
		return
	}

	var queryUser models.User
	db.DB.Where("email = ?", user.Email).First(&queryUser)
	if queryUser.Email == "" {
		context.JSON(http.StatusNotFound, gin.H{"message": "Email not found"})
		return
	}

	if !utils.ValidatePassword(queryUser.Password, user.Password) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	token, err := utils.GenerateJWT(int(queryUser.ID), queryUser.Email)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Could not generate JWT token"})
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "User logged in successfully", "token": token})
}
