package auth

import (
	"database/sql"
	"fmt"
	"time"

	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/krishanu7/battleship-backend/config"
	"github.com/krishanu7/battleship-backend/db"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db  *sql.DB
	cfg config.Config
}

func NewService(db *sql.DB, cfg config.Config) *Service {
	return &Service{
		db:  db,
		cfg: cfg,
	}
}

func (s *Service) Register(username, email, password string) (db.User, error) {
	if username == "" || password == "" {
		return db.User{},fmt.Errorf("username and password cannot be empty")
	}
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return db.User{}, err
	}
	userID := uuid.New()
	// Correct query with $1 and $2
	query := "INSERT INTO users (id, username, email, password, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, username, email, created_at"
	var user db.User
	// Insert into database
	err = s.db.QueryRow(query, userID, username, email, string(hashedPassword), time.Now()).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)

	if err != nil {
		// Check for unique constraint violation
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			if pqErr.Constraint == "users_username_key" {
				return db.User{}, fmt.Errorf("username already exists")
			}
			if pqErr.Constraint == "users_email_key" {
				return db.User{}, fmt.Errorf("email already exists")
			}
		}
		return db.User{}, err
	}
	user.Password = string(hashedPassword)
	return user, nil
}

func (s *Service) Login(username, password string) (string, error) {
	var user db.User
	err := s.db.QueryRow(`
	SELECT id, username, email, password, created_at 
	FROM users 
	WHERE username = $1
`, username).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)


	if err != nil {
		return "", errors.New("invalid credentials")
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
