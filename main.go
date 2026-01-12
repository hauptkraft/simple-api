package main

// type PageInfo struct {
// 	HasStructuredData bool `json:"hasStructuredData"`
// 	PriceElements     int  `json:"priceElements"`
// 	ProductElements   int  `json:"productElements"`
// 	TotalElements     int  `json:"totalElements"`
// }

// type Product struct {
// 	Discount    *float64  `json:"discount"`
// 	ElementText string    `json:"elementText"`
// 	Image       string    `json:"image"`
// 	Name        string    `json:"name"`
// 	OldPrice    *float64  `json:"oldPrice"`
// 	PageTitle   string    `json:"pageTitle"`
// 	PageURL     string    `json:"pageUrl"`
// 	Price       float64   `json:"price"`
// 	Source      string    `json:"source"`
// 	Timestamp   time.Time `json:"timestamp"`
// 	Unit        string    `json:"unit"`
// 	URL         string    `json:"url"`
// 	Weight      *float64  `json:"weight"`
// }

// type Stats struct {
// 	AvgPrice      float64 `json:"avgPrice"`
// 	MaxPrice      float64 `json:"maxPrice"`
// 	MinPrice      float64 `json:"minPrice"`
// 	TotalProducts int     `json:"totalProducts"`
// 	WithDiscount  int     `json:"withDiscount"`
// 	WithWeight    int     `json:"withWeight"`
// }

// type Response struct {
// 	PageInfo  PageInfo  `json:"pageInfo"`
// 	PageTitle string    `json:"pageTitle"`
// 	Products  []Product `json:"products"`
// 	Stats     Stats     `json:"stats"`
// 	Success   bool      `json:"success"`
// 	Timestamp time.Time `json:"timestamp"`
// 	URL       string    `json:"url"`
// 	UserAgent string    `json:"userAgent"`
// }

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Структуры данных
type PageInfo struct {
	HasStructuredData bool `json:"hasStructuredData"`
	PriceElements     int  `json:"priceElements"`
	ProductElements   int  `json:"productElements"`
	TotalElements     int  `json:"totalElements"`
}

func (p PageInfo) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *PageInfo) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal PageInfo value: %v", value)
	}
	return json.Unmarshal(bytes, p)
}

type Product struct {
	ID          string    `json:"id,omitempty" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Discount    *float64  `json:"discount,omitempty" gorm:"type:decimal(10,2)"`
	ElementText string    `json:"elementText" gorm:"type:text"`
	Image       string    `json:"image" gorm:"type:text"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null;index"`
	OldPrice    *float64  `json:"oldPrice,omitempty" gorm:"type:decimal(10,2)"`
	PageTitle   string    `json:"pageTitle" gorm:"type:text"`
	PageURL     string    `json:"pageUrl" gorm:"type:text;index"`
	Price       float64   `json:"price" gorm:"type:decimal(10,2);not null;index"`
	Source      string    `json:"source" gorm:"type:varchar(100);not null;index"`
	Timestamp   time.Time `json:"timestamp" gorm:"type:timestamptz;index"`
	Unit        string    `json:"unit" gorm:"type:varchar(50)"`
	URL         string    `json:"url" gorm:"type:text;uniqueIndex"`
	Weight      *string   `json:"weight,omitempty" gorm:"type:varchar(50)"`
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
	PageDataID  *string   `json:"-" gorm:"type:uuid;index"`
}

type Stats struct {
	AvgPrice      float64 `json:"avgPrice"`
	MaxPrice      float64 `json:"maxPrice"`
	MinPrice      float64 `json:"minPrice"`
	TotalProducts int     `json:"totalProducts"`
	WithDiscount  int     `json:"withDiscount"`
	WithWeight    int     `json:"withWeight"`
}

func (s Stats) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *Stats) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal Stats value: %v", value)
	}
	return json.Unmarshal(bytes, s)
}

type PageData struct {
	ID        string    `json:"id,omitempty" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PageInfo  PageInfo  `json:"pageInfo" gorm:"type:jsonb"`
	PageTitle string    `json:"pageTitle" gorm:"type:text"`
	Products  []Product `json:"products" gorm:"foreignKey:PageDataID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Stats     Stats     `json:"stats" gorm:"type:jsonb"`
	Success   bool      `json:"success" gorm:"default:true"`
	Timestamp string    `json:"timestamp" gorm:"type:text"`
	URL       string    `json:"url" gorm:"type:text;uniqueIndex"`
	UserAgent string    `json:"userAgent" gorm:"type:text"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

// Request/Response структуры
type SavePageDataRequest struct {
	PageData PageData `json:"pageData"`
}

type SavePageDataResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ID        string `json:"id,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

type GetPageDataResponse struct {
	Success bool      `json:"success"`
	Data    *PageData `json:"data,omitempty"`
	Error   string    `json:"error,omitempty"`
}

type GetPageDataHistoryResponse struct {
	Success bool        `json:"success"`
	Data    []PageData  `json:"data"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"perPage"`
	Filters interface{} `json:"filters,omitempty"`
}

type GetStatisticsResponse struct {
	Success bool        `json:"success"`
	Stats   interface{} `json:"stats"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// Конфигурация
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
	APIPort  string `json:"apiPort"`
}

var (
	cfg = Config{
		Host:     "jadatabmarel.beget.app",
		Port:     5432,
		User:     "cloud_user",
		Password: "wKtK%vSXgci7",
		DBName:   "api_db",
		SSLMode:  "disable",
		APIPort:  "8080",
	}

	db  *gorm.DB
	ctx = context.Background()
)

// Application структура с HTTP обработчиками
type Application struct {
	db *gorm.DB
}

func NewApplication(db *gorm.DB) *Application {
	return &Application{db: db}
}

// HTTP Handlers
func (app *Application) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"database":  "connected",
		"version":   "1.0.0",
	}

	app.respondWithJSON(w, http.StatusOK, response)
}

// Основной обработчик для сохранения данных
func (app *Application) savePageDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Проверяем Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		app.respondWithError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	var req SavePageDataRequest

	// Декодируем тело запроса
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.respondWithError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Валидация
	if req.PageData.URL == "" {
		app.respondWithError(w, http.StatusBadRequest, "URL is required")
		return
	}

	if len(req.PageData.Products) == 0 {
		app.respondWithError(w, http.StatusBadRequest, "At least one product is required")
		return
	}

	// Устанавливаем timestamp если не указан
	if req.PageData.Timestamp == "" {
		req.PageData.Timestamp = time.Now().Format(time.RFC3339)
	}

	// Устанавливаем UserAgent из заголовков если не указан
	if req.PageData.UserAgent == "" {
		req.PageData.UserAgent = r.UserAgent()
	}

	// Устанавливаем PageTitle для каждого продукта если не указан
	for i := range req.PageData.Products {
		if req.PageData.Products[i].PageTitle == "" {
			req.PageData.Products[i].PageTitle = req.PageData.PageTitle
		}
		if req.PageData.Products[i].PageURL == "" {
			req.PageData.Products[i].PageURL = req.PageData.URL
		}
	}

	// Сохраняем данные
	err := app.savePageData(&req.PageData)
	if err != nil {
		log.Printf("Error saving page data: %v", err)

		// Проверяем конкретные ошибки
		if strings.Contains(err.Error(), "duplicate key") {
			app.respondWithError(w, http.StatusConflict, "Page data already exists")
		} else {
			app.respondWithError(w, http.StatusInternalServerError, "Failed to save page data")
		}
		return
	}

	// Формируем ответ
	response := SavePageDataResponse{
		Success:   true,
		Message:   "Page data saved successfully",
		ID:        req.PageData.ID,
		CreatedAt: req.PageData.CreatedAt.Format(time.RFC3339),
	}

	app.respondWithJSON(w, http.StatusCreated, response)
}

func (app *Application) getPageDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Извлекаем параметры из URL
	query := r.URL.Query()
	url := query.Get("url")
	id := query.Get("id")

	if url == "" && id == "" {
		app.respondWithError(w, http.StatusBadRequest, "URL or ID parameter is required")
		return
	}

	var pageData *PageData
	var err error

	if id != "" {
		pageData, err = app.getPageDataByID(id)
	} else {
		// Получаем последний по дате
		limit := 1
		if query.Get("all") == "true" {
			limit = 100
		}
		pageData, err = app.getLatestPageDataByURL(url, limit)
	}

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			app.respondWithError(w, http.StatusNotFound, "Page data not found")
		} else {
			log.Printf("Error getting page data: %v", err)
			app.respondWithError(w, http.StatusInternalServerError, "Failed to get page data")
		}
		return
	}

	response := GetPageDataResponse{
		Success: true,
		Data:    pageData,
	}

	app.respondWithJSON(w, http.StatusOK, response)
}

func (app *Application) getPageDataHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query()
	url := query.Get("url")
	if url == "" {
		app.respondWithError(w, http.StatusBadRequest, "URL parameter is required")
		return
	}

	// Параметры пагинации
	page := app.getQueryInt(query, "page", 1)
	perPage := app.getQueryInt(query, "per_page", 10)

	// Фильтры
	source := query.Get("source")
	dateFrom := query.Get("date_from")
	dateTo := query.Get("date_to")
	successOnly := query.Get("success") == "true"

	// Получаем историю
	pageDataList, total, err := app.getPageDataHistory(url, page, perPage, source, dateFrom, dateTo, successOnly)
	if err != nil {
		log.Printf("Error getting page data history: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "Failed to get history")
		return
	}

	// Формируем фильтры для ответа
	filters := map[string]interface{}{
		"url":      url,
		"source":   source,
		"dateFrom": dateFrom,
		"dateTo":   dateTo,
		"success":  successOnly,
	}

	response := GetPageDataHistoryResponse{
		Success: true,
		Data:    pageDataList,
		Total:   total,
		Page:    page,
		PerPage: perPage,
		Filters: filters,
	}

	app.respondWithJSON(w, http.StatusOK, response)
}

func (app *Application) getStatisticsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query()
	url := query.Get("url")
	period := query.Get("period") // day, week, month, year

	var stats interface{}
	var err error

	if url != "" {
		// Статистика по конкретному URL
		stats, err = app.getURLStatistics(url, period)
	} else {
		// Глобальная статистика
		stats, err = app.getGlobalStatistics()
	}

	if err != nil {
		log.Printf("Error getting statistics: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "Failed to get statistics")
		return
	}

	response := GetStatisticsResponse{
		Success: true,
		Stats:   stats,
	}

	app.respondWithJSON(w, http.StatusOK, response)
}

func (app *Application) searchProductsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query()
	searchQuery := query.Get("q")
	if searchQuery == "" {
		app.respondWithError(w, http.StatusBadRequest, "Search query is required")
		return
	}

	page := app.getQueryInt(query, "page", 1)
	perPage := app.getQueryInt(query, "per_page", 20)

	// Фильтры
	minPrice := app.getQueryFloat(query, "min_price", 0)
	maxPrice := app.getQueryFloat(query, "max_price", 0)
	source := query.Get("source")
	withDiscount := query.Get("discount") == "true"

	products, total, err := app.searchProducts(searchQuery, page, perPage, minPrice, maxPrice, source, withDiscount)
	if err != nil {
		log.Printf("Error searching products: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "Failed to search products")
		return
	}

	response := map[string]interface{}{
		"success": true,
		"query":   searchQuery,
		"results": products,
		"total":   total,
		"page":    page,
		"perPage": perPage,
		"filters": map[string]interface{}{
			"minPrice":     minPrice,
			"maxPrice":     maxPrice,
			"source":       source,
			"withDiscount": withDiscount,
		},
	}

	app.respondWithJSON(w, http.StatusOK, response)
}

func (app *Application) deletePageDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		app.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Извлекаем ID из query параметров
	query := r.URL.Query()
	id := query.Get("id")
	if id == "" {
		app.respondWithError(w, http.StatusBadRequest, "ID parameter is required")
		return
	}

	err := app.deletePageData(id)
	if err != nil {
		log.Printf("Error deleting page data: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "Failed to delete page data")
		return
	}

	app.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Page data deleted successfully",
		"id":      id,
	})
}

// Обработчик для получения продукта по ID
func (app *Application) getProductHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query()
	id := query.Get("id")
	url := query.Get("url")

	if id == "" && url == "" {
		app.respondWithError(w, http.StatusBadRequest, "ID or URL parameter is required")
		return
	}

	var product Product
	var err error

	if id != "" {
		err = app.db.First(&product, "id = ?", id).Error
	} else {
		err = app.db.First(&product, "url = ?", url).Error
	}

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			app.respondWithError(w, http.StatusNotFound, "Product not found")
		} else {
			log.Printf("Error getting product: %v", err)
			app.respondWithError(w, http.StatusInternalServerError, "Failed to get product")
		}
		return
	}

	app.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"product": product,
	})
}

// Вспомогательные методы HTTP
func (app *Application) respondWithError(w http.ResponseWriter, code int, message string) {
	app.respondWithJSON(w, code, ErrorResponse{
		Success: false,
		Error:   message,
	})
}

func (app *Application) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success": false, "error": "Failed to marshal response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (app *Application) getQueryInt(query map[string][]string, key string, defaultValue int) int {
	values, ok := query[key]
	if !ok || len(values) == 0 {
		return defaultValue
	}

	intValue, err := strconv.Atoi(values[0])
	if err != nil {
		return defaultValue
	}

	if intValue < 1 {
		return defaultValue
	}

	return intValue
}

func (app *Application) getQueryFloat(query map[string][]string, key string, defaultValue float64) float64 {
	values, ok := query[key]
	if !ok || len(values) == 0 {
		return defaultValue
	}

	floatValue, err := strconv.ParseFloat(values[0], 64)
	if err != nil {
		return defaultValue
	}

	return floatValue
}

// CORS middleware
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Устанавливаем заголовки CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Обработка preflight запросов
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// Методы работы с данными
func (app *Application) savePageData(pageData *PageData) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
		// Сначала сохраняем PageData
		if err := tx.Create(pageData).Error; err != nil {
			return fmt.Errorf("ошибка сохранения PageData: %w", err)
		}

		// Затем сохраняем продукты
		for i := range pageData.Products {
			pageData.Products[i].PageDataID = &pageData.ID

			// Проверяем существование продукта
			var existingProduct Product
			err := tx.Where("url = ?", pageData.Products[i].URL).First(&existingProduct).Error

			if err == nil {
				// Обновляем существующий
				pageData.Products[i].ID = existingProduct.ID
				if err := tx.Save(&pageData.Products[i]).Error; err != nil {
					return fmt.Errorf("ошибка обновления продукта: %w", err)
				}
			} else if err == gorm.ErrRecordNotFound {
				// Создаем новый
				if err := tx.Create(&pageData.Products[i]).Error; err != nil {
					return fmt.Errorf("ошибка создания продукта: %w", err)
				}
			} else {
				return fmt.Errorf("ошибка проверки продукта: %w", err)
			}
		}

		return nil
	})
}

func (app *Application) getPageDataByID(id string) (*PageData, error) {
	var pageData PageData
	err := app.db.Where("id = ?", id).
		Preload("Products").
		First(&pageData).Error

	if err != nil {
		return nil, err
	}

	return &pageData, nil
}

func (app *Application) getLatestPageDataByURL(url string, limit int) (*PageData, error) {
	var pageData PageData

	query := app.db.Where("url = ?", url).
		Preload("Products").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.First(&pageData).Error
	if err != nil {
		return nil, err
	}

	return &pageData, nil
}

func (app *Application) getPageDataHistory(url string, page, perPage int, source, dateFrom, dateTo string, successOnly bool) ([]PageData, int64, error) {
	var pageDataList []PageData
	var total int64

	query := app.db.Model(&PageData{}).Where("url = ?", url)

	// Применяем фильтры
	if source != "" {
		query = query.Joins("JOIN products ON products.page_data_id = page_data.id").
			Where("products.source = ?", source)
	}

	if dateFrom != "" {
		if dateFromTime, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			query = query.Where("page_data.created_at >= ?", dateFromTime)
		}
	}

	if dateTo != "" {
		if dateToTime, err := time.Parse(time.RFC3339, dateTo); err == nil {
			query = query.Where("page_data.created_at <= ?", dateToTime)
		}
	}

	if successOnly {
		query = query.Where("success = ?", true)
	}

	// Считаем общее количество
	query.Count(&total)

	// Получаем данные с пагинацией
	offset := (page - 1) * perPage
	err := query.
		Preload("Products").
		Order("created_at DESC").
		Offset(offset).
		Limit(perPage).
		Find(&pageDataList).Error

	return pageDataList, total, err
}

// Статистика по URL
type URLStatistics struct {
	URL                string    `json:"url"`
	TotalParsings      int64     `json:"totalParsings"`
	SuccessfulParsings int64     `json:"successfulParsings"`
	TotalProducts      int64     `json:"totalProducts"`
	UniqueProducts     int64     `json:"uniqueProducts"`
	AvgPrice           float64   `json:"avgPrice"`
	MaxPrice           float64   `json:"maxPrice"`
	MinPrice           float64   `json:"minPrice"`
	FirstParsed        time.Time `json:"firstParsed"`
	LastParsed         time.Time `json:"lastParsed"`
	LastStats          Stats     `json:"lastStats"`
}

func (app *Application) getURLStatistics(url string, period string) (*URLStatistics, error) {
	var stats URLStatistics
	stats.URL = url

	// Базовый запрос для данного URL
	baseQuery := app.db.Model(&PageData{}).Where("url = ?", url)

	// Применяем период если указан
	if period != "" {
		var timeAgo time.Time
		now := time.Now()

		switch period {
		case "day":
			timeAgo = now.AddDate(0, 0, -1)
		case "week":
			timeAgo = now.AddDate(0, 0, -7)
		case "month":
			timeAgo = now.AddDate(0, -1, 0)
		case "year":
			timeAgo = now.AddDate(-1, 0, 0)
		default:
			timeAgo = now.AddDate(0, 0, -30)
		}

		baseQuery = baseQuery.Where("created_at >= ?", timeAgo)
	}

	// Общее количество парсингов
	baseQuery.Count(&stats.TotalParsings)

	// Успешные парсинги
	app.db.Model(&PageData{}).
		Where("url = ? AND success = ?", url, true).
		Count(&stats.SuccessfulParsings)

	// Первый и последний парсинг
	var first, last PageData
	app.db.Where("url = ?", url).
		Order("created_at ASC").
		First(&first)
	app.db.Where("url = ?", url).
		Order("created_at DESC").
		First(&last)

	if first.ID != "" {
		stats.FirstParsed = first.CreatedAt
	}
	if last.ID != "" {
		stats.LastParsed = last.CreatedAt
		stats.LastStats = last.Stats
	}

	// Статистика по продуктам
	productQuery := app.db.Model(&Product{}).
		Joins("JOIN page_data ON page_data.id = products.page_data_id").
		Where("page_data.url = ?", url)

	if period != "" {
		productQuery = productQuery.Where("page_data.created_at >= ?", time.Now().AddDate(0, 0, -30))
	}

	productQuery.Count(&stats.TotalProducts)

	// Уникальные продукты
	app.db.Model(&Product{}).
		Joins("JOIN page_data ON page_data.id = products.page_data_id").
		Where("page_data.url = ?", url).
		Distinct("products.url").
		Count(&stats.UniqueProducts)

	// Статистика цен
	var priceStats struct {
		AvgPrice float64
		MaxPrice float64
		MinPrice float64
	}

	app.db.Model(&Product{}).
		Select("AVG(price) as avg_price, MAX(price) as max_price, MIN(price) as min_price").
		Joins("JOIN page_data ON page_data.id = products.page_data_id").
		Where("page_data.url = ?", url).
		Scan(&priceStats)

	stats.AvgPrice = priceStats.AvgPrice
	stats.MaxPrice = priceStats.MaxPrice
	stats.MinPrice = priceStats.MinPrice

	return &stats, nil
}

func (app *Application) getGlobalStatistics() (interface{}, error) {
	var stats struct {
		TotalParsings      int64   `json:"totalParsings"`
		SuccessfulParsings int64   `json:"successfulParsings"`
		TotalProducts      int64   `json:"totalProducts"`
		UniqueURLs         int64   `json:"uniqueUrls"`
		AvgPrice           float64 `json:"avgPrice"`
		MaxPrice           float64 `json:"maxPrice"`
		MinPrice           float64 `json:"minPrice"`
		Last24Hours        int64   `json:"last24Hours"`
		Last7Days          int64   `json:"last7Days"`
		Last30Days         int64   `json:"last30Days"`
	}

	// Общая статистика
	app.db.Model(&PageData{}).Count(&stats.TotalParsings)
	app.db.Model(&PageData{}).Where("success = ?", true).Count(&stats.SuccessfulParsings)
	app.db.Model(&Product{}).Count(&stats.TotalProducts)
	app.db.Model(&PageData{}).Distinct("url").Count(&stats.UniqueURLs)

	// Статистика за различные периоды
	now := time.Now()
	last24Hours := now.Add(-24 * time.Hour)
	last7Days := now.AddDate(0, 0, -7)
	last30Days := now.AddDate(0, 0, -30)

	app.db.Model(&PageData{}).Where("created_at >= ?", last24Hours).Count(&stats.Last24Hours)
	app.db.Model(&PageData{}).Where("created_at >= ?", last7Days).Count(&stats.Last7Days)
	app.db.Model(&PageData{}).Where("created_at >= ?", last30Days).Count(&stats.Last30Days)

	// Статистика цен
	var priceStats struct {
		AvgPrice float64
		MaxPrice float64
		MinPrice float64
	}

	app.db.Model(&Product{}).
		Select("AVG(price) as avg_price, MAX(price) as max_price, MIN(price) as min_price").
		Scan(&priceStats)

	stats.AvgPrice = priceStats.AvgPrice
	stats.MaxPrice = priceStats.MaxPrice
	stats.MinPrice = priceStats.MinPrice

	// Топ источников
	var topSources []map[string]interface{}
	app.db.Model(&Product{}).
		Select("source, COUNT(*) as count, AVG(price) as avg_price").
		Group("source").
		Order("count DESC").
		Limit(10).
		Scan(&topSources)

	// Последние парсинги
	var recentParsings []PageData
	app.db.Order("created_at DESC").Limit(5).Find(&recentParsings)

	return map[string]interface{}{
		"overview":       stats,
		"topSources":     topSources,
		"recentParsings": recentParsings,
	}, nil
}

func (app *Application) searchProducts(query string, page, perPage int, minPrice, maxPrice float64, source string, withDiscount bool) ([]Product, int64, error) {
	var products []Product
	var total int64

	dbQuery := app.db.Model(&Product{})

	// Поисковый запрос
	if query != "" {
		searchQuery := "%" + strings.ToLower(query) + "%"
		dbQuery = dbQuery.Where("LOWER(name) LIKE ? OR LOWER(element_text) LIKE ?", searchQuery, searchQuery)
	}

	// Фильтры по цене
	if minPrice > 0 {
		dbQuery = dbQuery.Where("price >= ?", minPrice)
	}
	if maxPrice > 0 {
		dbQuery = dbQuery.Where("price <= ?", maxPrice)
	}

	// Фильтр по источнику
	if source != "" {
		dbQuery = dbQuery.Where("source = ?", source)
	}

	// Фильтр по наличию скидки
	if withDiscount {
		dbQuery = dbQuery.Where("discount IS NOT NULL AND discount > 0")
	}

	// Подсчет общего количества
	dbQuery.Count(&total)

	// Получение данных с пагинацией
	offset := (page - 1) * perPage
	err := dbQuery.
		Order("created_at DESC").
		Offset(offset).
		Limit(perPage).
		Find(&products).Error

	return products, total, err
}

func (app *Application) deletePageData(id string) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
		// Сначала удаляем связанные продукты
		if err := tx.Where("page_data_id = ?", id).Delete(&Product{}).Error; err != nil {
			return err
		}

		// Затем удаляем PageData
		if err := tx.Delete(&PageData{}, "id = ?", id).Error; err != nil {
			return err
		}

		return nil
	})
}

// Инициализация базы данных
func initDatabase() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormLogger,
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	// Настройка пула соединений
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	// Выполнение миграций
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("ошибка миграции: %w", err)
	}

	log.Println("База данных успешно инициализирована")
	return db, nil
}

func runMigrations(db *gorm.DB) error {
	db.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";")

	err := db.AutoMigrate(&PageData{}, &Product{})
	if err != nil {
		return fmt.Errorf("ошибка AutoMigrate: %w", err)
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_page_data_url_created ON page_data(url, created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_page_data_success ON page_data(success) WHERE success = true;",
		"CREATE INDEX IF NOT EXISTS idx_products_price ON products(price);",
		"CREATE INDEX IF NOT EXISTS idx_products_name_lower ON products(LOWER(name));",
		"CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_products_page_data_id ON products(page_data_id);",
		"CREATE INDEX IF NOT EXISTS idx_products_discount ON products((discount IS NOT NULL)) WHERE discount IS NOT NULL;",
	}

	for _, idx := range indexes {
		db.Exec(idx)
	}

	return nil
}

// Маршрутизатор на основе стандартной библиотеки
func setupRouter(app *Application) *http.ServeMux {
	mux := http.NewServeMux()

	// Регистрируем обработчики
	mux.HandleFunc("/health", corsMiddleware(app.healthHandler))

	// API endpoints
	mux.HandleFunc("/api/v1/page-data", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			corsMiddleware(app.savePageDataHandler)(w, r)
		case http.MethodGet:
			corsMiddleware(app.getPageDataHandler)(w, r)
		default:
			app.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})

	mux.HandleFunc("/api/v1/page-data/history", corsMiddleware(app.getPageDataHistoryHandler))
	mux.HandleFunc("/api/v1/statistics", corsMiddleware(app.getStatisticsHandler))
	mux.HandleFunc("/api/v1/search/products", corsMiddleware(app.searchProductsHandler))
	mux.HandleFunc("/api/v1/product", corsMiddleware(app.getProductHandler))

	// DELETE endpoint для удаления данных
	mux.HandleFunc("/api/v1/page-data/delete", corsMiddleware(app.deletePageDataHandler))

	// Главная страница
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			app.respondWithError(w, http.StatusNotFound, "Not found")
			return
		}

		info := map[string]interface{}{
			"name":        "Scraper API",
			"version":     "1.0.0",
			"description": "API для сохранения и получения данных парсинга",
			"endpoints": []map[string]string{
				{"method": "GET", "path": "/health", "description": "Проверка здоровья сервера"},
				{"method": "POST", "path": "/api/v1/page-data", "description": "Сохранение данных парсинга"},
				{"method": "GET", "path": "/api/v1/page-data", "description": "Получение данных по URL"},
				{"method": "GET", "path": "/api/v1/page-data/history", "description": "История парсингов"},
				{"method": "GET", "path": "/api/v1/statistics", "description": "Статистика"},
				{"method": "GET", "path": "/api/v1/search/products", "description": "Поиск продуктов"},
				{"method": "GET", "path": "/api/v1/product", "description": "Получение продукта"},
				{"method": "DELETE", "path": "/api/v1/page-data/delete", "description": "Удаление данных"},
			},
		}

		app.respondWithJSON(w, http.StatusOK, info)
	})

	return mux
}

// Middleware для логирования запросов
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем wrapper для захвата статуса ответа
		ww := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		log.Printf("%s %s %d %v", r.Method, r.URL.Path, ww.statusCode, time.Since(start))
	})
}

// Wrapper для захвата статуса ответа
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Вспомогательные функции
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Главная функция
func main() {
	// Инициализация базы данных
	var err error
	db, err = initDatabase()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Создаем приложение
	app := NewApplication(db)

	// Настраиваем маршрутизатор
	router := setupRouter(app)

	// Оборачиваем в middleware для логирования
	handler := loggingMiddleware(router)

	// Запускаем сервер
	serverAddr := ":" + cfg.APIPort
	log.Printf("Starting server on %s", serverAddr)
	log.Printf("Database: %s@%s:%d/%s", cfg.User, cfg.Host, cfg.Port, cfg.DBName)
	log.Printf("API endpoints:")
	log.Printf("  GET    /                         - Информация об API")
	log.Printf("  GET    /health                   - Проверка здоровья")
	log.Printf("  POST   /api/v1/page-data         - Сохранение данных парсинга")
	log.Printf("  GET    /api/v1/page-data         - Получение данных по URL")
	log.Printf("  GET    /api/v1/page-data/history - История парсингов")
	log.Printf("  GET    /api/v1/statistics        - Статистика")
	log.Printf("  GET    /api/v1/search/products   - Поиск продуктов")
	log.Printf("  GET    /api/v1/product           - Получение продукта")
	log.Printf("  DELETE /api/v1/page-data/delete  - Удаление данных")

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server failed:", err)
	}
}

// Пример использования с Beget
func connectToBeget() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=require TimeZone=UTC",
		"your_host.postgresql.beget.com", 5432, "your_user", "your_password", "your_db",
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})

	if err != nil {
		return nil, fmt.Errorf("ошибка подключения: %w", err)
	}

	return db, nil
}
