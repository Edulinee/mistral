package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/your_username/mistral/internal/auth"
	"github.com/your_username/mistral/internal/models"
	"github.com/your_username/mistral/internal/repository"
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

	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Получение последних сообщений
	lastMessages, err := h.repo.GetLastMessages(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Добавление контекста из последних сообщений
	messages := append(lastMessages, req.Messages...)

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

	// Отправка запроса к Mistral API
	mistralResp, err := h.sendMistralRequest(jsonData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Сохранение сообщения пользователя и ответа
	if err := h.repo.SaveMessage(userID, req.Messages[len(req.Messages)-1].Content, "user"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.repo.SaveMessage(userID, mistralResp, "assistant"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"response": mistralResp,
	})
}

func (h *Handler) sendMistralRequest(jsonData []byte) (string, error) {
	req, err := http.NewRequest("POST", "https://api.mistral.ai/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.mistralApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Извлекаем ответ из структуры Mistral API
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("invalid response format")
	}

	firstChoice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid choice format")
	}

	message, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid message format")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid content format")
	}

	return content, nil
}
