package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/Jamolkhon5/mistral/internal/ai/project/models"
	"github.com/Jamolkhon5/mistral/internal/ai/project/service"
	"github.com/Jamolkhon5/mistral/internal/auth"
)

type ProjectAssistantHandler struct {
	assistant *service.ProjectAssistant
}

func NewProjectAssistantHandler(mistralApiKey, modelName string) *ProjectAssistantHandler {
	return &ProjectAssistantHandler{
		assistant: service.NewProjectAssistant(mistralApiKey, modelName),
	}
}

func (h *ProjectAssistantHandler) ChatWithAssistant(w http.ResponseWriter, r *http.Request) {
	// Проверка авторизации
	userID, err := auth.VerifyToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Декодируем запрос
	var req struct {
		Message string                         `json:"message"`
		Context *models.ProjectCreationContext `json:"context,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Обработка сообщения ассистентом
	response, err := h.assistant.HandleMessage(req.Message, req.Context)
	if err != nil {
		log.Printf("Error handling message: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Если есть подсказка к действию "create_project", создаем проект
	if response.SuggestedAction == "create_project" {
		if err := h.createProject(userID, response.ProjectContext.ProjectData); err != nil {
			log.Printf("Error creating project: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ProjectAssistantHandler) GenerateDescription(w http.ResponseWriter, r *http.Request) {
	// Проверка авторизации
	_, err := auth.VerifyToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Декодируем запрос
	var req struct {
		ProjectInfo string `json:"project_info"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Получаем ответ от Mistral API
	messages := []models.AssistantMessage{
		{
			Role: "system",
			Content: `Ты - опытный менеджер проектов и специалист по написанию проектной документации. 
                    Твоя задача - создать четкое, структурированное и профессиональное описание проекта.
                    
                    Описание должно включать:
                    - Цели и задачи проекта
                    - Ожидаемые результаты
                    - Основные этапы реализации
                    - Используемые технологии или методологии
                    
                    Формат описания:
                    - Используй деловой стиль
                    - Разбивай текст на логические абзацы
                    - Используй маркированные списки где уместно`,
		},
		{
			Role:    "user",
			Content: req.ProjectInfo,
		},
	}

	response, err := h.assistant.SendMistralRequest(messages)
	if err != nil {
		log.Printf("Ошибка генерации описания: %v", err)
		http.Error(w, "Не удалось сгенерировать описание проекта", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"description": response,
	}); err != nil {
		log.Printf("Ошибка кодирования ответа: %v", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}
}

func (h *ProjectAssistantHandler) createProject(userID string, projectData *models.ProjectData) error {
	// Подготавливаем данные для создания проекта
	projectRequest := map[string]interface{}{
		"name":            projectData.Name,
		"description":     projectData.Description,
		"deadline":        projectData.Deadline,
		"status":          projectData.Status,
		"priority":        projectData.Priority,
		"team":            projectData.Team,
		"budget":          projectData.Budget,
		"spent":           projectData.Spent,
		"confidentiality": projectData.Confidentiality,
		"progress":        projectData.Progress,
	}

	// Создаем JSON
	jsonData, err := json.Marshal(projectRequest)
	if err != nil {
		return err
	}

	// Отправляем запрос к сервису проектов
	req, err := http.NewRequest("POST", "http://project-service:5641/v1/projects", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// Добавляем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", userID)) // Используем ID пользователя как токен

	// Выполняем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to create project: %s", string(body))
	}

	return nil
}

// RegisterRoutes регистрирует маршруты для AI-ассистента
func (h *ProjectAssistantHandler) RegisterRoutes(r chi.Router) {
	r.Post("/ai/project/chat", h.ChatWithAssistant)
	r.Post("/ai/project/generate-description", h.GenerateDescription)
}
