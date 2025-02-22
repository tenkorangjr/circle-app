package db

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/tenkorangjr/circle-app/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	err := godotenv.Load()
	if err != nil {
		panic("could not load in the .env")
	}

	dsn := fmt.Sprintf("host=localhost user=%s password=%s dbname=circle port=5432 sslmode=disable TimeZone=Asia/Shanghai", os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to Database")
	}

	fmt.Println("\u2714 Connected to Database successfully")

	err = DB.AutoMigrate(&models.User{}, &models.Post{}, &models.PostLike{}, &models.PostComment{})
	if err != nil {
		log.Fatal("Failed to migrate database schema")
	}
}
