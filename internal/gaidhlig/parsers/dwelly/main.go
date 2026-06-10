package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// incompleteTagID is fetched once at startup and reused
// It's the id of the 'incomplete' tag in the tags table
var incompleteTagID int64

// Patterns for identifying line types
var (
	// Headword line: starts with a lowercase (or accented) letter,
	// followed by a comma or space — e.g. "teasairg, pr.pt..."
	headwordRe = regexp.MustCompile(`^([a-zàáâãäåæçèéêëìíîïòóôõöùúûüýāăąćĉċčēĕėęěĝğġģĥħĩīĭįĵķĺļľłńņňŋōŏőœŕŗřśŝşšţťŧũūŭůűųŵŷźżžÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÒÓÔÕÖÙÚÛÜÝāăąćĉċčēĕėęěĝğġģĥħĩīĭįĵķĺļľłńņňŋōŏőœŕŗřśŝşšţťŧũūŭůűųŵŷźżž][a-zàáâãäåæçèéêëìíîïòóôõöùúûüýāăąćĉċčēĕėęěĝğġģĥħĩīĭįĵķĺļľłńņňŋōŏőœŕŗřśŝşšţťŧũūŭůűųŵŷźżž\-\' ]{1,40}),`)
	// Derived entry: starts with dashes e.g. "——eachd" or "—each"
	derivedRe = regexp.MustCompile(`^—+([a-zàáâãäåæçèéêëìíîïðñòóôõöøùúûüýA-Z\-\']*)`)

	// Sense number: a digit at the start or after whitespace e.g. "2 Lexicographer"
	senseNumRe = regexp.MustCompile(`(?:^|\s)(\d+)\s+(.+)`)

	// POS markers — used to extract pos and gender
	posMasculineRe = regexp.MustCompile(`\bs\.m\.`)
	posFeminineRe  = regexp.MustCompile(`\bs\.f\.`)
	posVerbRe      = regexp.MustCompile(`\bv\.[and]\.`)
	posAdjRe       = regexp.MustCompile(`\b(?:^|,\s*)a\.`)
	posAdvRe       = regexp.MustCompile(`\badv\.`)
)

type dwellyEntry struct {
	word    string
	pos     string
	gender  string
	senses  []string
	derived []dwellyDerived
}

type dwellyDerived struct {
	suffix  string // the suffix after dashes, e.g. "eachd"
	english string
}

func main() {
	db, err := sql.Open("sqlite3", "../../gaidhlig.db")
	if err != nil {
		fmt.Println("Error opening db:", err)
		return
	}
	defer db.Close()

	// Fetch or create the 'incomplete' tag once
	// All Dwelly entries get this tag since IPA, etymology,
	// forms, and other fields will be missing
	incompleteTagID, err = ensureTag(db, "status", "incomplete",
		"Entry is missing one or more fields that may be filled in later")
	if err != nil {
		fmt.Println("Error ensuring incomplete tag:", err)
		return
	}

	file, err := os.Open("Dwelly_djvu.txt")
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

	var (
		current  *dwellyEntry
		inserted int
		skipped  int
		total    int
	)

	flush := func() {
		if current == nil {
			return
		}
		// Skip if already in db from kaikki — kaikki data is better
		if alreadyExists(db, current.word) {
			skipped++
			current = nil
			return
		}
		if err := insertEntry(stmts, current); err != nil {
			fmt.Printf("Insert error for %q: %v\n", current.word, err)
		} else {
			inserted++
		}
		current = nil
	}

	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum < 2084 {
			continue
		}
		line := scanner.Text()
		total++

		// Strip noise markers: **, †, ‡, {, t at start of content
		cleaned := cleanLine(line)
		if cleaned == "" {
			continue
		}

		// Check if this is a derived sub-entry (starts with dashes)
		if derivedRe.MatchString(cleaned) {
			if current != nil {
				m := derivedRe.FindStringSubmatch(cleaned)
				suffix := m[1]
				english := extractEnglish(cleaned)
				current.derived = append(current.derived, dwellyDerived{
					suffix:  suffix,
					english: english,
				})
			}
			continue
		}

		// Check if this is a new headword entry
		if headwordRe.MatchString(cleaned) {
			// Flush previous entry before starting new one
			flush()

			m := headwordRe.FindStringSubmatch(cleaned)
			word := strings.TrimSpace(m[1])

			entry := &dwellyEntry{
				word:   word,
				pos:    extractPOS(cleaned),
				gender: extractGender(cleaned),
			}

			// Extract first sense from the same line
			english := extractEnglish(cleaned)
			if english != "" {
				entry.senses = append(entry.senses, english)
			}

			current = entry
			continue
		}

		// Check if this is a numbered additional sense e.g. "2 Lexicographer"
		if senseNumRe.MatchString(cleaned) && current != nil {
			matches := senseNumRe.FindAllStringSubmatch(cleaned, -1)
			for _, m := range matches {
				gloss := strings.TrimSpace(m[2])
				gloss = cleanLine(gloss)
				if gloss != "" {
					current.senses = append(current.senses, gloss)
				}
			}
			continue
		}

		// Continuation line — append to the last sense if we have one
		if current != nil && len(current.senses) > 0 {
			// Only append if it looks like prose, not a new structure
			trimmed := strings.TrimSpace(cleaned)
			if trimmed != "" && !headwordRe.MatchString(trimmed) {
				last := current.senses[len(current.senses)-1]
				current.senses[len(current.senses)-1] = last + " " + trimmed
			}
		}

	}

	// Flush final entry
	flush()

	if err := scanner.Err(); err != nil {
		fmt.Println("Scanner error:", err)
	}

	fmt.Printf("\nFinished.\n")
	fmt.Printf("  Total lines:  %d\n", total)
	fmt.Printf("  Inserted:     %d\n", inserted)
	fmt.Printf("  Skipped:      %d (already in db from kaikki)\n", skipped)
}

type statements struct {
	insertLemma   *sql.Stmt
	insertSense   *sql.Stmt
	insertDerived *sql.Stmt
	insertTag     *sql.Stmt
}

func prepareStatements(db *sql.DB) (*statements, error) {
	s := &statements{}
	var err error

	s.insertLemma, err = db.Prepare(`
		INSERT INTO lemmas(word, pos, gender, ipa, etymology, source, verified)
		VALUES(?, ?, ?, '', '', 'dwelly', 1)
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare lemma: %w", err)
	}

	s.insertSense, err = db.Prepare(`
		INSERT INTO senses(lemma_id, gloss, tags, topics, example_gd, example_en)
		VALUES(?, ?, '', '', '', '')
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare sense: %w", err)
	}

	s.insertDerived, err = db.Prepare(`
		INSERT INTO derived(lemma_id, word, english)
		VALUES(?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare derived: %w", err)
	}

	s.insertTag, err = db.Prepare(`
		INSERT OR IGNORE INTO tag_associations(tag_id, target_table, target_id, note)
		VALUES(?, 'lemmas', ?, 'Dwelly entry — IPA, etymology, and forms not available')
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare tag: %w", err)
	}

	return s, nil
}

func closeStatements(s *statements) {
	s.insertLemma.Close()
	s.insertSense.Close()
	s.insertDerived.Close()
	s.insertTag.Close()
}

func insertEntry(s *statements, e *dwellyEntry) error {
	if len(e.senses) == 0 {
		return nil // nothing useful to insert
	}

	res, err := s.insertLemma.Exec(e.word, e.pos, e.gender)
	if err != nil {
		return fmt.Errorf("insert lemma: %w", err)
	}

	lemmaID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("get lemma id: %w", err)
	}

	// Tag as incomplete
	if _, err := s.insertTag.Exec(incompleteTagID, lemmaID); err != nil {
		fmt.Printf("  tag error for %q: %v\n", e.word, err)
	}

	// Insert senses
	for _, gloss := range e.senses {
		gloss = strings.TrimSpace(gloss)
		if gloss == "" {
			continue
		}
		if _, err := s.insertSense.Exec(lemmaID, gloss); err != nil {
			fmt.Printf("  sense error for %q: %v\n", e.word, err)
		}
	}

	// Insert derived forms — reconstruct full word from headword + suffix
	for _, d := range e.derived {
		if d.suffix == "" && d.english == "" {
			continue
		}
		// Full derived word = headword stem + suffix
		// e.g. faclair + eachd = faclaireach
		// We store the suffix as-is and let the application reconstruct
		// or store a best-effort concatenation
		fullDerived := e.word + d.suffix
		if _, err := s.insertDerived.Exec(lemmaID, fullDerived, d.english); err != nil {
			fmt.Printf("  derived error for %q: %v\n", fullDerived, err)
		}
	}

	return nil
}

// alreadyExists checks lemmas table for any entry with this word
// regardless of POS — if kaikki has it in any form, skip it
func alreadyExists(db *sql.DB, word string) bool {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM lemmas WHERE word = ?", word).Scan(&count)
	return count > 0
}

// ensureTag gets or creates a tag and returns its id
func ensureTag(db *sql.DB, category, label, description string) (int64, error) {
	var id int64
	err := db.QueryRow(
		"SELECT id FROM tags WHERE category = ? AND label = ?",
		category, label,
	).Scan(&id)
	if err == nil {
		return id, nil
	}

	res, err := db.Exec(
		"INSERT INTO tags(category, label, description) VALUES(?, ?, ?)",
		category, label, description,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// extractPOS maps Dwelly POS markers to clean strings
func extractPOS(line string) string {
	switch {
	case posMasculineRe.MatchString(line) || posFeminineRe.MatchString(line):
		return "noun"
	case posVerbRe.MatchString(line):
		return "verb"
	case posAdjRe.MatchString(line):
		return "adj"
	case posAdvRe.MatchString(line):
		return "adv"
	default:
		return ""
	}
}

// extractGender pulls m/f from s.m. or s.f. markers
func extractGender(line string) string {
	switch {
	case posMasculineRe.MatchString(line):
		return "masculine"
	case posFeminineRe.MatchString(line):
		return "feminine"
	default:
		return ""
	}
}

// extractEnglish pulls the definition text from a line
// by stripping the headword, grammatical markers, and noise
func extractEnglish(line string) string {
	// Remove headword and inflection info up to the first real English word
	// Strategy: find the last grammatical marker and take everything after it
	markers := []string{
		"v.a.", "v.n.", "v.d.", "v.irr.",
		"s.m.", "s.f.", "s.m.f.", "s.f.ind.", "s.m.ind.",
		"adv.", "prep.", "conj.", "pron.", "interj.",
		"a.", "pr.pt.", "past pt.",
	}

	best := -1
	for _, m := range markers {
		idx := strings.Index(line, m)
		if idx != -1 {
			end := idx + len(m)
			if end > best {
				best = end
			}
		}
	}

	if best != -1 {
		rest := strings.TrimSpace(line[best:])
		// Remove leading punctuation artifacts
		rest = strings.TrimLeft(rest, ".,; ")
		return cleanLine(rest)
	}

	// No marker found — try to strip just the headword portion
	if headwordRe.MatchString(line) {
		m := headwordRe.FindStringSubmatch(line)
		rest := line[len(m[0]):]
		return cleanLine(strings.TrimSpace(rest))
	}

	return cleanLine(strings.TrimSpace(line))
}

// cleanLine strips Dwelly source markers and OCR noise
func cleanLine(line string) string {
	// Remove source markers
	line = strings.ReplaceAll(line, "**", "")
	line = strings.ReplaceAll(line, "††", "")
	line = strings.ReplaceAll(line, "‡‡", "")
	line = strings.ReplaceAll(line, "{", "")

	// Remove leading † or t used as dagger symbol by OCR
	line = regexp.MustCompile(`^[†‡t]\s`).ReplaceAllString(line, "")

	// Collapse multiple spaces
	line = regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")

	return strings.TrimSpace(line)
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
