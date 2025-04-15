package main

import (
	"github.com/gin-gonic/gin"
	"github.com/tenkorangjr/circle-app/db"
	"github.com/tenkorangjr/circle-app/routes"
	"go.uber.org/zap"
)

func main() {
	gin.ForceConsoleColor()

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	zap.ReplaceGlobals(logger)

	db.InitDB()
	server := gin.Default()

	routes.RegisterRoutes(server)

	server.Run(":8080")
}
