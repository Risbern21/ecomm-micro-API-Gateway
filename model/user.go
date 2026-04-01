package model

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/risbern21/api_gateway/internal/database"
)

type Role string

const (
	Buyer  Role = "buyer"
	Seller Role = "seller"
)

func (r Role) String() string {
	return string(r)
}

type User struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey;default:gen_random_uuid()"`
	Username  string    `json:"username" gorm:"column:username;unique"`
	Email     string    `json:"email" gorm:"column:email;unique"`
	Password  string    `json:"password" gorm:"column:password"`
	Role      Role      `json:"role" gorm:"column:role"`
	Address   string    `json:"address" gorm:"column:address"`
	Phone     string    `json:"phone" gorm:"column:phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt time.Time `json:"deleted_at"`
}

func NewUser() *User {
	return &User{}
}

func (u *User) AddUser(ctx context.Context) error {
	return database.Client().Save(&u).Error
}

func (u *User) GetUserByEmail(ctx context.Context, email string) error {
	return database.Client().Table("users").Where("email= ?", email).First(&u).Error
}
