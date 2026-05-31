package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"

	_ "github.com/mattn/go-sqlite3"
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
		fmt.Println("Error opening file: ", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var buf []string

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")

		if len(fields) != 4 {
			continue
		}

		if fields[1] == "gla:lemma" {
			buf = append(buf, fields[2])
		}
	}

	var dictionary []entry

	for batch := range slices.Chunk(buf, 10) {
		fmt.Println(batch)
		dictionary = getDescriptions(batch)
		fmt.Println(dictionary)
		err := InsertToDB(dictionary)
		if err != nil {
			fmt.Println("error: ", err, "dictionary: ", dictionary)
		}
	}
}

func getDescriptions(words []string) []entry {
	wordList := strings.Join(words, ", ")
	prompt := fmt.Sprintf(`You are a Scottish Gaelic dictionary API. You will receive a list of Scottish Gaelic words. Return one entry per word in the same order. Output a JSON array of objects. Each object has exactly two fields: "definitions" and "cefr". No extra fields, no explanation, no markdown, no prose. Field rules: "definitions" is an array of English definitions for that word — include grammatical role if applicable (e.g. verb, preposition) and all distinct meanings as separate entries. "cefr" is a single CEFR level string (A1–C2), be conservative and default to B1 or higher unless the word is extremely common. Constraints: return exactly as many objects as there are words in the input. Preserve input order — the Nth object corresponds to the Nth word. One word equals one concept in the list even if it contains spaces. Words: %s`, wordList)
	var content contentResponse
	for len(content.Entries) != len(words) {
		reqBody := fmt.Sprintf(`{
	"model":  "gemma4",
	"stream": false,
	"messages": [
		{"role": "user", "content": "%s"}
	],
	"format": {
		"type": "object",
		"properties": {
			"entries": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"definitions": {
							"type": "array",
							"items": {"type": "string"}
						},
						"cefr": {"type": "string"}
					},
					"required": ["definitions", "cefr"]
				}
			}
		},
		"required": ["entries"]
	}
}`, prompt)

		resp, err := http.Post("http://localhost:11434/api/chat", "application/json", bytes.NewBuffer([]byte(reqBody)))
		if err != nil {
			fmt.Println(err)
			return nil
		}
		defer resp.Body.Close()

		var b bytes.Buffer
		io.Copy(&b, resp.Body)

		fmt.Printf("Resp body: %s", b.String())

		var result apiResponse
		json.NewDecoder(resp.Body).Decode(&result)

		json.Unmarshal([]byte(result.Message.Content), &content)
	}

	if len(content.Entries) != len(words) {
		panic(fmt.Errorf("content entry length does not match words length"))
	}

	for i := range content.Entries {
		content.Entries[i].Word = words[i]
	}

	return content.Entries
}

func InsertToDB(dictionary []entry) error {
	db, err := sql.Open("sqlite3", "../../gaidhlig.db")
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO words(text, definition, cefr) VALUES(?, ?, ?)")
	if err != nil {
		return err
	}

	for _, e := range dictionary {
		_, err := stmt.Exec(e.Word, strings.Join(e.Definitions, ", "), e.CEFR)
		if err != nil {
			fmt.Println("Insert error:", err)
		}
	}

	return nil
}
