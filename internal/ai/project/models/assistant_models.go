package models

// AssistantMessage представляет сообщение в диалоге с ассистентом
type AssistantMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AssistantRequest представляет запрос к AI-ассистенту
type AssistantRequest struct {
	Messages []AssistantMessage `json:"messages"`
}

// ProjectCreationContext содержит контекст создания проекта
type ProjectCreationContext struct {
	CurrentStep     string          `json:"current_step"`     // Текущий шаг создания проекта
	ProjectData     *ProjectData    `json:"project_data"`     // Данные проекта
	ValidationState ValidationState `json:"validation_state"` // Состояние валидации
}

// ProjectData содержит данные проекта
type ProjectData struct {
	Name            string       `json:"name"`
	Description     string       `json:"description"`
	Deadline        string       `json:"deadline"`
	Status          string       `json:"status"`
	Priority        string       `json:"priority"`
	Team            []TeamMember `json:"team"`
	Budget          string       `json:"budget"`
	Spent           string       `json:"spent"`
	Confidentiality string       `json:"confidentiality"`
	Progress        int          `json:"progress"`
}

// TeamMember представляет участника команды
type TeamMember struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Lastname string `json:"lastname"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Photo    string `json:"photo"`
}

// ValidationState содержит состояние валидации данных
type ValidationState struct {
	IsValid  bool              `json:"is_valid"`
	Errors   map[string]string `json:"errors"`
	Warnings map[string]string `json:"warnings"`
}

// AssistantResponse представляет ответ от AI-ассистента
type AssistantResponse struct {
	Message         string                 `json:"message"`
	ProjectContext  ProjectCreationContext `json:"project_context"`
	SuggestedAction string                 `json:"suggested_action"`
	Error           string                 `json:"error,omitempty"`
}
