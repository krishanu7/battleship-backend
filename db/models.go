package db

type User struct {
	ID 	 int64    `json:"id" db:"id"`
	Username string `json:"username" db:"username"`
	Password string `json:"password" db:"password"` // Hashed password
	Email    string `json:"email" db:"email"`
	CreatedAt string `json:"created_at" db:"created_at"`
}