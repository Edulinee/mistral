package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Jamolkhon5/mistral/internal/ai/project/models"
	"github.com/Jamolkhon5/mistral/internal/ai/project/prompts"
	"github.com/Jamolkhon5/mistral/internal/ai/project/validator"
)

type ProjectAssistant struct {
	mistralApiKey string
	modelName     string
}

func NewProjectAssistant(apiKey, modelName string) *ProjectAssistant {
	return &ProjectAssistant{
		mistralApiKey: apiKey,
		modelName:     modelName,
	}
}

// HandleMessage обрабатывает сообщение пользователя и возвращает ответ ассистента
func (pa *ProjectAssistant) HandleMessage(userMessage string, context *models.ProjectCreationContext) (*models.AssistantResponse, error) {
	// Если контекст не определен или пустой, инициализируем новый
	if context == nil || context.CurrentStep == "" {
		context = &models.ProjectCreationContext{
			CurrentStep: "name",
			ProjectData: &models.ProjectData{
				Status:          "В_ПРОЦЕССЕ",
				Priority:        "СРЕДНИЙ",
				Budget:          "0",
				Spent:           "0",
				Confidentiality: "Только для участников",
				Progress:        0,
				Team:            make([]models.TeamMember, 0),
			},
		}
		return &models.AssistantResponse{
			Message:        prompts.WelcomeMessage,
			ProjectContext: *context,
		}, nil
	}

	// Обработка запроса на генерацию описания
	if strings.Contains(strings.ToLower(userMessage), "сгенерируй описание") {
		return pa.handleDescriptionGeneration(userMessage, context)
	}

	// Добавляем логирование для отладки
	log.Printf("Обработка шага: %s с сообщением: %s", context.CurrentStep, userMessage)

	// Обработка текущего шага
	switch context.CurrentStep {
	case "name":
		return pa.handleNameStep(userMessage, context)
	case "description":
		return pa.handleDescriptionStep(userMessage, context)
	case "deadline":
		return pa.handleDeadlineStep(userMessage, context)
	case "priority":
		return pa.handlePriorityStep(userMessage, context)
	case "team":
		return pa.handleTeamStep(userMessage, context)
	case "confirmation":
		return pa.handleConfirmationStep(userMessage, context)
	default:
		return nil, fmt.Errorf("неизвестный шаг: %s", context.CurrentStep)
	}
}

func (pa *ProjectAssistant) handleNameStep(userMessage string, context *models.ProjectCreationContext) (*models.AssistantResponse, error) {
	context.ProjectData.Name = strings.TrimSpace(userMessage)

	if err := validator.ValidateProjectStep("name", context.ProjectData); err != nil {
		return &models.AssistantResponse{
			Message:        fmt.Sprintf("❌ %s\n\nПожалуйста, введите корректное название проекта.", err.Error()),
			ProjectContext: *context,
		}, nil
	}

	context.CurrentStep = "description"
	return &models.AssistantResponse{
		Message:        prompts.DescriptionPrompt,
		ProjectContext: *context,
	}, nil
}

func (pa *ProjectAssistant) handleDescriptionStep(userMessage string, context *models.ProjectCreationContext) (*models.AssistantResponse, error) {
	context.ProjectData.Description = strings.TrimSpace(userMessage)

	if err := validator.ValidateProjectStep("description", context.ProjectData); err != nil {
		return &models.AssistantResponse{
			Message:        fmt.Sprintf("❌ %s\n\nПожалуйста, введите корректное описание проекта.", err.Error()),
			ProjectContext: *context,
		}, nil
	}

	context.CurrentStep = "deadline"
	return &models.AssistantResponse{
		Message:        prompts.DeadlinePrompt,
		ProjectContext: *context,
	}, nil
}

func (pa *ProjectAssistant) handleDeadlineStep(userMessage string, context *models.ProjectCreationContext) (*models.AssistantResponse, error) {
	context.ProjectData.Deadline = strings.TrimSpace(userMessage)

	if err := validator.ValidateProjectStep("deadline", context.ProjectData); err != nil {
		return &models.AssistantResponse{
			Message:        fmt.Sprintf("❌ %s\n\nПожалуйста, введите корректную дату в формате ДД.ММ.ГГГГ.", err.Error()),
			ProjectContext: *context,
		}, nil
	}

	context.CurrentStep = "priority"
	return &models.AssistantResponse{
		Message:        prompts.PriorityPrompt,
		ProjectContext: *context,
	}, nil
}

func (pa *ProjectAssistant) handlePriorityStep(userMessage string, context *models.ProjectCreationContext) (*models.AssistantResponse, error) {
	priority := strings.ToUpper(strings.TrimSpace(userMessage))
	switch priority {
	case "ВЫСОКИЙ", "СРЕДНИЙ", "НИЗКИЙ":
		context.ProjectData.Priority = priority
	default:
		return &models.AssistantResponse{
			Message:        "❌ Пожалуйста, выберите один из вариантов: высокий, средний или низкий.",
			ProjectContext: *context,
		}, nil
	}

	context.CurrentStep = "team"
	return &models.AssistantResponse{
		Message:        prompts.TeamPrompt,
		ProjectContext: *context,
	}, nil
}

func (pa *ProjectAssistant) handleTeamStep(userMessage string, context *models.ProjectCreationContext) (*models.AssistantResponse, error) {
	if strings.ToLower(userMessage) == "готово" {
		context.CurrentStep = "confirmation"
		summary := prompts.GetProjectDataSummary(context.ProjectData)
		return &models.AssistantResponse{
			Message:        fmt.Sprintf(prompts.ConfirmationPrompt, summary),
			ProjectContext: *context,
		}, nil
	}

	// Логика добавления участника команды...
	// Здесь должна быть интеграция с сервисом аутентификации для поиска пользователей

	return &models.AssistantResponse{
		Message:        "Введите email следующего участника или напишите 'готово' для завершения.",
		ProjectContext: *context,
	}, nil
}

func (pa *ProjectAssistant) handleConfirmationStep(userMessage string, context *models.ProjectCreationContext) (*models.AssistantResponse, error) {
	answer := strings.ToLower(strings.TrimSpace(userMessage))

	if answer == "да" {
		// Финальная валидация всех данных
		validationState := validator.ValidateProjectData(context.ProjectData)
		if !validationState.IsValid {
			errorMessages := make([]string, 0)
			for field, err := range validationState.Errors {
				errorMessages = append(errorMessages, fmt.Sprintf("- %s: %s", field, err))
			}
			return &models.AssistantResponse{
				Message:        fmt.Sprintf("❌ Обнаружены ошибки:\n%s\n\nПожалуйста, исправьте их и попробуйте снова.", strings.Join(errorMessages, "\n")),
				ProjectContext: *context,
			}, nil
		}

		return &models.AssistantResponse{
			Message:         "✅ Отлично! Создаю проект...",
			ProjectContext:  *context,
			SuggestedAction: "create_project",
		}, nil
	} else if answer == "нет" {
		context.CurrentStep = "name"
		return &models.AssistantResponse{
			Message:        "Хорошо, давайте начнем сначала. Как назовем проект?",
			ProjectContext: *context,
		}, nil
	}

	return &models.AssistantResponse{
		Message:        "Пожалуйста, ответьте 'да' или 'нет'.",
		ProjectContext: *context,
	}, nil
}

func (pa *ProjectAssistant) handleDescriptionGeneration(userMessage string, context *models.ProjectCreationContext) (*models.AssistantResponse, error) {
	// Формируем промпт для Mistral API
	messages := []models.AssistantMessage{
		{
			Role:    "system",
			Content: "Ты - специалист по написанию описаний проектов. Твоя задача - создать четкое и структурированное описание проекта на основе краткой информации от пользователя.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Создай подробное описание проекта на основе этой информации: %s", userMessage),
		},
	}

	// Отправляем запрос к Mistral API
	response, err := pa.SendMistralRequest(messages)
	if err != nil {
		return nil, fmt.Errorf("ошибка при генерации описания: %w", err)
	}

	// Сохраняем сгенерированное описание
	context.ProjectData.Description = response

	return &models.AssistantResponse{
		Message:        fmt.Sprintf("✨ Я сгенерировал следующее описание для вашего проекта:\n\n%s\n\nХотите использовать это описание? Ответьте 'да' или 'нет'.", response),
		ProjectContext: *context,
	}, nil
}

func (pa *ProjectAssistant) SendMistralRequest(messages []models.AssistantMessage) (string, error) {
	requestBody := map[string]interface{}{
		"model":    pa.modelName,
		"messages": messages,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("ошибка при маршалинге запроса: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.mistral.ai/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка при создании запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+pa.mistralApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка при отправке запроса: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ошибка при декодировании ответа: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("пустой ответ от API")
	}

	return result.Choices[0].Message.Content, nil
}
