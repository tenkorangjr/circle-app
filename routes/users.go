package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tenkorangjr/circle-app/db"
	"github.com/tenkorangjr/circle-app/models"
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
