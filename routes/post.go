package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

	src, err := file.Open()
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "File failed to open", "error": err.Error()})
		return
	}
	defer src.Close()

}
