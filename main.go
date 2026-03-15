package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

type NewsArticle struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Category   string `json:"category"`
	Date       string `json:"date"`
	IsHot      bool   `json:"is_hot"`
	AuthorName string `json:"author"`
}

type Stats struct {
	TotalNews  int
	LatestDate string
	TechNews   int
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "news_portal.db")
	if err != nil {
		log.Fatal(err)
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS authors (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT, content TEXT, category TEXT, date TEXT, is_hot BOOLEAN, author_id INTEGER,
		FOREIGN KEY(author_id) REFERENCES authors(id)
	)`)
	db.Exec("INSERT OR IGNORE INTO authors (id, name) VALUES (1, 'Редакція NewsPro')")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	categoryFilter := r.URL.Query().Get("category")
	var filtered []NewsArticle

	query := "SELECT a.id, a.title, a.content, a.category, a.date, a.is_hot, auth.name FROM articles a JOIN authors auth ON a.author_id = auth.id"
	var rows *sql.Rows
	var err error

	if categoryFilter != "" {
		rows, err = db.Query(query+" WHERE a.category = ? ORDER BY a.id DESC", categoryFilter)
	} else {
		rows, err = db.Query(query + " ORDER BY a.id DESC")
	}

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a NewsArticle
			rows.Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Date, &a.IsHot, &a.AuthorName)
			filtered = append(filtered, a)
		}
	}

	// Рахуємо статистику
	var stats Stats
	db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&stats.TotalNews)
	db.QueryRow("SELECT COALESCE(MAX(date), 'немає') FROM articles").Scan(&stats.LatestDate)
	db.QueryRow("SELECT COUNT(*) FROM articles WHERE category = 'Технології'").Scan(&stats.TechNews)

	data := struct {
		Articles []NewsArticle
		Stats    Stats
		Year     int
	}{Articles: filtered, Stats: stats, Year: time.Now().Year()}

	tmpl, _ := template.ParseFiles("index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func addNewsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		db.Exec("INSERT INTO articles (title, content, category, date, is_hot, author_id) VALUES (?, ?, ?, ?, ?, 1)",
			r.FormValue("title"), r.FormValue("content"), r.FormValue("category"), r.FormValue("date"), r.FormValue("is_hot") == "on")
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

func viewHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	var a NewsArticle
	db.QueryRow("SELECT a.id, a.title, a.content, a.category, a.date, a.is_hot, auth.name FROM articles a JOIN authors auth ON a.author_id = auth.id WHERE a.id = ?", id).
		Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Date, &a.IsHot, &a.AuthorName)
	tmpl, _ := template.ParseFiles("view.html")
	tmpl.Execute(w, a)
}

func main() {
	initDB()
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/add", addNewsHandler)
	http.HandleFunc("/news", viewHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/api/news", func(w http.ResponseWriter, r *http.Request) {
		rows, _ := db.Query("SELECT id, title, content, category, date, is_hot FROM articles")
		var arts []NewsArticle
		for rows.Next() {
			var a NewsArticle
			rows.Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Date, &a.IsHot)
			arts = append(arts, a)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(arts)
	})
	log.Println("Сервер запущено на http://localhost:9000")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
