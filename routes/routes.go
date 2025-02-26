package routes

import "github.com/gin-gonic/gin"

func RegisterRoutes(server *gin.Engine) {
	// user routes
	server.POST("/signup", signUp)
	server.POST("/signin", signIn)

	authenticated := server.Group("/")
	authenticated.POST("/posts")
}
