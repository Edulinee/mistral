package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/your_username/mistral/internal/models"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SaveMessage(userID string, message, role string) error {
	query := `
        INSERT INTO messages (user_id, message, role) 
        VALUES ($1, $2, $3)`

	_, err := r.db.Exec(query, userID, message, role)
	return err
}

func (r *Repository) GetLastMessages(userID string) ([]models.Message, error) {
	query := `
        SELECT role, message 
        FROM messages 
        WHERE user_id = $1 
        ORDER BY created_at DESC 
        LIMIT 2`

	var messages []models.Message
	err := r.db.Select(&messages, query, userID)
	return messages, err
}

func (r *Repository) CountUserTokens(userID string) (int, error) {
	query := `
        SELECT COALESCE(SUM(LENGTH(message)), 0) as total_tokens
        FROM messages 
        WHERE user_id = $1`

	var totalTokens int
	err := r.db.Get(&totalTokens, query, userID)
	return totalTokens / 4, err // примерная оценка токенов
}
