package routes

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tenkorangjr/circle-app/middleware"
	"github.com/tenkorangjr/circle-app/routes/websockets"
)

func RegisterRoutes(server *gin.Engine) {
	server.Use(middleware.RateLimiter(5, time.Second))
	// user routes
	server.POST("/signup", signUp)
	server.POST("/signin", signIn)

	authenticated := server.Group("/")
	authenticated.Use(middleware.Authenticate)
	authenticated.POST("/posts", createPost)
	authenticated.GET("/:id/:postid", getPostbyUserAndPostID)
	authenticated.POST("/:postid/comment", postComment)
	authenticated.POST("/:postid/like", postLike)
	authenticated.GET("/chat", websockets.HandleWs)
}
