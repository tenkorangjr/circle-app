package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/tenkorangjr/circle-app/db"
	"github.com/tenkorangjr/circle-app/models"
	requestmodel "github.com/tenkorangjr/circle-app/models/requests"
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

		if err := tx.Save(&post).Error; err != nil {
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
	var wg sync.WaitGroup

	errorChan := make(chan error, 3)
	wg.Add(3)

	go func() {
		defer wg.Done()
		if err := db.DB.Preload("User").First(&post, requestPostID).Error; err != nil {
			errorChan <- err
		}
	}()

	go func() {
		defer wg.Done()
		if err := db.DB.Preload("Likes").First(&post, requestPostID).Error; err != nil {
			errorChan <- err
		}
	}()

	go func() {
		defer wg.Done()
		if err := db.DB.Preload("Comments").First(&post, requestPostID).Error; err != nil {
			errorChan <- err
		}
	}()

	wg.Wait()
	close(errorChan)

	for err := range errorChan {
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"message": "failed to preload associated field", "error": err})
			return
		}
	}

	url, err := utils.GenerateGetSignedURL(post.ImageURL, gc.Request.Context())
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate signed URL", "error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"post": post, "signed_url": url, "likes": len(post.Likes)})
}

func postLike(gc *gin.Context) {
	postId := gc.Param("postid")
	userId := gc.GetUint("userId")
	if postId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "no post id given"})
		return
	}

	parsedPostID, err := strconv.Atoi(postId)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "invalid post id format", "error": err.Error()})
		return
	}

	var like models.PostLike
	var post models.Post
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		like = models.PostLike{
			PostID:  uint(parsedPostID),
			LikerID: userId,
		}

		if err := tx.Create(&like).Error; err != nil {
			return err
		}

		var wg sync.WaitGroup
		wg.Add(3)
		errChan := make(chan error, 3)

		go func() {
			defer wg.Done()
			if err := tx.Preload("User").
				First(&post, parsedPostID).Error; err != nil {
				errChan <- err
			}
		}()

		go func() {
			defer wg.Done()
			if err := tx.Preload("Likes").
				First(&post, parsedPostID).Error; err != nil {
				errChan <- err
			}
		}()

		go func() {
			defer wg.Done()
			if err := tx.Preload("Comments").
				First(&post, parsedPostID).Error; err != nil {
				errChan <- err
			}
		}()

		wg.Wait()
		close(errChan)

		for err := range errChan {
			if err != nil {
				return err
			}
		}

		if post.Likes != nil {
			post.Likes = append(post.Likes, like)
		}

		if err := tx.Save(&post).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "failed to add like", "error": err.Error()})
		return
	}

	gc.JSON(http.StatusCreated, gin.H{"message": "like added to post", "likes": len(post.Likes), "post": post})
}

func postComment(gc *gin.Context) {
	postId := gc.Param("postid")
	userId := gc.GetUint("userId")
	if postId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "no post id given"})
		return
	}

	var postComment requestmodel.CommentRequest
	err := gc.ShouldBindJSON(&postComment)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "no content attached to comment"})
		return
	}
	if validate.Struct(postComment) != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "comment exceed required length"})
	}

	parsedPostID, err := strconv.Atoi(postId)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"message": "invalid post id format", "error": err.Error()})
		return
	}

	var comment models.PostComment
	var post models.Post
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		comment = models.PostComment{
			Content:     postComment.Content,
			PostID:      uint(parsedPostID),
			CommenterID: userId,
		}

		if err := tx.Create(&comment).Error; err != nil {
			return err
		}

		var wg sync.WaitGroup
		wg.Add(3)
		errChan := make(chan error, 3)

		go func() {
			defer wg.Done()
			if err := tx.Preload("User").
				First(&post, parsedPostID).Error; err != nil {
				errChan <- err
			}
		}()

		go func() {
			defer wg.Done()
			if err := tx.Preload("Likes").
				First(&post, parsedPostID).Error; err != nil {
				errChan <- err
			}
		}()

		go func() {
			defer wg.Done()
			if err := tx.Preload("Comments").
				First(&post, parsedPostID).Error; err != nil {
				errChan <- err
			}
		}()

		wg.Wait()
		close(errChan)

		for err := range errChan {
			if err != nil {
				return err
			}
		}

		if post.Comments != nil {
			post.Comments = append(post.Comments, comment)
		}

		if err := tx.Save(&post).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "failed to add comment", "error": err.Error()})
		return
	}

	gc.JSON(http.StatusCreated, gin.H{"message": "comment added to post", "comment": comment, "post": post})
}
