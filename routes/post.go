package routes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/tenkorangjr/circle-app/db"
	"github.com/tenkorangjr/circle-app/models"
	"gorm.io/gorm"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
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

		path, err := uploadToBucket(f, postName, gctx)
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

	url, err := generateGetSignedURL(post.ImageURL, gc.Request.Context())
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate signed URL", "error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"post": post, "signed_url": url})
}

func generateGetSignedURL(object string, context context.Context) (string, error) {
	bucket := os.Getenv("BUCKET_NAME")
	sakeyFile := "./sa-cred.json"

	saKey, err := os.ReadFile(sakeyFile)
	if err != nil {
		return "", fmt.Errorf("failed to read service account key")
	}

	cfg, err := google.JWTConfigFromJSON(saKey)
	if err != nil {
		return "", fmt.Errorf("failed to read config file with service account key")
	}

	client, err := storage.NewClient(context)
	if err != nil {
		return "", err
	}
	defer client.Close()

	opts := &storage.SignedURLOptions{
		GoogleAccessID: cfg.Email,
		PrivateKey:     cfg.PrivateKey,
		Scheme:         storage.SigningSchemeV4,
		Method:         "GET",
		Expires:        time.Now().Add(15 * time.Minute),
	}

	url, err := client.Bucket(bucket).SignedURL(object, opts)
	if err != nil {
		return "", fmt.Errorf("Bucket(%q).SignedURL: %w", bucket, err)
	}

	return url, nil
}

func uploadToBucket(file multipart.File, postName string, ctx context.Context) (string, error) {
	bucketName := "circle_app_posts"

	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	o := client.Bucket(bucketName).Object(postName)

	wc := o.NewWriter(ctx)
	if _, err = io.Copy(wc, file); err != nil {
		wc.Close()
		return "", err
	}

	if err := wc.Close(); err != nil {
		return "", err
	}

	fmt.Printf("Blob uploaded successfully: %s", postName)
	return postName, nil
}
