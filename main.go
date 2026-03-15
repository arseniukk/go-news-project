package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

// --- МОДЕЛІ ДАНИХ ---

type NewsArticle struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Category   string `json:"category"`
	Date       string `json:"date"`
	IsHot      bool   `json:"is_hot"`
	ImageURL   string `json:"image_url"` // Посилання на картинку
	Views      int    `json:"views"`     // Кількість переглядів
	AuthorName string `json:"author"`
}

type Stats struct {
	TotalNews  int
	TotalViews int
	TechNews   int
	SportNews  int
	EventNews  int
	LatestDate string
}

var db *sql.DB

// --- ПІДГОТОВКА СЕРВЕРА ---

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "news_portal.db")
	if err != nil {
		log.Fatal("Помилка підключення до БД:", err)
	}

	// Створення таблиць з новими полями (image_url та views)
	db.Exec(`CREATE TABLE IF NOT EXISTS authors (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT, content TEXT, category TEXT, date TEXT, 
		is_hot BOOLEAN, image_url TEXT, views INTEGER DEFAULT 0, author_id INTEGER,
		FOREIGN KEY(author_id) REFERENCES authors(id)
	)`)
	db.Exec("INSERT OR IGNORE INTO authors (id, name) VALUES (1, 'Адміністратор NewsPro')")
}

// --- MIDDLEWARE (Логування та Авторизація) ---

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func BasicAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "1234" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Авторизуйтесь (admin/1234)", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// --- ОБРОБНИКИ (HANDLERS) ---

// Головна сторінка (Пошук + Фільтрація + Статистика)
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		if r.URL.Path == "/favicon.ico" {
			return
		}
	}

	search := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")

	query := `SELECT a.id, a.title, a.content, a.category, a.date, a.is_hot, a.image_url, a.views, auth.name 
              FROM articles a JOIN authors auth ON a.author_id = auth.id`

	var rows *sql.Rows
	var err error

	if search != "" {
		// Простий LIKE без LOWER (в SQLite для кирилиці краще так)
		// Важливо: введіть слово в пошук точно так, як воно в заголовку (наприклад, з великої літери)
		rows, err = db.Query(query+" WHERE a.title LIKE ? OR a.content LIKE ? ORDER BY a.id DESC", "%"+search+"%", "%"+search+"%")
	} else if category != "" {
		rows, err = db.Query(query+" WHERE a.category = ? ORDER BY a.id DESC", category)
	} else {
		rows, err = db.Query(query + " ORDER BY a.id DESC")
	}

	if err != nil {
		http.Error(w, "Помилка БД", 500)
		return
	}
	defer rows.Close()

	var filtered []NewsArticle
	for rows.Next() {
		var a NewsArticle
		rows.Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Date, &a.IsHot, &a.ImageURL, &a.Views, &a.AuthorName)
		// Якщо картинки немає, ставимо дефолтну
		if a.ImageURL == "" {
			a.ImageURL = "https://images.unsplash.com/photo-1504711434969-e33886168f5c?w=500"
		}
		filtered = append(filtered, a)
	}

	var stats Stats
	// 1. Загальна кількість та сума переглядів
	db.QueryRow("SELECT COUNT(*), COALESCE(SUM(views), 0) FROM articles").Scan(&stats.TotalNews, &stats.TotalViews)

	// 2. Підрахунок по кожній категорії окремо
	db.QueryRow("SELECT COUNT(*) FROM articles WHERE category = 'Технології'").Scan(&stats.TechNews)
	db.QueryRow("SELECT COUNT(*) FROM articles WHERE category = 'Спорт'").Scan(&stats.SportNews)
	db.QueryRow("SELECT COUNT(*) FROM articles WHERE category = 'Події'").Scan(&stats.EventNews)

	// 3. Дата останньої новини (виправляємо, щоб не була порожньою)
	db.QueryRow("SELECT COALESCE(MAX(date), 'немає') FROM articles").Scan(&stats.LatestDate)

	data := struct {
		Articles []NewsArticle
		Stats    Stats
		Year     int
	}{Articles: filtered, Stats: stats, Year: time.Now().Year()}

	tmpl, _ := template.ParseFiles("index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// Перегляд новини (Лічильник переглядів ++)
func viewHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))

	// Збільшуємо перегляди в БД
	db.Exec("UPDATE articles SET views = views + 1 WHERE id = ?", id)

	var a NewsArticle
	err := db.QueryRow(`SELECT a.id, a.title, a.content, a.category, a.date, a.is_hot, a.image_url, a.views, auth.name 
	             FROM articles a JOIN authors auth ON a.author_id = auth.id WHERE a.id = ?`, id).
		Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Date, &a.IsHot, &a.ImageURL, &a.Views, &a.AuthorName)

	if err != nil {
		http.NotFound(w, r)
		return
	}
	if a.ImageURL == "" {
		a.ImageURL = "https://images.unsplash.com/photo-1504711434969-e33886168f5c?w=800"
	}

	tmpl, _ := template.ParseFiles("view.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, a)
}

func addNewsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		db.Exec("INSERT INTO articles (title, content, category, date, is_hot, image_url, author_id) VALUES (?, ?, ?, ?, ?, ?, 1)",
			r.FormValue("title"), r.FormValue("content"), r.FormValue("category"), r.FormValue("date"), r.FormValue("is_hot") == "on", r.FormValue("image_url"))
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	tmpl, _ := template.ParseFiles("add.html")
	tmpl.Execute(w, nil)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	db.Exec("DELETE FROM articles WHERE id = ?", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getNewsAPI(w http.ResponseWriter, r *http.Request) {
	rows, _ := db.Query("SELECT id, title, content, category, date, is_hot, views FROM articles")
	var arts []NewsArticle
	for rows.Next() {
		var a NewsArticle
		rows.Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Date, &a.IsHot, &a.Views)
		arts = append(arts, a)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arts)
}

// --- ЗАПУСК ---

func main() {
	initDB()
	defer db.Close()

	mux := http.NewServeMux()

	// Публічні
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/news", viewHandler)
	mux.HandleFunc("/api/news", getNewsAPI)

	// Захищені (admin/1234)
	mux.HandleFunc("/add", BasicAuthMiddleware(addNewsHandler))
	mux.HandleFunc("/delete", BasicAuthMiddleware(deleteHandler))

	finalHandler := LoggingMiddleware(mux)

	fmt.Println("💎 NewsPro GEM Version запущенна на http://localhost:9000")
	log.Fatal(http.ListenAndServe(":9000", finalHandler))
}
