package ai

import (
	"regexp"
	"strings"
)

type fallbackCategory struct {
	markers []string
	ru      string
	kk      string
}

var fallbackCategories = []fallbackCategory{
	{[]string{"сейчас", "вручную", "текущий процесс", "қазір", "қолмен"}, "Как сейчас выполняется эта работа, есть ли трудности?", "Қазір бұл жұмыс қалай орындалады, қиындықтар бар ма?"},
	{[]string{"пользовател", "пользоваться будут", "использовать будут", "пайдаланушы", "қолдануш", "пайдаланады"}, "Кто будет пользоваться решением?", "Шешімді кімдер пайдаланады?"},
	{[]string{"данные", "данных", "excel", "csv", "деректер", "мәліметтер"}, "Какие данные или материалы доступны для задачи?", "Тапсырмаға қандай деректер немесе материалдар бар?"},
	{[]string{"ограничени", "бюджет", "срок", "шектеу", "мерзім"}, "Есть ли ограничения по срокам, бюджету или условиям работы?", "Мерзімге, бюджетке немесе жұмыс шартына шектеулер бар ма?"},
	{[]string{"ожидаем", "результат", "прототип", "нужен чат-бот", "нужен веб", "нәтиже", "керек өнім"}, "Какой результат вы хотите получить от команды?", "Командадан қандай нәтиже алғыңыз келеді?"},
	{[]string{"критери", "успех", "проверим", "считаем готов", "тексер", "табыс өлшем"}, "Как вы проверите, что результат решает вашу задачу?", "Нәтиже тапсырманы шешкенін қалай тексересіз?"},
	{[]string{"контакт", "@", "связь", "связи", "связываться", "байланыс"}, "Как команда сможет связаться с вами и обсуждать работу?", "Команда сізбен қалай байланысып, жұмысты талқылай алады?"},
}

var kazakhWords = regexp.MustCompile(`(?i)(?:^|\s)(?:біз|биз|керек|ушін|ушин|койма|калай|болады|бар)(?:\s|[.,!?]|$)`)

func isKazakh(text string) bool {
	return strings.ContainsAny(strings.ToLower(text), "әғқңөұүһі") || kazakhWords.MatchString(text)
}

// FallbackQuestions uses coarse keyword heuristics, not semantic extraction.
// Generic confirmation prompts fill the minimum of three for detailed inputs;
// they do not assert that the supplied description has missing facts.
func FallbackQuestions(description string) QuestionsResponse {
	kk := isKazakh(description)
	lower := strings.ToLower(description)
	result := QuestionsResponse{Questions: make([]string, 0, 5)}
	for _, category := range fallbackCategories {
		covered := false
		for _, marker := range category.markers {
			if strings.Contains(lower, marker) {
				covered = true
				break
			}
		}
		if !covered {
			question := category.ru
			if kk {
				question = category.kk
			}
			result.Questions = append(result.Questions, question)
		}
		if len(result.Questions) == 5 {
			break
		}
	}
	confirmations := []string{
		"Есть ли в описании сведения, которые нужно исправить?",
		"Нужно ли дополнительно уточнить границы задачи?",
		"Есть ли ещё сведения, которые вы хотите добавить в карточку?",
	}
	if kk {
		confirmations = []string{
			"Сипаттамада түзетуді қажет ететін мәлімет бар ма?",
			"Тапсырманың аясын қосымша нақтылау керек пе?",
			"Карточкаға қосқыңыз келетін басқа мәлімет бар ма?",
		}
	}
	for _, q := range confirmations {
		if len(result.Questions) >= 3 {
			break
		}
		result.Questions = append(result.Questions, q)
	}
	return result
}

// FallbackCard deliberately preserves only the original description.
// Answers must remain available in the host UI for manual transcription.
func FallbackCard(req CardRequest) TaskCard {
	return TaskCard{Context: req.Description}
}
