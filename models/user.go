package models

import "gorm.io/gorm"

type User struct {
	gorm.Model

	Email    string  `binding:"required"`
	Password string  `binding:"required"`
	Friends  []*User `gorm:"many2many:user_friends;"`
}

func NewUser(email, password string) *User {
	return &User{
		Email:    email,
		Password: password,
	}
}
