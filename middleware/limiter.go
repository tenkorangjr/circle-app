package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RateLimiter(limit int, duration time.Duration) gin.HandlerFunc {
	tokens := make(chan struct{}, limit)

	go func() {
		ticker := time.NewTicker(time.Millisecond * 50)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case tokens <- struct{}{}:
			default:
			}
		}
	}()

	return func(ctx *gin.Context) {
		select {
		case <-tokens:
			ctx.Next()
		default:
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Rate limit exceeded"})
		}
	}
}
