package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
)

type entry struct {
	Word        string   `json:"word"`
	Definitions []string `json:"definitions"`
	CEFR        string   `json:"cefr"`
}

type contentResponse struct {
	Entries []entry `json:"entries"`
}

type apiResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
}

func main() {
	file, err := os.Open("USGW_v0.9.tab")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	db, err := sql.Open("sqlite3", "../../gaidhlig.db")
	if err != nil {
		fmt.Println("Error opening db:", err)
		return
	}
	defer db.Close()

	// Deduplicate lemmas as we parse — prevents duplicate AI calls
	// and duplicate inserts from synsets sharing lemmas
	seen := make(map[string]bool)
	var buf []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) != 4 {
			continue
		}
		if fields[1] == "gla:lemma" && !seen[fields[2]] {
			seen[fields[2]] = true
			buf = append(buf, fields[2])
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Scanner error:", err)
		return
	}

	fmt.Printf("Total unique lemmas: %d\n", len(buf))

	for batch := range slices.Chunk(buf, 10) {
		// Filter out words already in the db — makes restarts safe and fast
		// without re-calling the AI for already-processed words
		pending := make([]string, 0, len(batch))
		for _, w := range batch {
			if !alreadyInserted(db, w) {
				pending = append(pending, w)
			}
		}
		if len(pending) == 0 {
			fmt.Println("Batch already complete, skipping")
			continue
		}

		fmt.Printf("Processing: %v\n", pending)
		entries := getDescriptions(pending)
		if entries == nil {
			fmt.Println("Skipping batch due to nil result")
			continue
		}

		if err := InsertToDB(db, entries); err != nil {
			fmt.Println("Insert error:", err)
		}
	}

	fmt.Println("Done.")
}

// alreadyInserted checks whether a word exists in the db
// so we can skip it on restart rather than re-calling the AI
func alreadyInserted(db *sql.DB, word string) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM words WHERE text = ?", word).Scan(&count)
	if err != nil {
		fmt.Printf("DB check error for %q: %v\n", word, err)
		return false
	}
	return count > 0
}

func getDescriptions(words []string) []entry {
	wordList := strings.Join(words, ", ")
	prompt := fmt.Sprintf(
		`You are a Scottish Gaelic dictionary API. You will receive a list of Scottish Gaelic words. `+
			`Return one entry per word in the same order. Output a JSON object. `+
			`The object has one field: "entries", which is an array of objects. `+
			`Each object has exactly two fields: "definitions" and "cefr". `+
			`No extra fields, no explanation, no markdown, no prose. `+
			`Field rules: "definitions" is an array of English definitions for that word — `+
			`include grammatical role if applicable and all distinct meanings as separate entries. `+
			`"cefr" is a single CEFR level string (A1-C2), default to B1 or higher unless extremely common. `+
			`Return exactly as many objects as there are words. Preserve input order. `+
			`Words: %s`, wordList)

	type ollamaMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type ollamaRequest struct {
		Model    string          `json:"model"`
		Stream   bool            `json:"stream"`
		Messages []ollamaMessage `json:"messages"`
		Format   map[string]any  `json:"format"`
	}

	format := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"entries": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"definitions": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"cefr": map[string]any{"type": "string"},
					},
					"required": []string{"definitions", "cefr"},
				},
			},
		},
		"required": []string{"entries"},
	}

	reqStruct := ollamaRequest{
		Model:  "gemma4",
		Stream: false,
		Messages: []ollamaMessage{
			{Role: "user", Content: prompt},
		},
		Format: format,
	}

	// Marshal once outside the loop — the request doesn't change between retries
	reqBytes, err := json.Marshal(reqStruct)
	if err != nil {
		fmt.Println("Error marshaling request:", err)
		return nil
	}

	var content contentResponse

	for len(content.Entries) != len(words) {
		resp, err := http.Post(
			"http://localhost:11434/api/chat",
			"application/json",
			bytes.NewBuffer(reqBytes),
		)
		if err != nil {
			fmt.Println("HTTP error:", err)
			return nil
		}

		// Read body once into a buffer — decoding from resp.Body directly
		// after any other read would return nothing (body is a one-time stream)
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Println("Error reading body:", err)
			return nil
		}

		var result apiResponse
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Println("Error decoding apiResponse:", err)
			continue
		}

		// Reset content before each attempt so partial results don't carry over
		content = contentResponse{}
		if err := json.Unmarshal([]byte(result.Message.Content), &content); err != nil {
			fmt.Printf("Error decoding content: %v\nRaw content: %s\n",
				err, result.Message.Content)
			continue
		}

		if len(content.Entries) != len(words) {
			fmt.Printf("Entry count mismatch: got %d, want %d — retrying\n",
				len(content.Entries), len(words))
		}
	}

	// Attach the original word to each entry — the AI only returns
	// definitions and cefr, so we pair by position
	for i := range content.Entries {
		content.Entries[i].Word = words[i]
	}

	return content.Entries
}

func InsertToDB(db *sql.DB, dictionary []entry) error {
	// INSERT OR IGNORE means duplicate words skip silently
	// rather than returning an error — requires UNIQUE on the text column:
	// CREATE TABLE words (
	//     id         INTEGER PRIMARY KEY AUTOINCREMENT,
	//     text       TEXT NOT NULL UNIQUE,
	//     definition TEXT NOT NULL,
	//     cefr       TEXT NOT NULL
	// );
	stmt, err := db.Prepare("INSERT OR IGNORE INTO words(text, definition, cefr) VALUES(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range dictionary {
		_, err := stmt.Exec(e.Word, strings.Join(e.Definitions, ", "), e.CEFR)
		if err != nil {
			fmt.Printf("Insert error for %q: %v\n", e.Word, err)
		}
	}
	return nil
}
