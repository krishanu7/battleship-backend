package leaderboard

import (
	"database/sql"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

type LeaderboardEntry struct {
	PlayerID  string    `json:"player_id"`
	Username  string    `json:"username"`
	Wins      int       `json:"wins"`
	Losses    int       `json:"losses"`
	Elo       int       `json:"elo"`
	UpdatedAt string    `json:"updated_at"`
}

func (s *Service) GetLeaderboard(limit int) ([]LeaderboardEntry, error) {
	rows, err := s.db.Query(`
		SELECT s.player_id, u.username, s.wins, s.losses, s.elo, s.updated_at
		FROM stats s
		JOIN users u ON s.player_id = u.id
		ORDER BY s.elo DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(&entry.PlayerID, &entry.Username, &entry.Wins, &entry.Losses, &entry.Elo, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		leaderboard = append(leaderboard, entry)
	}
	return leaderboard, nil
}
