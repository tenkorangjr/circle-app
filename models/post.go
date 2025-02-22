package models

import "gorm.io/gorm"

type Post struct {
	gorm.Model

	ImageURL string
	UserID   uint
	User     User
}

type PostComment struct {
	gorm.Model

	Content     string
	PostID      uint
	Post        Post
	CommenterID uint
}

type PostLike struct {
	gorm.Model

	PostID  uint
	Post    Post
	LikerID uint
}
