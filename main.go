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
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq" // драйвер PostgreSQL
)

var db *sql.DB

// Структуры для запроса и ответа
type NameRequest struct {
	Name string `json:"name"`
}

type NameResponse struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Database  string `json:"database"`
}

// Инициализация базы данных
func initDB() error {
	// Получаем параметры подключения из переменных окружения
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		return fmt.Errorf("не все параметры БД заданы")
	}

	// SSL обязателен для Beget
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %v", err)
	}

	// Проверяем подключение
	err = db.Ping()
	if err != nil {
		return fmt.Errorf("ошибка ping БД: %v", err)
	}

	// Создаем таблицу если не существует
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS names (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы: %v", err)
	}

	log.Println("База данных успешно подключена")
	return nil
}

// Обработчик POST /name
func nameHandler(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	// Проверяем Content-Type
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		http.Error(w, `{"error": "Content-Type должен быть application/json"}`, http.StatusBadRequest)
		return
	}

	// Декодируем JSON
	var req NameRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Неверный JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	// Валидация
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 255 {
		http.Error(w, `{"error": "Имя не должно быть пустым и должно быть менее 255 символов"}`, http.StatusBadRequest)
		return
	}

	// Вставляем в базу данных
	var id int
	var createdAt time.Time

	err := db.QueryRow(
		"INSERT INTO names (name) VALUES ($1) RETURNING id, created_at",
		req.Name,
	).Scan(&id, &createdAt)

	if err != nil {
		log.Printf("Ошибка вставки в БД: %v", err)
		http.Error(w, `{"error": "Ошибка сохранения в базу данных"}`, http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	resp := NameResponse{
		ID:        id,
		Name:      req.Name,
		CreatedAt: createdAt,
		Status:    "success",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Ошибка кодирования JSON: %v", err)
	}
}

// Обработчик GET /health
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	// Проверяем подключение к БД
	dbStatus := "connected"
	err := db.Ping()
	if err != nil {
		dbStatus = "disconnected"
	}

	resp := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
		Database:  dbStatus,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Ошибка кодирования JSON: %v", err)
	}
}

// Middleware для логирования
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем обертку для ResponseWriter, чтобы перехватить статус
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next(rw, r)

		log.Printf("%s %s %d %v", r.Method, r.URL.Path, rw.statusCode, time.Since(start))
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	// Инициализация БД
	// if err := initDB(); err != nil {
	// 	log.Fatalf("Ошибка инициализации БД: %v", err)
	// }
	// defer db.Close()

	// Настройка маршрутов
	// http.HandleFunc("/api/name", loggingMiddleware(nameHandler))
	// http.HandleFunc("/api/health", loggingMiddleware(healthHandler))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service": "Simple API", "version": "1.0.0"}`)
	})

	// Запуск сервера
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	//serverAddr := ":" + port
	log.Printf("Сервер запущен на порту %s", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:8000", nil))
}
