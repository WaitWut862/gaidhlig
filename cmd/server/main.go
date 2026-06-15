package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"language_v1/internal/web/db"
)

func featsFormat(feats string) string {
	if feats == "" {
		return ""
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(feats), &m); err != nil {
		return feats
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+m[k])
	}
	return strings.Join(pairs, "|")
}

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"featsFormat": featsFormat,
}).ParseGlob("./internal/web/templates/*.html"))

type App struct {
	DB *sql.DB
}

func main() {
	database, err := sql.Open("sqlite3", "./internal/gaidhlig/gaidhlig.db")
	if err != nil {
		log.Fatal("Error opening db:", err)
	}
	if err := database.Ping(); err != nil {
		log.Fatal("Error connecting to db:", err)
	}
	defer database.Close()

	app := &App{DB: database}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleHome)
	mux.HandleFunc("/search", app.handleSearch)
	mux.HandleFunc("/word/{id}", app.handleWord)
	mux.HandleFunc("/sentence/{id}", app.handleSentence)
	mux.HandleFunc("/rule/{id}", app.handleRule)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./internal/web/static"))))

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	if err := tmpl.ExecuteTemplate(w, "base", nil); err != nil {
		log.Println("Template error:", err)
	}
}

func (a *App) handleWord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	entry, err := db.GetWord(a.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "word", entry); err != nil {
		log.Println("Template error:", err)
	}
}

func (a *App) handleSentence(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	entry, err := db.GetSentence(a.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tokens, err := db.GetSentenceAnalysis(a.DB, id)
	if err != nil {
		log.Println("Error fetching tokens:", err)
		// don't 404 — sentence exists, analysis may just be missing
	}
	view := db.SentenceView{Entry: entry, Tokens: tokens}
	if err := tmpl.ExecuteTemplate(w, "sentence", view); err != nil {
		log.Println("Template error:", err)
	}
}

func (a *App) handleRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	entry, err := db.GetRule(a.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "rule", entry); err != nil {
		log.Println("Template error:", err)
	}
}

func (a *App) handleSearch(w http.ResponseWriter, r *http.Request) {
	queryType := r.URL.Query().Get("type")
	query := r.URL.Query().Get("query")
	pos := r.URL.Query().Get("pos")
	exactForm := r.URL.Query().Get("exact") == "true"
	category := r.URL.Query().Get("category")
	mode := r.URL.Query().Get("mode")
	difficulty := r.URL.Query().Get("difficulty")
	offsetStr := r.URL.Query().Get("offset")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	initial := offset == 0

	switch queryType {
	case "words":
		results, err := db.SearchWords(a.DB, query, pos, mode, offset)
		if err != nil {
			log.Println("Error:", err)
			return
		}
		nextOffset := offset + 50
		if len(results) < 50 {
			nextOffset = -1
		}
		view := db.WordListView{Items: results, NextOffset: nextOffset, Query: query, POS: pos, Mode: mode}
		t := "items_words"
		if initial {
			t = "results_words"
		}
		if err := tmpl.ExecuteTemplate(w, t, view); err != nil {
			log.Println("Template error:", err)
		}
	case "sentences":
		results, err := db.SearchSentences(a.DB, query, exactForm, offset)
		if err != nil {
			log.Println("Error:", err)
			return
		}
		nextOffset := offset + 50
		if len(results) < 50 {
			nextOffset = -1
		}
		view := db.SentenceListView{Items: results, NextOffset: nextOffset, Query: query, Exact: r.URL.Query().Get("exact")}
		t := "items_sentences"
		if initial {
			t = "results_sentences"
		}
		if err := tmpl.ExecuteTemplate(w, t, view); err != nil {
			log.Println("Template error:", err)
		}
	case "rules":
		results, err := db.SearchRules(a.DB, query, category, difficulty, offset)
		if err != nil {
			log.Println("Error:", err)
			return
		}
		nextOffset := offset + 50
		if len(results) < 50 {
			nextOffset = -1
		}
		view := db.RuleListView{Items: results, NextOffset: nextOffset, Query: query, Category: category, Difficulty: difficulty}
		t := "items_rules"
		if initial {
			t = "results_rules"
		}
		if err := tmpl.ExecuteTemplate(w, t, view); err != nil {
			log.Println("Template error:", err)
		}
	}
}
