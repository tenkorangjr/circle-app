package models

import (
	"github.com/tenkorangjr/circle-app/utils"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	Email    string  `binding:"required" validate:"required,email"`
	Password string  `binding:"required"`
	Friends  []*User `gorm:"many2many:user_friends;"`
}

func NewUser(email, password string) *User {
	return &User{
		Email:    email,
		Password: password,
	}
}

func (u *User) Save(db *gorm.DB) error {
	var err error
	u.Password, err = utils.HashPassword(u.Password)
	if err != nil {
		return err
	}

	result := db.Create(u)

	return result.Error
}
