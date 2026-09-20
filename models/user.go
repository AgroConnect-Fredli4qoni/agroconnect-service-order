package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type RegisterDTO struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponseDTO struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type UpdateProfileDTO struct {
	Name        string `json:"name"`
	OldPassword string `json:"old_password,omitempty"`
	NewPassword string `json:"new_password,omitempty"`
}
