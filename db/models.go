package db

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"password" db:"password"` 
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Stats struct {
	PlayerID   uuid.UUID `json:"player_id" db:"player_id"`
	Wins       int       `json:"wins" db:"wins"`
	Losses     int       `json:"losses" db:"losses"`
	Elo        int       `json:"elo" db:"elo"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}


type Game struct {
	ID        uuid.UUID `json:"id" db:"id"`
	RoomID    string    `json:"room_id" db:"room_id"`
	WinnerID  uuid.UUID `json:"winner_id" db:"winner_id"`
	LoserID   uuid.UUID `json:"loser_id" db:"loser_id"`
	StartedAt time.Time `json:"started_at" db:"started_at"`
	EndedAt   time.Time `json:"ended_at" db:"ended_at"`
}