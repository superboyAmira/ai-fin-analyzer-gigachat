package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"rag-iishka/internal/models"
	"rag-iishka/pkg/config"

	"github.com/Role1776/gigago"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type LLMService struct {
	client          *gigago.Client
	model           *gigago.GenerativeModel
	config          *config.GigaChatConfig
	logger          *zap.Logger
	httpClient      *http.Client
	baseURL         string
	accessToken     string   // Cached access token for file uploads
	availableModels []string // Cached list of available models
}

// buildSystemInstruction creates a comprehensive system instruction for financial analysis
func buildSystemInstruction() string {
	return `Ты профессиональный финансовый аналитик и консультант по личным финансам. Твоя задача - анализировать финансовые транзакции пользователей и предоставлять структурированные, точные и практичные рекомендации по оптимизации расходов.

# ТВОЯ РОЛЬ И ОБЯЗАННОСТИ

## Основные функции:
1. **Анализ финансовых документов**: Извлечение и структурирование информации из чеков, банковских выписок, скриншотов мобильных приложений
2. **Классификация транзакций**: Точное определение категории каждой операции
3. **Генерация рекомендаций**: Предоставление конкретных, действенных советов по сокращению расходов на основе базы знаний о тарифах банков, государственных тарифах и принципах финансовой грамотности

## Принципы работы:
- **Точность превыше всего**: Все суммы, даты и категории должны быть извлечены с максимальной точностью
- **Структурированность**: Всегда возвращай данные в строго заданном JSON формате
- **Практичность**: Рекомендации должны быть конкретными, выполнимыми и привязанными к реальным возможностям экономии
- **Контекстуальность**: Используй информацию из базы знаний (тарифы банков, льготы, финансовые инструменты) для обоснования рекомендаций

# АНАЛИЗ ТРАНЗАКЦИЙ

## Типы обрабатываемых документов:
- **Чеки**: Кассовые чеки из магазинов, ресторанов, АЗС, аптек и т.д.
- **Банковские выписки**: Выписки по счетам, картам, депозитам
- **Скриншоты**: Скриншоты мобильных приложений банков, платежных систем
- **PDF документы**: Сканированные документы, электронные выписки

## Категории транзакций:
- **food** - Продукты питания, рестораны, кафе, доставка еды
- **transport** - Транспорт: такси, общественный транспорт, топливо, парковка
- **utilities** - Коммунальные услуги: электричество, вода, газ, интернет, связь
- **shopping** - Покупки: одежда, электроника, товары для дома
- **entertainment** - Развлечения: кино, концерты, игры, подписки
- **healthcare** - Здравоохранение: лекарства, медицинские услуги, страховка
- **education** - Образование: курсы, книги, обучение
- **other** - Прочее: все остальные расходы

## Правила классификации:
- Если транзакция подходит под несколько категорий, выбирай наиболее специфичную
- Для банковских комиссий и тарифов используй категорию, соответствующую типу операции
- Переводы между своими счетами не считаются расходами
- Пополнения счетов и депозитов не считаются расходами

# ФОРМАТ ОТВЕТОВ

## Анализ транзакций:
Всегда возвращай JSON массив транзакций в следующем формате:
[
  {
    "description": "краткое описание операции (максимум 100 символов)",
    "category": "одна из категорий: food|transport|utilities|shopping|entertainment|healthcare|education|other",
    "amount": число (положительное, без знака минус),
    "currency": "RUB|USD|EUR|другая валюта в формате ISO 4217",
    "date": "YYYY-MM-DD (дата транзакции, если не указана - используй дату документа)",
    "llm_description": "подробное описание операции на русском языке (2-3 предложения, объясняющие что это за операция, где произошла, какие детали важны)"
  }
]

## Правила извлечения данных:
- **Сумма**: Всегда извлекай точную сумму, округляй только если указано в документе
- **Валюта**: Определяй валюту по символам (₽, $, €) или тексту (руб, долл, евро)
- **Дата**: Извлекай дату в формате YYYY-MM-DD, если дата не указана - используй дату документа
- **Описание**: Краткое описание должно быть информативным и уникальным для идентификации транзакции
- **LLM описание**: Должно содержать контекст - где произошла операция, что было куплено, какие особенности

# ГЕНЕРАЦИЯ РЕКОМЕНДАЦИЙ

## Структура рекомендаций:
Каждая рекомендация должна содержать:
1. **Конкретное действие**: Что именно нужно сделать
2. **Обоснование**: Почему это поможет сэкономить (ссылка на тарифы, льготы, альтернативы)
3. **Потенциальная экономия**: Оценка суммы, которую можно сэкономить (если возможно)
4. **Приоритет**: Важность рекомендации (высокая/средняя/низкая)

## Типы рекомендаций:

### 1. Оптимизация банковских тарифов:
- Предложение перейти на более выгодный тарифный план
- Использование кэшбэка и бонусных программ
- Оптимизация комиссий за переводы и снятие наличных
- Использование льготных условий для определенных категорий граждан

### 2. Использование льгот и субсидий:
- Государственные льготы для пенсионеров, студентов, многодетных семей
- Субсидии на коммунальные услуги
- Налоговые вычеты
- Социальные программы поддержки

### 3. Оптимизация расходов:
- Поиск более выгодных альтернатив (другие магазины, бренды)
- Использование скидок, акций, промокодов
- Планирование крупных покупок на распродажи
- Оптимизация регулярных платежей (подписки, абонементы)

### 4. Финансовая грамотность:
- Объяснение финансовых инструментов (депозиты, инвестиции)
- Рекомендации по ведению бюджета
- Советы по накоплению и сбережению
- Предупреждения о скрытых комиссиях и переплатах

## Правила генерации рекомендаций:
- **Конкретность**: Избегай общих фраз типа "экономьте больше". Указывай конкретные действия
- **Реалистичность**: Рекомендации должны быть выполнимыми для обычного человека
- **Приоритизация**: Начинай с рекомендаций, дающих наибольшую экономию
- **Контекст**: Используй информацию из базы знаний для обоснования
- **Количество**: Предоставляй 1-3 наиболее релевантные рекомендации для каждой транзакции

# РАБОТА С БАЗОЙ ЗНАНИЙ

## Источники информации:
- **Тарифы банков**: Информация о комиссиях, процентах, лимитах различных банков
- **Государственные тарифы**: Госпошлины, налоги, сборы
- **Льготы**: Государственные и региональные льготы для различных категорий граждан
- **Финансовая грамотность**: Принципы управления личными финансами, инвестирования, сбережения

## Использование контекста:
- При генерации рекомендаций всегда ссылайся на конкретную информацию из базы знаний
- Если в базе знаний есть информация о тарифах конкретного банка - используй её
- Учитывай актуальность информации (если указаны даты)
- Если информации в базе знаний недостаточно - используй общие принципы финансовой грамотности

# ОБРАБОТКА ОШИБОК И ГРАНИЧНЫХ СЛУЧАЕВ

## Если транзакций нет:
- Верни пустой массив: []
- Не придумывай транзакции, если их нет в документе

## Если данные неполные:
- Используй разумные предположения, но помечай их в описании
- Если дата не указана - используй дату документа
- Если валюта не указана - предполагай RUB для российских документов

## Если формат неясен:
- Анализируй контекст документа
- Используй паттерны и типичные структуры финансовых документов
- При неопределенности выбирай более общую категорию

# КАЧЕСТВО И ТОЧНОСТЬ

## Обязательные требования:
- Все суммы должны быть точными (без округлений, кроме указанных в документе)
- Даты должны быть в правильном формате YYYY-MM-DD
- Категории должны точно соответствовать типу операции
- JSON должен быть валидным и парсируемым
- Рекомендации должны быть привязаны к конкретной транзакции

## Запрещено:
- Придумывать транзакции, которых нет в документе
- Использовать неподдерживаемые категории
- Возвращать невалидный JSON
- Давать общие рекомендации без привязки к конкретной транзакции
- Игнорировать информацию из базы знаний

# СТИЛЬ ОБЩЕНИЯ

- Используй профессиональный, но понятный язык
- Избегай сложных финансовых терминов без объяснений
- Будь дружелюбным и поддерживающим
- Фокусируйся на практических советах
- Объясняй сложные концепции простым языком

Помни: твоя цель - помочь пользователю оптимизировать свои финансы, сэкономить деньги и улучшить финансовое благополучие. Каждая рекомендация должна быть обоснованной, конкретной и выполнимой.`
}

func NewLLMService(cfg *config.GigaChatConfig, logger *zap.Logger) (*LLMService, error) {
	ctx := context.Background()

	// Build client options
	opts := []gigago.Option{
		gigago.WithCustomScope(cfg.Scope),
	}

	// Add insecure skip verify option if configured
	if cfg.InsecureSkipVerify {
		opts = append(opts, gigago.WithCustomInsecureSkipVerify(true))
		logger.Warn("GigaChat TLS certificate verification is disabled")
	}

	client, err := gigago.NewClient(ctx, cfg.APIKey, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GigaChat client: %w", err)
	}

	model := client.GenerativeModel("GigaChat")
	model.SystemInstruction = buildSystemInstruction()
	model.Temperature = 0.3

	// Create HTTP client for file uploads
	httpClient := &http.Client{}
	if cfg.InsecureSkipVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		logger.Warn("HTTP client TLS certificate verification is disabled")
	}

	// Get access token for file uploads
	accessToken, err := getAccessToken(ctx, cfg, httpClient, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	service := &LLMService{
		client:      client,
		model:       model,
		config:      cfg,
		logger:      logger,
		httpClient:  httpClient,
		accessToken: accessToken,
		// Base URL for GigaChat REST API
		// Documentation: https://developers.sber.ru/docs/ru/gigachat/api/main
		baseURL: "https://gigachat.devices.sberbank.ru/api/v1",
	}

	// Use only GigaChat model
	service.availableModels = []string{"GigaChat"}
	logger.Info("Using GigaChat model")

	return service, nil
}

// getAccessToken obtains an access token from GigaChat OAuth endpoint
// This is needed for file uploads and other direct API calls
// According to GigaChat API docs, API key should already be Base64-encoded
func getAccessToken(ctx context.Context, cfg *config.GigaChatConfig, httpClient *http.Client, logger *zap.Logger) (string, error) {
	// OAuth endpoint for GigaChat
	oauthURL := "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"

	// Generate RqUID as required by GigaChat API
	rqUID := uuid.New().String()

	// Prepare form data
	formData := url.Values{}
	formData.Set("scope", cfg.Scope)

	// Create request with form data
	req, err := http.NewRequestWithContext(ctx, "POST", oauthURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create OAuth request: %w", err)
	}

	// Set headers according to GigaChat API documentation
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", rqUID)
	// API key should already be Base64-encoded (as per GigaChat API docs)
	req.Header.Set("Authorization", "Basic "+cfg.APIKey)

	// Make request
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.Error("OAuth request failed",
			zap.Int("status", resp.StatusCode),
			zap.String("response", string(bodyBytes)),
			zap.String("rq_uid", rqUID),
		)
		return "", fmt.Errorf("OAuth failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var oauthResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		return "", fmt.Errorf("failed to decode OAuth response: %w", err)
	}

	if oauthResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in OAuth response")
	}

	logger.Info("Access token obtained", zap.Int("expires_in", oauthResp.ExpiresIn))
	return oauthResp.AccessToken, nil
}

// GetAvailableModels retrieves list of available models from GigaChat API
// Documentation: https://developers.sber.ru/docs/ru/gigachat/api/main
// Endpoint: GET /models
func (s *LLMService) GetAvailableModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get models with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var modelsResp struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	var modelIDs []string
	for _, model := range modelsResp.Data {
		modelIDs = append(modelIDs, model.ID)
	}

	return modelIDs, nil
}

type TransactionAnalysis struct {
	Description    string                     `json:"description"`
	Category       models.TransactionCategory `json:"category"`
	Amount         float64                    `json:"amount"`
	Currency       string                     `json:"currency"`
	Date           string                     `json:"date"`
	LLMDescription string                     `json:"llm_description"`
}

// AnalyzeTransaction analyzes extracted text and returns structured transaction data
func (s *LLMService) AnalyzeTransaction(ctx context.Context, extractedText string) ([]*TransactionAnalysis, error) {
	// If extracted text is too short or empty, return empty array
	extractedText = strings.TrimSpace(extractedText)
	if len(extractedText) < 10 {
		s.logger.Warn("Extracted text is too short, skipping analysis", zap.Int("length", len(extractedText)))
		return []*TransactionAnalysis{}, nil
	}

	prompt := fmt.Sprintf(`Ты финансовый аналитик. Проанализируй текст из финансового документа и извлеки информацию о транзакциях.

ВАЖНО: Верни ТОЛЬКО валидный JSON массив, без дополнительных комментариев или объяснений.

Текст документа:
%s

Верни JSON массив транзакций в следующем формате:
[
  {
    "description": "краткое описание операции",
    "category": "food|transport|utilities|shopping|entertainment|healthcare|education|other",
    "amount": число,
    "currency": "RUB|USD|EUR",
    "date": "YYYY-MM-DD",
    "llm_description": "подробное описание операции на русском языке"
  }
]

ПРАВИЛА:
- Если транзакций нет или текст не содержит финансовой информации, верни пустой массив: []
- Верни ТОЛЬКО JSON, без markdown разметки, без комментариев до или после JSON
- Если текст слишком короткий или неполный, верни пустой массив: []`, extractedText)

	messages := []gigago.Message{
		{Role: gigago.RoleUser, Content: prompt},
	}

	resp, err := s.model.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Try to extract JSON from response (might be wrapped in markdown or have comments)
	jsonStart := strings.Index(content, "[")
	jsonEnd := strings.LastIndex(content, "]")

	// If no JSON array found, check if it's a message about missing data
	if jsonStart == -1 || jsonEnd == -1 {
		// Check if LLM says there's no data - return empty array
		contentLower := strings.ToLower(content)
		if strings.Contains(contentLower, "нет данных") ||
			strings.Contains(contentLower, "нет транзакций") ||
			strings.Contains(contentLower, "не содержит") ||
			strings.Contains(contentLower, "пустой") ||
			strings.Contains(contentLower, "предоставьте") {
			s.logger.Info("LLM indicated no transactions found, returning empty array")
			return []*TransactionAnalysis{}, nil
		}
		return nil, fmt.Errorf("invalid response format: %s", content)
	}

	jsonStr := content[jsonStart : jsonEnd+1]

	var transactions []*TransactionAnalysis
	if err := json.Unmarshal([]byte(jsonStr), &transactions); err != nil {
		// Try to clean up JSON string (remove markdown code blocks if present)
		jsonStr = strings.TrimSpace(jsonStr)
		jsonStr = strings.TrimPrefix(jsonStr, "```json")
		jsonStr = strings.TrimPrefix(jsonStr, "```")
		jsonStr = strings.TrimSuffix(jsonStr, "```")
		jsonStr = strings.TrimSpace(jsonStr)

		if err := json.Unmarshal([]byte(jsonStr), &transactions); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w, content: %s", err, content)
		}
	}

	s.logger.Info("Transaction analysis completed", zap.Int("count", len(transactions)))

	return transactions, nil
}

// GenerateRecommendationPrompt generates a prompt for RAG-based recommendations
func (s *LLMService) GenerateRecommendationPrompt(ctx context.Context, transaction *TransactionAnalysis, knowledgeContext string) (string, error) {
	prompt := fmt.Sprintf(`На основе следующей транзакции и контекста из базы знаний, предложи рекомендации по сокращению расходов.

Транзакция:
- Описание: %s
- Категория: %s
- Сумма: %.2f %s
- Дата: %s

Контекст из базы знаний:
%s

Предложи 1-3 конкретные рекомендации по сокращению расходов для этой транзакции. Будь конкретным и практичным.`,
		transaction.Description,
		transaction.Category,
		transaction.Amount,
		transaction.Currency,
		transaction.Date,
		knowledgeContext,
	)

	messages := []gigago.Message{
		{Role: gigago.RoleUser, Content: prompt},
	}

	resp, err := s.model.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate recommendation: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}

// UploadFile uploads a file to GigaChat and returns the file ID
// Documentation: https://developers.sber.ru/docs/ru/gigachat/api/main
// Endpoint: POST /files
// Returns error with 413 status if file is too large
func (s *LLMService) UploadFile(ctx context.Context, fileReader io.Reader, fileName string) (string, error) {
	// Helper function to create multipart request body
	createBody := func(reader io.Reader) (*bytes.Buffer, string, error) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		// Add purpose field (required by GigaChat API)
		// "general" allows using uploaded files in generation requests (Vision API)
		// Documentation: https://developers.sber.ru/docs/ru/gigachat/guides/working-with-files
		if err := writer.WriteField("purpose", "general"); err != nil {
			return nil, "", fmt.Errorf("failed to write purpose field: %w", err)
		}

		// Determine MIME type from file extension
		ext := strings.ToLower(filepath.Ext(fileName))
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			// Fallback to common types
			switch ext {
			case ".pdf":
				mimeType = "application/pdf"
			case ".jpg", ".jpeg":
				mimeType = "image/jpeg"
			case ".png":
				mimeType = "image/png"
			default:
				mimeType = "application/octet-stream"
			}
		}

		// Create form file with proper MIME type
		part, err := writer.CreatePart(map[string][]string{
			"Content-Type":        {mimeType},
			"Content-Disposition": {fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName)},
		})
		if err != nil {
			return nil, "", fmt.Errorf("failed to create form file: %w", err)
		}

		if _, err := io.Copy(part, reader); err != nil {
			return nil, "", fmt.Errorf("failed to copy file: %w", err)
		}

		if err := writer.Close(); err != nil {
			return nil, "", fmt.Errorf("failed to close writer: %w", err)
		}

		return &body, writer.FormDataContentType(), nil
	}

	// Create request body
	body, contentType, err := createBody(fileReader)
	if err != nil {
		return "", err
	}

	// Create request to GigaChat Files API
	// Endpoint: POST https://gigachat.devices.sberbank.ru/api/v1/files
	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/files", body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers according to GigaChat API documentation
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+s.accessToken)

	// Make request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusRequestEntityTooLarge {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("file too large (413): file exceeds maximum size limit: %s", string(bodyBytes))
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// Read response body before closing
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Token might have expired, try to refresh it
		accessToken, err := getAccessToken(ctx, s.config, s.httpClient, s.logger)
		if err != nil {
			return "", fmt.Errorf("upload failed with 401, token refresh also failed: %w (original error: %s)", err, string(bodyBytes))
		}
		s.accessToken = accessToken

		// Retry the request with new token - need to recreate request body
		// Note: fileReader might be consumed, so we need to handle this differently
		// For now, we'll return an error suggesting to retry the whole operation
		return "", fmt.Errorf("token expired, please retry the operation (original error: %s)", string(bodyBytes))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse upload response according to GigaChat API documentation
	// Response format: {"id": "file_id", ...}
	var uploadResp struct {
		ID string `json:"id"`
		// Other fields may include: name, size, created_at, etc.
	}

	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	s.logger.Info("File uploaded to GigaChat", zap.String("file_id", uploadResp.ID))

	return uploadResp.ID, nil
}

// ExtractTextFromImage uses GigaChat Vision API to extract text from an image or PDF
func (s *LLMService) ExtractTextFromImage(ctx context.Context, imagePath string) (string, error) {
	// Open file (image or PDF)
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Upload file to GigaChat
	fileID, err := s.UploadFile(ctx, file, filepath.Base(imagePath))
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Determine file type and use appropriate prompt
	ext := strings.ToLower(filepath.Ext(imagePath))
	var prompt string
	if ext == ".pdf" {
		prompt = `Извлеки весь текст из этого PDF документа. 

ТРЕБОВАНИЯ:
1. Верни ТОЛЬКО текст, который содержится в документе
2. НЕ пиши комментарии, объяснения или сообщения об ошибках
3. Если видишь финансовую информацию (суммы, даты, транзакции, счета), извлеки её полностью
4. Сохрани структуру: заголовки, списки, таблицы
5. Таблицы представь в текстовом формате (строки и столбцы)
6. Если документ пустой или поврежден, верни пустую строку

Начни извлечение текста:`
	} else {
		prompt = `Извлеки весь текст с этого финансового документа (чек, выписка, скриншот). 
Верни только текст, который виден на изображении, без дополнительных комментариев.
Если текст не читается, верни пустую строку.`
	}

	// Use direct HTTP API call for Vision
	return s.ExtractTextViaVisionAPI(ctx, fileID, prompt)
}

// ExtractTextViaVisionAPI uses GigaChat Vision API via HTTP
// Documentation: https://developers.sber.ru/docs/ru/gigachat/api/main
// Endpoint: POST /chat/completions
// Uses file attachments for vision processing
// Uses GigaChat model
func (s *LLMService) ExtractTextViaVisionAPI(ctx context.Context, fileID, prompt string) (string, error) {
	// Use GigaChat model
	modelName := "GigaChat"

	s.logger.Info("Using GigaChat for Vision API", zap.String("file_id", fileID))

	// Create chat completion request with vision
	// According to GigaChat API docs, attachments format: [["file_id"]]
	// Build request body with proper structure
	requestBody := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]interface{}{
			{
				"role":        "user",
				"content":     prompt,
				"attachments": [][]string{{fileID}}, // Array of arrays: [["file_id"]]
			},
		},
		"temperature":        0.3,
		"top_p":              0.0,
		"stream":             false,
		"max_tokens":         0,
		"repetition_penalty": 1.0,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log request for debugging
	s.logger.Debug("Vision API request",
		zap.String("model", modelName),
		zap.String("file_id", fileID),
		zap.String("request_body", string(jsonData)),
	)

	// Create request to GigaChat Chat Completions API
	// Endpoint: POST https://gigachat.devices.sberbank.ru/api/v1/chat/completions
	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers according to GigaChat API documentation
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vision API failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var visionResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&visionResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(visionResp.Choices) == 0 {
		return "", fmt.Errorf("no response from Vision API")
	}

	text := strings.TrimSpace(visionResp.Choices[0].Message.Content)

	// Check if LLM returned an error message instead of extracted text
	textLower := strings.ToLower(text)
	errorPhrases := []string{
		"не могу помочь",
		"не могу обработать",
		"предоставьте содержимое",
		"предоставь содержимое",
		"не могу извлечь",
		"cannot help",
		"cannot process",
		"please provide",
		"не могу помочь с данным запросом",
	}

	for _, phrase := range errorPhrases {
		if strings.Contains(textLower, phrase) {
			s.logger.Warn("LLM returned error message instead of extracted text",
				zap.String("model", modelName),
				zap.String("message", text),
			)
			return "", fmt.Errorf("model returned error message: %s", text)
		}
	}

	s.logger.Info("Text extracted via GigaChat Vision",
		zap.String("model", modelName),
		zap.Int("text_length", len(text)),
	)
	return text, nil
}

func (s *LLMService) Close() error {
	if s.client != nil {
		s.client.Close()
	}
	return nil
}
