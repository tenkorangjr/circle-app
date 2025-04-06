package routes

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/tenkorangjr/circle-app/db"
	"github.com/tenkorangjr/circle-app/models"
	"github.com/tenkorangjr/circle-app/utils"
	"gorm.io/gorm"
)

const uploadTimeout = 50 * time.Second

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

func createPost(gc *gin.Context) {

	gctx := gc.Request.Context()

	caption := gc.PostForm("caption")
	if caption == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "Caption is required"})
		return
	}

	file, err := gc.FormFile("post")
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "Failed to retrieve file", "error": err.Error()})
		return
	}

	f, err := file.Open()
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "server failed to open file"})
		return
	}
	defer f.Close()

	userId := gc.GetUint("userId")

	var user models.User
	if err := db.DB.Where("id = ?", userId).First(&user).Error; err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "couldn't find user", "error": err.Error()})
		return
	}

	post := models.NewPost("", caption, userId, user)
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(post).Error; err != nil {
			return err
		}

		postName := fmt.Sprintf("%d/%d.jpg", userId, post.ID)

		path, err := utils.UploadToBucket(f, postName, gctx, uploadTimeout)
		if err != nil {
			return err
		}
		post.ImageURL = path

		if err := tx.Save(post).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create post in db", "err": err.Error()})
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Post created successfully", "post": post})
}

func getPostbyUserAndPostID(gc *gin.Context) {
	requestUserID := gc.Param("id")
	requestPostID := gc.Param("postid")
	if requestPostID == "" || requestUserID == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "Incorrect params for request"})
		return
	}

	var post models.Post
	if result := db.DB.Preload("User").First(&post, requestPostID); errors.Is(result.Error, gorm.ErrRecordNotFound) {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "post does not exist"})
		return
	}

	url, err := utils.GenerateGetSignedURL(post.ImageURL, gc.Request.Context())
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate signed URL", "error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"post": post, "signed_url": url})
}
