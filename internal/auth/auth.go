package auth

import (
	"database/sql"
	"fmt"
	"time"

	"errors"

	"github.com/golang-jwt/jwt/v5"
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

func (s *Service) Register(username, password string) error {
	if username == "" || password == "" {
		return fmt.Errorf("username and password cannot be empty")
	}
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	// Correct query with $1 and $2
	query := "INSERT INTO users (username, password) VALUES ($1, $2)"
	// Insert into database
	_, err = s.db.Exec(query, username, string(hashedPassword))
	if err != nil {
		// Check for unique constraint violation
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" {
				return fmt.Errorf("username already exists")
			}
		}
		return err
	}
	return nil
}

func (s *Service) Login(username, password string) (string, error) {
	var user db.User
	err := s.db.QueryRow("SELECT id, username, password FROM users WHERE username = $1", username).Scan(&user.ID, &user.Username, &user.Password)

	if err != nil {
		return "", errors.New("invalid credentials")
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
