package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type KaikkiEntry struct {
	Word          string    `json:"word"`
	Pos           string    `json:"pos"`
	EtymologyText string    `json:"etymology_text"`
	Sounds        []Sound   `json:"sounds"`
	Senses        []Sense   `json:"senses"`
	Forms         []Form    `json:"forms"`
	Derived       []Related `json:"derived"`
	Synonyms      []Synonym `json:"synonyms"`
}

type Sound struct {
	IPA string `json:"ipa"`
}

type Sense struct {
	Glosses  []string  `json:"glosses"`
	Tags     []string  `json:"tags"`
	Topics   []string  `json:"topics"`
	Examples []Example `json:"examples"`
	Synonyms []Synonym `json:"synonyms"`
}

type Example struct {
	Text    string `json:"text"`
	English string `json:"english"`
}

type Form struct {
	Form string   `json:"form"`
	Tags []string `json:"tags"`
}

type Related struct {
	Word    string `json:"word"`
	English string `json:"english"`
}

type Synonym struct {
	Word  string `json:"word"`
	Sense string `json:"sense"`
}

// insertResult tracks what happened to one entry
// so we can report accurately without conflating
// different kinds of failures
type insertResult struct {
	lemmaOK        bool
	sensesFailed   int
	formsFailed    int
	synonymsFailed int
	derivedFailed  int
}

func main() {
	db, err := sql.Open("sqlite3", "./internal/gaidhlig/gaidhlig.db")
	if err != nil {
		fmt.Println("Error opening db:", err)
		return
	}
	defer db.Close()

	file, err := os.Open("./resources/kaikki.org-dictionary-ScottishGaelic.jsonl")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	stmts, err := prepareStatements(db)
	if err != nil {
		fmt.Println("Error preparing statements:", err)
		return
	}
	defer closeStatements(stmts)

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// Counters for final report
	var (
		total           int
		inserted        int
		skipped         int
		partialFailures int
	)

	for scanner.Scan() {
		total++

		var entry KaikkiEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			fmt.Printf("Line %d: parse error: %v\n", total, err)
			skipped++
			continue
		}

		// Skip if this word+pos already exists in the db
		// This makes restarts safe — already-processed entries
		// are skipped without re-inserting or erroring
		if alreadyExists(db, entry.Word, entry.Pos) {
			skipped++
			continue
		}

		result, err := insertEntry(stmts, entry)
		if err != nil {
			// Hard failure — lemma itself didn't insert
			fmt.Printf("Line %d: failed to insert %q (%s): %v\n",
				total, entry.Word, entry.Pos, err)
			skipped++
			continue
		}

		// Lemma inserted — count it even if sub-tables had issues
		inserted++

		// Report partial failures so you can investigate
		// without stopping the whole run
		if !result.lemmaOK ||
			result.sensesFailed > 0 ||
			result.formsFailed > 0 ||
			result.synonymsFailed > 0 ||
			result.derivedFailed > 0 {
			partialFailures++
			fmt.Printf("Line %d: %q inserted with issues — senses failed: %d, forms: %d, synonyms: %d, derived: %d\n",
				total,
				entry.Word,
				result.sensesFailed,
				result.formsFailed,
				result.synonymsFailed,
				result.derivedFailed,
			)
		}

		// Print progress every 500 entries so you can
		// see it's running without flooding the terminal
		if total%500 == 0 {
			fmt.Printf("Progress: %d processed, %d inserted, %d skipped\n",
				total, inserted, skipped)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Scanner error:", err)
	}

	fmt.Printf("\nFinished.\n")
	fmt.Printf("  Total lines:      %d\n", total)
	fmt.Printf("  Inserted:         %d\n", inserted)
	fmt.Printf("  Skipped:          %d\n", skipped)
	fmt.Printf("  Partial failures: %d\n", partialFailures)
}

type statements struct {
	insertLemma   *sql.Stmt
	insertSense   *sql.Stmt
	insertForm    *sql.Stmt
	insertSynonym *sql.Stmt
	insertDerived *sql.Stmt
}

func prepareStatements(db *sql.DB) (*statements, error) {
	s := &statements{}
	var err error

	s.insertLemma, err = db.Prepare(`
		INSERT INTO lemmas(word, pos, gender, ipa, etymology, source, verified)
		VALUES(?, ?, ?, ?, ?, 'kaikki', 1)
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare lemma: %w", err)
	}

	s.insertSense, err = db.Prepare(`
		INSERT INTO senses(lemma_id, gloss, tags, topics, example_gd, example_en)
		VALUES(?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare sense: %w", err)
	}

	s.insertForm, err = db.Prepare(`
		INSERT OR IGNORE INTO forms(lemma_id, form, tags)
		VALUES(?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare form: %w", err)
	}

	s.insertSynonym, err = db.Prepare(`
		INSERT INTO synonyms(lemma_id, sense_id, word)
		VALUES(?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare synonym: %w", err)
	}

	s.insertDerived, err = db.Prepare(`
		INSERT INTO derived(lemma_id, word, english)
		VALUES(?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare derived: %w", err)
	}

	return s, nil
}

func closeStatements(s *statements) {
	s.insertLemma.Close()
	s.insertSense.Close()
	s.insertForm.Close()
	s.insertSynonym.Close()
	s.insertDerived.Close()
}

func insertEntry(s *statements, entry KaikkiEntry) (insertResult, error) {
	result := insertResult{}

	ipa := ""
	for _, sound := range entry.Sounds {
		if sound.IPA != "" {
			ipa = sound.IPA
			break
		}
	}

	gender := extractGender(entry.Senses)

	res, err := s.insertLemma.Exec(
		entry.Word,
		entry.Pos,
		gender,
		ipa,
		entry.EtymologyText,
	)
	if err != nil {
		// Hard failure — return error to caller
		return result, fmt.Errorf("insert lemma: %w", err)
	}

	result.lemmaOK = true

	lemmaID, err := res.LastInsertId()
	if err != nil {
		return result, fmt.Errorf("get lemma id: %w", err)
	}

	for _, sense := range entry.Senses {
		if len(sense.Glosses) == 0 {
			continue
		}

		gloss := strings.Join(sense.Glosses, "; ")
		tags := toJSON(sense.Tags)
		topics := toJSON(sense.Topics)

		exGd, exEn := "", ""
		if len(sense.Examples) > 0 {
			exGd = sense.Examples[0].Text
			exEn = sense.Examples[0].English
		}

		res, err := s.insertSense.Exec(lemmaID, gloss, tags, topics, exGd, exEn)
		if err != nil {
			// Soft failure — log and continue to next sense
			result.sensesFailed++
			continue
		}

		senseID, err := res.LastInsertId()
		if err != nil {
			result.sensesFailed++
			continue
		}

		for _, syn := range sense.Synonyms {
			if syn.Word == "" {
				continue
			}
			if _, err := s.insertSynonym.Exec(lemmaID, senseID, syn.Word); err != nil {
				result.synonymsFailed++
			}
		}
	}

	for _, syn := range entry.Synonyms {
		if syn.Word == "" {
			continue
		}
		if _, err := s.insertSynonym.Exec(lemmaID, nil, syn.Word); err != nil {
			result.synonymsFailed++
		}
	}

	for _, form := range entry.Forms {
		if form.Form == "" {
			continue
		}
		tags := strings.Join(form.Tags, ",")
		if _, err := s.insertForm.Exec(lemmaID, form.Form, tags); err != nil {
			result.formsFailed++
		}
	}

	for _, d := range entry.Derived {
		if d.Word == "" {
			continue
		}
		if _, err := s.insertDerived.Exec(lemmaID, d.Word, d.English); err != nil {
			result.derivedFailed++
		}
	}

	return result, nil
}

func alreadyExists(db *sql.DB, word, pos string) bool {
	var count int
	db.QueryRow(
		"SELECT COUNT(*) FROM lemmas WHERE word = ? AND pos = ?",
		word, pos,
	).Scan(&count)
	return count > 0
}

func extractGender(senses []Sense) string {
	for _, s := range senses {
		for _, t := range s.Tags {
			if t == "masculine" || t == "feminine" {
				return t
			}
		}
	}
	return ""
}

func toJSON(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	b, err := json.Marshal(ss)
	if err != nil {
		return ""
	}
	return string(b)
}
