package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tenkorangjr/circle-app/utils"
)

func Authenticate(context *gin.Context) {
	token := context.GetHeader("authorization")
	if token == "" {
		context.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "unauthorized user"})
		return
	}

	userId, err := utils.ValidateToken(token)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "unauthorized user"})
		return
	}

	context.Set("userId", userId)
	context.Next()
}
