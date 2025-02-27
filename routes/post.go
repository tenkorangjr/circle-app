package routes

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/tenkorangjr/circle-app/db"
	"github.com/tenkorangjr/circle-app/models"
)

type postResponse struct {
	caption string `binding:"required"`
}

func createPost(context *gin.Context) {
	var response postResponse
	if err := context.ShouldBindJSON(&response); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Response doesn't have required field", "error": err.Error()})
		return
	}

	file, err := context.FormFile("post")
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Failed to retrieve file", "error": err.Error()})
		return
	}

	userId := context.GetUint("userId")

	var user models.User
	if err := db.DB.Where("id = ?", userId).First(&user).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "couldn't find user", "error": err})
		return
	}

	post := models.NewPost("", response.caption, userId, user)

	fileName := fmt.Sprintf("%d_%d%s", userId, post.ID, filepath.Ext(file.Filename))
	post.ImageURL = fileName

	if err := context.SaveUploadedFile(file, fileName); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to save file", "error": err.Error()})
		return
	}

	if err := db.DB.Create(&post).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create post", "error": err.Error()})
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Post created successfully", "post": post})
}
