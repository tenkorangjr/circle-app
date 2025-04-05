package routes

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tenkorangjr/circle-app/db"
	"github.com/tenkorangjr/circle-app/models"

	"cloud.google.com/go/storage"
)

func createPost(context *gin.Context) {
	caption := context.PostForm("caption")
	if caption == "" {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Caption is required"})
		return
	}

	file, err := context.FormFile("post")
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Failed to retrieve file", "error": err.Error()})
		return
	}

	f, err := file.Open()
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "server failed to open file"})
		return
	}
	defer f.Close()

	userId := context.GetUint("userId")

	var user models.User
	if err := db.DB.Where("id = ?", userId).First(&user).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "couldn't find user", "error": err.Error()})
		return
	}

	post := models.NewPost("", caption, userId, user)
	db.DB.Create(post)
	postName := fmt.Sprintf("%d/%d.jpg", userId, post.ID)

	mediaLink, err := uploadToBucket(&f, postName)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "unable to upload to bucket", "error": err.Error()})
		return
	}
	post.ImageURL = mediaLink
	db.DB.Save(post)

	context.JSON(http.StatusOK, gin.H{"message": "Post created successfully", "post": post})
}

func uploadToBucket(file *multipart.File, postName string) (string, error) {
	bucketName := "circle_app_posts"

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	o := client.Bucket(bucketName).Object(postName)

	wc := o.NewWriter(ctx)
	if _, err = io.Copy(wc, *file); err != nil {
		wc.Close() // Ensure the writer is closed before returning
		return "", err
	}

	if err := wc.Close(); err != nil { // Finalize the object upload
		return "", err
	}

	attrs, err := o.Attrs(ctx)
	if err != nil {
		return "", err
	}

	fmt.Printf("Blob uploaded successfully: %s", attrs.MediaLink)
	return attrs.MediaLink, nil
}
