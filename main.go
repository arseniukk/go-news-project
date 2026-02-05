package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

type NewsArticle struct {
	ID       int
	Title    string
	Content  string
	Category string
	Date     string
	IsHot    bool
}

type PageData struct {
	PageTitle string
	Articles  []NewsArticle
	Year      int // Для автоматичного підвалу сайту
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		return
	}

	// 1. Отримуємо категорію з URL
	categoryFilter := r.URL.Query().Get("category")

	// Повний список новин
	allArticles := []NewsArticle{
		{1, "Go 1.22: Революція у шаблонах", "Нова версія мови приносить ще більше швидкості у рендеринг сторінок.", "Технології", "05.02.2026", true},
		{2, "Київ стає цифровим хабом", "За останній рік кількість ІТ-стартапів зросла на 30%.", "Події", "04.02.2026", false},
		{3, "Майбутнє штучного інтелекту", "Чи замінить ШІ програмістів у 2026 році? Думки експертів.", "Технології", "03.02.2026", false},
		{4, "Спортивні досягнення тижня", "Українські атлети здобули 5 золотих медалей на міжнародних змаганнях.", "Спорт", "02.02.2026", true},
	}

	// 2. Фільтруємо новини, якщо категорія обрана
	var filteredArticles []NewsArticle
	if categoryFilter != "" {
		for _, a := range allArticles {
			if a.Category == categoryFilter {
				filteredArticles = append(filteredArticles, a)
			}
		}
	} else {
		// Якщо категорія не обрана, показуємо всі новини
		filteredArticles = allArticles
	}

	// 3. Передаємо відфільтровані новини в шаблон
	data := PageData{
		PageTitle: "Новини Головного",
		Articles:  filteredArticles,
		Year:      time.Now().Year(),
	}

	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		log.Printf("Помилка шаблону: %v", err)
		http.Error(w, "Помилка сервера", 500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}
func main() {
	http.HandleFunc("/", homeHandler)

	fmt.Println("Сервер намагається запуститися на http://localhost:8081") // Змінимо на 8081 для перевірки

	// Додаємо логування помилки
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Fatal("Сервер не зміг запуститися: ", err)
	}
}
