// GORM model + simple DTOs used in handlers.

package models

import "time"

//user represents a user record in the database 
//Gorm tags configure primary key , sizes and constrains
//json tags control how fields serialized in api respone 
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:120;not null" json:"name"` //amybe add uniqueIndex
	Email     string    `gorm:"size:180;uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"size:255;not null" json:"-"` // hashed
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DTOs (request/response)
// RegisterRequest is the expected payload for the register endpoint.
// Gin's binding tags add basic validation rules automatically.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

//expectedd payload for the login endpoint
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

//small resonse object hodl jwt token 
type AuthResponse struct {
	Token string `json:"token"`
}


//update user requst aylpad fpr updating a usr 
//allow parial updates by making fields pointers (nil means "no change")
type UpdateUserRequest struct {
	// Optional new name||email | password; if nil, keep existing. -> omitempty means do not change 
	Name *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
}


//list users query parameters for pagination when listing users 
//we keep i tin models to share between handlesr and service 
type ListUserQuery struct {
Page int `form:"page"` // Page number (1-based). We'll default in handler/service if 0.
Limit int `form:"limit"` // Page size (items per page). We'll clamp sane defaults.

}


//PageUsers-response envelope for list endpoint
type PagedUsers struct {
	Items []User `json:"items"` // Current page of users.
	Total int64  `json:"total"` // Total number of users in DB (for pagination UIs).
	Page  int    `json:"page"`  // Current page number (1-based).
	Limit int    `json:"limit"` // Page size used.
}