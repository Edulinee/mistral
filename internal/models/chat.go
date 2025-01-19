package models

type Message struct {
	Role    string `json:"role" db:"role"`
	Content string `json:"content" db:"content"`
}

type ChatRequest struct {
	Messages []Message `json:"messages"`
}

type ChatResponse struct {
	ID        int    `json:"-" db:"id"`
	UserID    string `json:"userId" db:"user_id"`
	Message   string `json:"message" db:"message"`
	Role      string `json:"role" db:"role"`
	UpdatedAt string `json:"updatedAt" db:"updated_at"`
	Response  string `json:"response"`
	Tokens    int    `json:"tokens"`
}

type LastMessages struct {
	Messages []Message
}
