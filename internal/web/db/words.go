package db

import (
	"database/sql"
	"fmt"
)

type Sense struct {
	ID        int
	Gloss     string
	Tags      string
	ExampleGd string
	ExampleEn string
}

type WordListView struct {
	Items      []WordResult
	NextOffset int
	Query      string
	POS        string
	Mode       string
}

type Form struct {
	ID   int
	Form string
	Tags string
}

type Synonym struct {
	ID      int
	Word    string
	SenseID sql.NullInt64
}

type Derived struct {
	ID      int
	Word    string
	English string
}

type WordEntry struct {
	ID        int
	Word      string
	POS       string
	Gender    string
	IPA       string
	Etymology string
	Verified  int
	Senses    []Sense
	Forms     []Form
	Synonyms  []Synonym
	Parents   []Derived
	Children  []Derived
}

type WordResult struct {
	ID   int
	Word string
	POS  string
}

func SearchWords(db *sql.DB, query, pos, mode string, offset int) ([]WordResult, error) {
	var rows *sql.Rows
	var err error

	switch mode {
	case "exact":
		rows, err = db.Query(`
			SELECT id, word, pos FROM lemmas
			WHERE (? = '' OR LOWER(word) = LOWER(?))
			AND (? = '' OR pos = ?)
			ORDER BY word
			LIMIT 50 OFFSET ?`,
			query, query, pos, pos, offset,
		)
	case "lemma":
		var lemma string
		err = db.QueryRow(`
			SELECT lemma_text FROM sentence_analyses
			WHERE token = ?
			LIMIT 1`, query).Scan(&lemma)
		if err != nil {
			lemma = query
		}
		rows, err = db.Query(`
			SELECT id, word, pos FROM lemmas
			WHERE (? = '' OR LOWER(word) LIKE '%' || LOWER(?) || '%')
			AND (? = '' OR pos = ?)
			ORDER BY word
			LIMIT 50 OFFSET ?`,
			lemma, lemma, pos, pos, offset,
		)
	default: // contains
		rows, err = db.Query(`
			SELECT id, word, pos FROM lemmas
			WHERE (? = '' OR LOWER(word) LIKE '%' || LOWER(?) || '%')
			AND (? = '' OR pos = ?)
			ORDER BY word
			LIMIT 50 OFFSET ?`,
			query, query, pos, pos, offset,
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []WordResult
	for rows.Next() {
		var r WordResult
		if err := rows.Scan(&r.ID, &r.Word, &r.POS); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func GetWord(db *sql.DB, id int) (WordEntry, error) {
	var entry WordEntry

	err := db.QueryRow(`
		SELECT id, word, pos, gender, ipa, etymology, verified
		FROM lemmas
		WHERE id = ?`, id).Scan(
		&entry.ID, &entry.Word, &entry.POS, &entry.Gender,
		&entry.IPA, &entry.Etymology, &entry.Verified,
	)
	if err == sql.ErrNoRows {
		return WordEntry{}, fmt.Errorf("word not found")
	}
	if err != nil {
		return WordEntry{}, err
	}

	// Senses
	rows, err := db.Query(`
		SELECT id, gloss, tags, example_gd, example_en
		FROM senses
		WHERE lemma_id = ?`, id)
	if err != nil {
		return WordEntry{}, err
	}
	for rows.Next() {
		var s Sense
		if err := rows.Scan(&s.ID, &s.Gloss, &s.Tags, &s.ExampleGd, &s.ExampleEn); err != nil {
			rows.Close()
			return WordEntry{}, err
		}
		entry.Senses = append(entry.Senses, s)
	}
	rows.Close()

	// Forms — exclude error tagged forms
	rows, err = db.Query(`
		SELECT id, form, tags
		FROM forms
		WHERE lemma_id = ?
		AND (tags NOT LIKE '%error%' OR tags IS NULL)`, id)
	if err != nil {
		return WordEntry{}, err
	}
	for rows.Next() {
		var f Form
		if err := rows.Scan(&f.ID, &f.Form, &f.Tags); err != nil {
			rows.Close()
			return WordEntry{}, err
		}
		entry.Forms = append(entry.Forms, f)
	}
	rows.Close()

	// Synonyms
	rows, err = db.Query(`
		SELECT id, word
		FROM synonyms
		WHERE lemma_id = ?`, id)
	if err != nil {
		return WordEntry{}, err
	}
	for rows.Next() {
		var s Synonym
		if err := rows.Scan(&s.ID, &s.Word); err != nil {
			rows.Close()
			return WordEntry{}, err
		}
		entry.Synonyms = append(entry.Synonyms, s)
	}
	rows.Close()

	// Parents — reverse derived lookup
	rows, err = db.Query(`
		SELECT l.word, d.english
		FROM derived d
		JOIN lemmas l ON l.id = d.lemma_id
		WHERE d.word = ?`, entry.Word)
	if err != nil {
		return WordEntry{}, err
	}
	for rows.Next() {
		var d Derived
		if err := rows.Scan(&d.Word, &d.English); err != nil {
			rows.Close()
			return WordEntry{}, err
		}
		entry.Parents = append(entry.Parents, d)
	}
	rows.Close()

	// Children — words derived from this lemma
	rows, err = db.Query(`
		SELECT id, word, english
		FROM derived
		WHERE lemma_id = ?`, id)
	if err != nil {
		return WordEntry{}, err
	}
	for rows.Next() {
		var d Derived
		if err := rows.Scan(&d.ID, &d.Word, &d.English); err != nil {
			rows.Close()
			return WordEntry{}, err
		}
		entry.Children = append(entry.Children, d)
	}
	rows.Close()

	return entry, nil
}
