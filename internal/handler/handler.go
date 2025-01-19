package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/Jamolkhon5/mistral/internal/auth"
	"github.com/Jamolkhon5/mistral/internal/models"
	"github.com/Jamolkhon5/mistral/internal/repository"
)

type Handler struct {
	repo          *repository.Repository
	mistralApiKey string
	modelName     string
}

func NewHandler(repo *repository.Repository, mistralApiKey, modelName string) *Handler {
	return &Handler{
		repo:          repo,
		mistralApiKey: mistralApiKey,
		modelName:     modelName,
	}
}

func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.VerifyToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	log.Printf("Received messages: %+v", req.Messages)
	// Проверяем наличие сообщений
	if len(req.Messages) == 0 {
		http.Error(w, "No messages provided", http.StatusBadRequest)
		return
	}

	// Получаем последние сообщения из истории
	lastMessages, err := h.repo.GetLastMessages(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Проверка количества токенов
	tokens, err := h.repo.CountUserTokens(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if tokens >= 20000 {
		http.Error(w, "Token limit exceeded", http.StatusBadRequest)
		return
	}

	// Формируем контекст для запроса к Mistral
	messages := make([]models.Message, 0)
	messages = append(messages, lastMessages...)
	messages = append(messages, req.Messages...)

	// Подготовка запроса к Mistral API
	mistralReq := map[string]interface{}{
		"model":    h.modelName,
		"messages": messages,
	}

	jsonData, err := json.Marshal(mistralReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Sending request to Mistral API: %s", string(jsonData))
	// Отправка запроса к Mistral API
	mistralResp, err := h.sendMistralRequest(jsonData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Сохраняем новое сообщение пользователя
	newUserMessage := req.Messages[len(req.Messages)-1]
	if err := h.repo.SaveMessage(userID, newUserMessage.Content, newUserMessage.Role); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Сохраняем ответ ассистента
	if err := h.repo.SaveMessage(userID, mistralResp, "assistant"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.ChatResponse{
		Response: mistralResp,
	})

	updatedTokens, err := h.repo.CountUserTokens(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.ChatResponse{
		Response: mistralResp,
		Tokens:   updatedTokens,
	})

}

func (h *Handler) ClearHistory(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.VerifyToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.repo.ClearUserHistory(userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем текущее количество токенов (должно быть 0 после очистки)
	tokens, err := h.repo.CountUserTokens(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "История успешно очищена",
		"tokens":  tokens,
	})
}

func (h *Handler) sendMistralRequest(jsonData []byte) (string, error) {
	req, err := http.NewRequest("POST", "https://api.mistral.ai/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.mistralApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	// Добавляем логирование для отладки
	log.Printf("Mistral API response: %s", string(body))

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %w, body: %s", err, string(body))
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response, body: %s", string(body))
	}

	return result.Choices[0].Message.Content, nil
}
