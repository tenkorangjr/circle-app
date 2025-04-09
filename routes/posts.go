package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/tenkorangjr/circle-app/db"
	"github.com/tenkorangjr/circle-app/models"
	requestmodel "github.com/tenkorangjr/circle-app/models/requests"
	"github.com/tenkorangjr/circle-app/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const uploadTimeout = 50 * time.Second

func init() {
	err := godotenv.Load()
	if err != nil {
		zap.S().Error("Error loading .env file", zap.Error(err))
	}
}

func createPost(gc *gin.Context) {
	gctx := gc.Request.Context()

	caption := gc.PostForm("caption")
	if caption == "" {
		zap.S().Error("Caption is required")
		gc.JSON(http.StatusBadRequest, gin.H{"message": "Caption is required"})
		return
	}

	file, err := gc.FormFile("post")
	if err != nil {
		zap.S().Error("Failed to retrieve file", zap.Error(err))
		gc.JSON(http.StatusBadRequest, gin.H{"message": "Failed to retrieve file"})
		return
	}

	f, err := file.Open()
	if err != nil {
		zap.S().Error("Server failed to open file", zap.Error(err))
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "server failed to open file"})
		return
	}
	defer f.Close()

	userId := gc.GetUint("userId")

	var user models.User
	if err := db.DB.Where("id = ?", userId).First(&user).Error; err != nil {
		zap.S().Error("Couldn't find user", zap.Error(err))
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "couldn't find user"})
		return
	}

	post := models.NewPost("", caption, userId, user)
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		defer func() {
			if r := recover(); r != nil || err != nil {
				tx.Rollback()
			}
		}()

		if err := tx.Create(post).Error; err != nil {
			zap.S().Error("Failed to create post", zap.Error(err))
			return err
		}

		postName := fmt.Sprintf("%d/%d.jpg", userId, post.ID)

		path, err := utils.UploadToBucket(f, postName, gctx, uploadTimeout)
		if err != nil {
			zap.S().Error("Failed to upload to bucket", zap.Error(err))
			return err
		}
		post.ImageURL = path

		if err := tx.Save(&post).Error; err != nil {
			zap.S().Error("Failed to save post", zap.Error(err))
			return err
		}
		return nil
	})
	if err != nil {
		zap.S().Error("Failed to create post in DB", zap.Error(err))
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create post in db"})
		return
	}

	zap.S().Info("Post created successfully", zap.Uint("postID", post.ID))
	gc.JSON(http.StatusOK, gin.H{"message": "Post created successfully", "post": post})
}

func getPostbyUserAndPostID(gc *gin.Context) {
	requestUserID := gc.Param("id")
	requestPostID := gc.Param("postid")
	if requestPostID == "" || requestUserID == "" {
		zap.S().Error("Incorrect params for request")
		gc.JSON(http.StatusBadRequest, gin.H{"message": "Incorrect params for request"})
		return
	}

	var post models.Post
	if err := db.DB.Preload(clause.Associations).
		First(&post, requestPostID).Error; err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"mesage": "could not find the post"})
		return
	}

	url, err := utils.GenerateGetSignedURL(post.ImageURL, gc.Request.Context())
	if err != nil {
		zap.S().Error("Failed to generate signed URL", zap.Error(err))
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate signed URL"})
		return
	}

	zap.S().Info("Successfully retrieved post", zap.Uint("postID", post.ID))
	gc.JSON(http.StatusOK, gin.H{"post": post, "signed_url": url, "likes": len(post.Likes)})
}

func postLike(gc *gin.Context) {
	postId := gc.Param("postid")
	userId := gc.GetUint("userId")
	if postId == "" {
		zap.S().Error("No post ID exists for user")
		gc.JSON(http.StatusBadRequest, gin.H{"message": "no post id given"})
		return
	}

	parsedPostID, err := strconv.Atoi(postId)
	if err != nil {
		zap.S().Error("Invalid post ID format", zap.Error(err))
		gc.JSON(http.StatusBadRequest, gin.H{"message": "invalid post id format"})
		return
	}

	var like models.PostLike
	var post models.Post
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		defer func() {
			if r := recover(); r != nil || err != nil {
				tx.Rollback()
			}
		}()

		like = models.PostLike{
			PostID:  uint(parsedPostID),
			LikerID: userId,
		}

		if err := tx.Create(&like).Error; err != nil {
			zap.S().Error("Failed to create like", zap.Error(err))
			return err
		}

		if err = db.DB.Preload(clause.Associations).
			First(&post, parsedPostID).Error; err != nil {
			return err
		}

		if post.Likes != nil {
			post.Likes = append(post.Likes, like)
		}

		if err := tx.Save(&post).Error; err != nil {
			zap.S().Error("Failed to save post after like", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		zap.S().Error("Failed to add like", zap.Error(err))
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "failed to add like"})
		return
	}

	zap.S().Info("Like added to post", zap.Uint("postID", post.ID), zap.Int("likesCount", len(post.Likes)))
	gc.JSON(http.StatusCreated, gin.H{"message": "like added to post", "likes": len(post.Likes), "post": post})
}

func postComment(gc *gin.Context) {
	postId := gc.Param("postid")
	userId := gc.GetUint("userId")
	if postId == "" {
		zap.S().Error("No post ID given")
		gc.JSON(http.StatusBadRequest, gin.H{"message": "no post id given"})
		return
	}

	var postComment requestmodel.CommentRequest
	err := gc.ShouldBindJSON(&postComment)
	if err != nil {
		zap.S().Error("No content attached to comment", zap.Error(err))
		gc.JSON(http.StatusBadRequest, gin.H{"message": "no content attached to comment"})
		return
	}
	if validate.Struct(postComment) != nil {
		zap.S().Error("Comment exceeds required length")
		gc.JSON(http.StatusBadRequest, gin.H{"message": "comment exceed required length"})
	}

	parsedPostID, err := strconv.Atoi(postId)
	if err != nil {
		zap.S().Error("Invalid post ID format", zap.Error(err))
		gc.JSON(http.StatusBadRequest, gin.H{"message": "invalid post id format"})
		return
	}

	var comment models.PostComment
	var post models.Post
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		defer func() {
			if r := recover(); r != nil || err != nil {
				tx.Rollback()
			}
		}()

		comment = models.PostComment{
			Content:     postComment.Content,
			PostID:      uint(parsedPostID),
			CommenterID: userId,
		}

		if err := tx.Create(&comment).Error; err != nil {
			zap.S().Error("Failed to create comment", zap.Error(err))
			return err
		}

		if err = db.DB.Preload(clause.Associations).
			First(&post, parsedPostID).Error; err != nil {
			return err
		}

		if post.Comments != nil {
			post.Comments = append(post.Comments, comment)
		}

		if err := tx.Save(&post).Error; err != nil {
			zap.S().Error("Failed to save post after comment", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		zap.S().Error("Failed to add comment", zap.Error(err))
		gc.JSON(http.StatusInternalServerError, gin.H{"message": "failed to add comment"})
		return
	}

	zap.S().Info("Comment added to post", zap.Uint("postID", post.ID), zap.String("commentContent", comment.Content))
	gc.JSON(http.StatusCreated, gin.H{"message": "comment added to post", "comment": comment, "post": post})
}
