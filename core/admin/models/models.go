package models

import "time"

type User struct {
	Id        int       `json:"id,omitempty" orm:"pk"`
	Uuid      string    `json:"uuid,omitempty" orm:"size:40"`
	Email     string    `json:"email,omitempty" orm:"size:50;iunique"`
	Password  string    `json:"password,omitempty" orm:"size:150"`
	IsAdmin   bool      `json:"is_admin,omitempty" orm:"default:false"`
	Image     string    `json:"image,omitempty" orm:"size:100;default:''"`
	CreatedAt time.Time `json:"created_at,omitempty" orm:"now"`
}
