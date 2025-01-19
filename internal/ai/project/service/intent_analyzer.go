package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Intent struct {
	Type    string
	Content string
	Extra   map[string]string
}

type IntentAnalyzer struct {
	dateRegex     *regexp.Regexp
	priorityWords map[string]string
	helpWords     []string
}

func NewIntentAnalyzer() *IntentAnalyzer {
	return &IntentAnalyzer{
		dateRegex: regexp.MustCompile(`(\d{2})[-./](\d{2})[-./](\d{4})`),
		priorityWords: map[string]string{
			"срочн":    "ВЫСОКИЙ",
			"критичн":  "ВЫСОКИЙ",
			"важн":     "ВЫСОКИЙ",
			"средн":    "СРЕДНИЙ",
			"нормальн": "СРЕДНИЙ",
			"обычн":    "СРЕДНИЙ",
			"низк":     "НИЗКИЙ",
			"потом":    "НИЗКИЙ",
			"несрочн":  "НИЗКИЙ",
		},
		helpWords: []string{
			"помоги", "придумай", "сгенерируй", "посоветуй", "предложи",
			"как", "что", "зачем", "почему", "когда",
		},
	}
}

func (ia *IntentAnalyzer) AnalyzeMessage(message string) Intent {
	message = strings.ToLower(message)

	// Проверка на запрос помощи
	if ia.containsAny(message, ia.helpWords) {
		if strings.Contains(message, "описани") {
			return Intent{Type: "generate_description", Content: message}
		}
		if strings.Contains(message, "назван") {
			return Intent{Type: "generate_name", Content: message}
		}
		return Intent{Type: "help", Content: message}
	}

	// Проверка на дату
	if date := ia.extractDate(message); date != "" {
		return Intent{Type: "date", Content: date}
	}

	// Проверка на приоритет
	if priority := ia.detectPriority(message); priority != "" {
		return Intent{Type: "priority", Content: priority}
	}

	// Проверка на email
	if email := ia.extractEmail(message); email != "" {
		return Intent{Type: "email", Content: email}
	}

	return Intent{Type: "text", Content: message}
}

func (ia *IntentAnalyzer) extractDate(message string) string {
	matches := ia.dateRegex.FindStringSubmatch(message)
	if len(matches) == 4 {
		// Преобразуем любой формат в ДД.ММ.ГГГГ
		return fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3])
	}
	return ""
}

func (ia *IntentAnalyzer) detectPriority(message string) string {
	for word, priority := range ia.priorityWords {
		if strings.Contains(message, word) {
			return priority
		}
	}
	return ""
}

func (ia *IntentAnalyzer) extractEmail(message string) string {
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	if email := emailRegex.FindString(message); email != "" {
		return email
	}
	return ""
}

func (ia *IntentAnalyzer) containsAny(message string, words []string) bool {
	for _, word := range words {
		if strings.Contains(message, word) {
			return true
		}
	}
	return false
}

// Дополнительные методы анализа
func (ia *IntentAnalyzer) AnalyzeContext(message string, currentStep string) (string, map[string]string) {
	context := make(map[string]string)

	switch currentStep {
	case "name":
		// Анализ возможных улучшений названия
		words := strings.Fields(message)
		if len(words) < 2 {
			context["suggestion"] = "maybe_extend"
		}
		if strings.Contains(message, "проект") {
			context["has_project_word"] = "true"
		}

	case "description":
		// Анализ информативности описания
		words := strings.Fields(message)
		if len(words) < 10 {
			context["suggestion"] = "too_short"
		}
		if !strings.Contains(strings.ToLower(message), "цель") {
			context["missing"] = "goals"
		}

	case "deadline":
		// Анализ адекватности дедлайна
		if date := ia.extractDate(message); date != "" {
			parsedDate, _ := time.Parse("02.01.2006", date)
			if parsedDate.Sub(time.Now()) < 7*24*time.Hour {
				context["warning"] = "too_soon"
			}
		}
	}

	return currentStep, context
}
