package routes

import "github.com/gin-gonic/gin"

func RegisterRoutes(server *gin.Engine) {
	// user routes
	server.POST("/signup")
	server.POST("/signin")
}
