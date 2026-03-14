package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

// NewsArticle описує новину (з тегами для JSON)
type NewsArticle struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Category string `json:"category"`
	Date     string `json:"date"`
	IsHot    bool   `json:"is_hot"`
}

// Глобальний список новин (імітація бази даних для звіту)
var articles = []NewsArticle{
	{1, "Go 1.24: Майбутнє системного програмування", "Розробники анонсували нові інструменти для оптимізації пам'яті та покращену роботу з хмарою.", "Технології", "2026-02-09", true},
	{2, "Відкриття інноваційного парку в Києві", "Новий простір для стартапів відкриває двері вже наступного місяця. Очікується понад 100 резидентів.", "Події", "2026-02-08", false},
	{3, "Українська збірна виборола золото", "Наші атлети продемонстрували неймовірну волю до перемоги та здобули рекордну кількість медалей.", "Спорт", "2026-02-07", true},
	{4, "Штучний інтелект у медіа", "Як сучасні алгоритми допомагають журналістам швидше обробляти великі масиви даних та створювати контент.", "Технології", "2026-02-06", false},
	{5, "Благодійний марафон", "Тисячі учасників зареєструвалися на забіг, щоб зібрати кошти на підтримку освітніх проєктів.", "Спорт", "2026-02-05", false},
}

var lastID = 5 // Лічильник для нових ID

// --- КЛАСИЧНІ ОБРОБНИКИ (HTML) ---

// Головна сторінка з фільтрацією
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		if r.URL.Path == "/favicon.ico" {
			return
		}
	}

	categoryFilter := r.URL.Query().Get("category")
	var filtered []NewsArticle

	if categoryFilter != "" {
		for _, a := range articles {
			if a.Category == categoryFilter {
				filtered = append(filtered, a)
			}
		}
	} else {
		filtered = articles
	}

	data := struct {
		Articles []NewsArticle
		Year     int
	}{Articles: filtered, Year: time.Now().Year()}

	tmpl, _ := template.ParseFiles("index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// Перегляд окремої новини
func viewHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	var article NewsArticle
	found := false
	for _, a := range articles {
		if a.ID == id {
			article = a
			found = true
			break
		}
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	tmpl, _ := template.ParseFiles("view.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, article)
}

// Видалення новини (з сайту)
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	for i, a := range articles {
		if a.ID == id {
			articles = append(articles[:i], articles[i+1:]...)
			break
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Додавання через форму
func addNewsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, _ := template.ParseFiles("add.html")
		tmpl.Execute(w, nil)
		return
	}
	if r.Method == "POST" {
		lastID++
		newArt := NewsArticle{
			ID:       lastID,
			Title:    r.FormValue("title"),
			Content:  r.FormValue("content"),
			Category: r.FormValue("category"),
			Date:     r.FormValue("date"),
			IsHot:    r.FormValue("is_hot") == "on",
		}
		articles = append([]NewsArticle{newArt}, articles...)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// --- REST API ОБРОБНИКИ (JSON) ---

func getNewsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles)
}

func main() {
	// Маршрути інтерфейсу
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/news", viewHandler)
	http.HandleFunc("/add", addNewsHandler)
	http.HandleFunc("/delete", deleteHandler)

	// Маршрути API (ПР №4)
	http.HandleFunc("/api/news", getNewsAPI)

	fmt.Println("Сервер запущено: http://localhost:9000")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
