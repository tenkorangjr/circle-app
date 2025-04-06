package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/tenkorangjr/circle-app/middleware"
)

func RegisterRoutes(server *gin.Engine) {
	// user routes
	server.POST("/signup", signUp)
	server.POST("/signin", signIn)

	authenticated := server.Group("/")
	authenticated.Use(middleware.Authenticate)
	authenticated.POST("/posts", createPost)
	authenticated.GET("/:id/:postid", getPostbyUserAndPostID)
	authenticated.POST("/:postid/comment")
	authenticated.POST("/:postid/like")
}
