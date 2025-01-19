package validator

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Jamolkhon5/mistral/internal/ai/project/models"
)

var (
	nameRegex  = regexp.MustCompile(`^[а-яА-Яa-zA-Z0-9\s\-_]+$`)
	emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

const (
	MinNameLength        = 3
	MaxNameLength        = 100
	MinDescriptionLength = 10
	MaxDescriptionLength = 3000
)

// ValidateProjectData проверяет все данные проекта
func ValidateProjectData(data *models.ProjectData) models.ValidationState {
	state := models.ValidationState{
		Errors:   make(map[string]string),
		Warnings: make(map[string]string),
	}

	// Валидация названия
	if err := validateName(data.Name); err != nil {
		state.Errors["name"] = err.Error()
	}

	// Валидация описания
	if err := validateDescription(data.Description); err != nil {
		state.Errors["description"] = err.Error()
	}

	// Валидация дедлайна
	if err := validateDeadline(data.Deadline); err != nil {
		state.Errors["deadline"] = err.Error()
	}

	// Валидация приоритета
	if err := validatePriority(data.Priority); err != nil {
		state.Errors["priority"] = err.Error()
	}

	// Валидация команды
	if len(data.Team) > 0 {
		for i, member := range data.Team {
			if err := validateTeamMember(member); err != nil {
				state.Errors[fmt.Sprintf("team[%d]", i)] = err.Error()
			}
		}
	}

	state.IsValid = len(state.Errors) == 0
	return state
}

func validateName(name string) error {
	name = strings.TrimSpace(name)
	if len(name) < MinNameLength {
		return fmt.Errorf("название должно содержать минимум %d символа", MinNameLength)
	}
	if len(name) > MaxNameLength {
		return fmt.Errorf("название не может быть длиннее %d символов", MaxNameLength)
	}
	if !nameRegex.MatchString(name) {
		return fmt.Errorf("название может содержать только буквы, цифры, пробелы, тире и подчеркивания")
	}
	return nil
}

func validateDescription(description string) error {
	description = strings.TrimSpace(description)
	if len(description) < MinDescriptionLength {
		return fmt.Errorf("описание должно содержать минимум %d символов", MinDescriptionLength)
	}
	if len(description) > MaxDescriptionLength {
		return fmt.Errorf("описание не может быть длиннее %d символов", MaxDescriptionLength)
	}
	return nil
}

func validateDeadline(deadline string) error {
	if deadline == "" {
		return fmt.Errorf("дедлайн обязателен")
	}

	date, err := time.Parse("02.01.2006", deadline)
	if err != nil {
		return fmt.Errorf("неверный формат даты, используйте ДД.ММ.ГГГГ")
	}

	if date.Before(time.Now()) {
		return fmt.Errorf("дата не может быть в прошлом")
	}

	return nil
}

func validatePriority(priority string) error {
	priority = strings.ToUpper(priority)
	validPriorities := map[string]bool{
		"ВЫСОКИЙ": true,
		"СРЕДНИЙ": true,
		"НИЗКИЙ":  true,
	}

	if !validPriorities[priority] {
		return fmt.Errorf("некорректный приоритет, допустимые значения: ВЫСОКИЙ, СРЕДНИЙ, НИЗКИЙ")
	}

	return nil
}

func validateTeamMember(member models.TeamMember) error {
	if !emailRegex.MatchString(member.Email) {
		return fmt.Errorf("некорректный email: %s", member.Email)
	}

	validRoles := map[string]bool{
		"MANAGER": true,
		"EDITOR":  true,
		"READER":  true,
	}

	if !validRoles[member.Role] {
		return fmt.Errorf("некорректная роль для %s, допустимые значения: MANAGER, EDITOR, READER", member.Email)
	}

	if strings.TrimSpace(member.Name) == "" {
		return fmt.Errorf("имя участника обязательно")
	}

	if strings.TrimSpace(member.Lastname) == "" {
		return fmt.Errorf("фамилия участника обязательна")
	}

	return nil
}

// ValidateProjectStep проверяет данные конкретного шага
func ValidateProjectStep(step string, data *models.ProjectData) error {
	switch step {
	case "name":
		return validateName(data.Name)
	case "description":
		return validateDescription(data.Description)
	case "deadline":
		return validateDeadline(data.Deadline)
	case "priority":
		return validatePriority(data.Priority)
	case "team":
		if len(data.Team) > 0 {
			for _, member := range data.Team {
				if err := validateTeamMember(member); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("неизвестный шаг создания проекта: %s", step)
	}
}
