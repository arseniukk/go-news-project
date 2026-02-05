package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

// NewsArticle описує новину
type NewsArticle struct {
	ID       int
	Title    string
	Content  string
	Category string
	Date     string
	IsHot    bool
}

// PageData для головної сторінки
type PageData struct {
	PageTitle string
	Articles  []NewsArticle
	Year      int
}

// FormResponse для сторінки додавання (передача помилок)
type FormResponse struct {
	Error string
	Data  NewsArticle
}

// Глобальна змінна (база даних у пам'яті)
var articles = []NewsArticle{
	{1, "Перша новина порталу", "Вітаємо на нашому новому двигуні Go!", "Події", "2026-02-05", true},
}

// Головна сторінка
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		// Якщо шлях не знайдено, ми не робимо нічого (це не дасть фавікону заважати)
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

	data := PageData{
		PageTitle: "Новини Головного",
		Articles:  filtered,
		Year:      time.Now().Year(),
	}

	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		log.Printf("Помилка index.html: %v", err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// Сторінка додавання новини
func addNewsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := template.ParseFiles("add.html")
		if err != nil {
			log.Printf("Помилка add.html: %v", err)
			http.Error(w, "Файл add.html не знайдено", 500)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == "POST" {
		// Отримуємо дані з форми
		title := r.FormValue("title")
		category := r.FormValue("category")
		content := r.FormValue("content")
		date := r.FormValue("date")
		isHot := r.FormValue("is_hot") == "on"

		// Валідація
		if title == "" || content == "" || date == "" {
			tmpl, _ := template.ParseFiles("add.html")
			tmpl.Execute(w, FormResponse{
				Error: "Заповніть, будь ласка, всі поля!",
				Data:  NewsArticle{Title: title, Content: content, Date: date},
			})
			return
		}

		// Додаємо новину
		newArticle := NewsArticle{
			ID:       len(articles) + 1,
			Title:    title,
			Content:  content,
			Category: category,
			Date:     date,
			IsHot:    isHot,
		}
		// Додаємо в початок списку
		articles = append([]NewsArticle{newArticle}, articles...)

		// Редирект на головну
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/add", addNewsHandler) // Шлях до форми

	port := ":9000"
	fmt.Printf("Сервер запущено: http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
