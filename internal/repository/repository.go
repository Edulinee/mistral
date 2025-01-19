package repository

import (
	"github.com/Jamolkhon5/mistral/internal/models"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetLastMessages(userID string) ([]models.Message, error) {
	query := `
        SELECT role, message as content  
        FROM messages 
        WHERE user_id = $1 
        ORDER BY created_at DESC 
        LIMIT 2`

	var messages []models.Message
	err := r.db.Select(&messages, query, userID)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *Repository) SaveMessage(userID string, content, role string) error {
	query := `
        INSERT INTO messages (user_id, message, role) 
        VALUES ($1, $2, $3)`

	_, err := r.db.Exec(query, userID, content, role)
	return err
}

func (r *Repository) CountUserTokens(userID string) (int, error) {
	query := `
        SELECT COALESCE(SUM(LENGTH(message)), 0) as total_tokens
        FROM messages 
        WHERE user_id = $1`

	var totalTokens int
	err := r.db.Get(&totalTokens, query, userID)
	return totalTokens / 4, err
}
func (r *Repository) ClearUserHistory(userID string) error {
	query := `DELETE FROM messages WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}
