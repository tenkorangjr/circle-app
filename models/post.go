package models

import "gorm.io/gorm"

type Post struct {
	gorm.Model

	ImageURL string
	Caption  string `validate:"max=100"`
	UserID   uint
	User     User
}

func NewPost(imageURL, caption string, userId uint, user User) *Post {
	return &Post{
		ImageURL: imageURL,
		Caption:  caption,
		UserID:   userId,
		User:     user,
	}
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
