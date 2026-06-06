package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	gla "language_v1/internal/modules/gaidhlig/gla"
)

func main() {
	db, err := sql.Open("sqlite3", "./internal/modules/gaidhlig/gaidhlig.db")
	if err != nil {
		fmt.Println("Error opening db:", err)
		return
	}
	defer db.Close()

	// Pull all example sentences from senses
	rows, err := db.Query(`
		SELECT s.example_gd, s.example_en
		FROM senses s
		WHERE s.example_gd IS NOT NULL
		AND s.example_gd != ''
	`)
	if err != nil {
		fmt.Println("Error querying sentences:", err)
		return
	}
	defer rows.Close()

	type rawSentence struct {
		gd string
		en string
	}

	var sentences []rawSentence
	for rows.Next() {
		var r rawSentence
		rows.Scan(&r.gd, &r.en)
		r.gd = strings.TrimSpace(r.gd)
		r.en = strings.TrimSpace(r.en)
		if r.gd != "" {
			sentences = append(sentences, r)
		}
	}

	fmt.Printf("Found %d example sentences\n", len(sentences))

	// Prepare statements
	insertSentence, err := db.Prepare(`
		INSERT OR IGNORE INTO sentences(hash, text_gd, text_en, source, difficulty, raw_conllu)
		VALUES(?, ?, ?, 'kaikki', '', ?)
	`)
	if err != nil {
		fmt.Println("Error preparing sentence stmt:", err)
		return
	}
	defer insertSentence.Close()

	insertAnalysis, err := db.Prepare(`
		INSERT INTO sentence_analyses(
			sentence_id, token, lemma_text, position,
			upos, xpos, feats, dependency_head,
			dependency_relation, misc
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		fmt.Println("Error preparing analysis stmt:", err)
		return
	}
	defer insertAnalysis.Close()

	inserted := 0
	skipped := 0
	failed := 0

	for i, s := range sentences {
		// Hash to deduplicate — same sentence from multiple
		// senses won't be analysed twice
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(s.gd)))

		// Check if already processed
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM sentences WHERE hash = ?", hash).Scan(&exists)
		if exists > 0 {
			skipped++
			continue
		}

		// Run GLA analysis
		analysed, err := gla.Analyse(s.gd)
		if err != nil {
			fmt.Printf("[%d/%d] GLA error for %q: %v\n", i+1, len(sentences), s.gd, err)
			failed++
			continue
		}

		if len(analysed) == 0 {
			skipped++
			continue
		}

		// Build raw CoNLL-U string from all sentences
		// (input may contain multiple sentences)
		var conlluParts []string
		for _, sent := range analysed {
			var lines []string
			lines = append(lines, fmt.Sprintf("# sent_id = %s", sent.ID))
			lines = append(lines, fmt.Sprintf("# text = %s", sent.Text))
			for _, tok := range sent.Tokens {
				lines = append(lines, fmt.Sprintf(
					"%d\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t_\t%s",
					tok.ID, tok.Form, tok.Lemma,
					tok.UPOS, tok.XPOS, tok.Feats,
					tok.Head, tok.DepRel, tok.Misc,
				))
			}
			conlluParts = append(conlluParts, strings.Join(lines, "\n"))
		}
		rawCoNLLU := strings.Join(conlluParts, "\n\n")

		// Insert sentence
		res, err := insertSentence.Exec(hash, s.gd, s.en, rawCoNLLU)
		if err != nil {
			fmt.Printf("[%d/%d] Insert error for %q: %v\n", i+1, len(sentences), s.gd, err)
			failed++
			continue
		}

		sentenceID, err := res.LastInsertId()
		if err != nil || sentenceID == 0 {
			skipped++
			continue
		}

		// Insert token analyses
		// Use first sentence only if input produced multiple
		// (kaikki examples are single sentences)
		if len(analysed) > 0 {
			for _, tok := range analysed[0].Tokens {
				// Resolve lemma_id if the lemma exists in our dictionary
				var lemmaID sql.NullInt64
				db.QueryRow(
					"SELECT id FROM lemmas WHERE word = ? LIMIT 1",
					tok.Lemma,
				).Scan(&lemmaID)

				// Encode feats as JSON for storage
				featsJSON := encodeFeats(tok.Feats)

				_, err := insertAnalysis.Exec(
					sentenceID,
					tok.Form,
					tok.Lemma,
					tok.ID,
					tok.UPOS,
					tok.XPOS,
					featsJSON,
					tok.Head,
					tok.DepRel,
					tok.Misc,
				)
				if err != nil {
					fmt.Printf("  token insert error: %v\n", err)
				}
			}
		}

		inserted++

		if i%50 == 0 {
			fmt.Printf("Progress: %d/%d — inserted: %d skipped: %d failed: %d\n",
				i+1, len(sentences), inserted, skipped, failed)
		}
	}

	fmt.Printf("\nFinished.\n")
	fmt.Printf("  Inserted: %d\n", inserted)
	fmt.Printf("  Skipped:  %d\n", skipped)
	fmt.Printf("  Failed:   %d\n", failed)
}

// encodeFeats converts "Case=Dat|Gender=Fem|Number=Sing" into
// a JSON object for structured storage
// Returns empty string if feats is "_"
func encodeFeats(feats string) string {
	if feats == "_" || feats == "" {
		return ""
	}
	m := make(map[string]string)
	for _, pair := range strings.Split(feats, "|") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return feats
	}
	return string(b)
}
