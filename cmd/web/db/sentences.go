package db

import (
	"database/sql"
	"fmt"
)

type SentenceEntry struct {
	ID     int
	TextGd string
	TextEn string
}

type SentenceView struct {
	Entry  SentenceEntry
	Tokens []Token
}

type SentenceListView struct {
	Items      []SentenceEntry
	NextOffset int
	Query      string
	Exact      string
}

type Token struct {
	ID             int
	Position       int
	Token          string
	LemmaText      string
	LemmaID        sql.NullInt64
	UPOS           string
	XPOS           string
	Feats          string
	DependencyHead int
	DepRel         string
}

func SearchSentences(db *sql.DB, query string, exactForm bool, offset int) ([]SentenceEntry, error) {
	var rows *sql.Rows
	var err error

	if exactForm {
		rows, err = db.Query(`
			SELECT id, text_gd, text_en
			FROM sentences
			WHERE text_gd LIKE '%' || ? || '%'
			ORDER BY id
			LIMIT 50 OFFSET ?`,
			query, offset,
		)
	} else {
		rows, err = db.Query(`
			SELECT DISTINCT s.id, s.text_gd, s.text_en
			FROM sentences s
			JOIN sentence_analyses sa ON sa.sentence_id = s.id
			WHERE sa.lemma_text = ?
			ORDER BY s.id
			LIMIT 50 OFFSET ?`,
			query, offset,
		)
	}

	if err != nil {
		return nil, err
	}

	var results []SentenceEntry
	for rows.Next() {
		var r SentenceEntry
		if err := rows.Scan(&r.ID, &r.TextGd, &r.TextEn); err != nil {
			rows.Close()
			return nil, err
		}
		results = append(results, r)
	}
	rows.Close()

	return results, rows.Err()
}

func GetSentence(db *sql.DB, id int) (SentenceEntry, error) {
	var entry SentenceEntry

	err := db.QueryRow(`
		SELECT id, text_gd, text_en
		FROM sentences
		WHERE id = ?`, id).Scan(
		&entry.ID, &entry.TextGd, &entry.TextEn,
	)
	if err == sql.ErrNoRows {
		return SentenceEntry{}, fmt.Errorf("sentence not found")
	}
	if err != nil {
		return SentenceEntry{}, err
	}

	return entry, nil
}

func GetSentenceAnalysis(db *sql.DB, id int) ([]Token, error) {
	rows, err := db.Query(`
		SELECT id, position, token, lemma_text, lemma_id,
			upos, xpos, feats, dependency_head, dependency_relation
		FROM sentence_analyses
		WHERE sentence_id = ?
		ORDER BY position`, id)
	if err != nil {
		return nil, err
	}

	var tokens []Token
	for rows.Next() {
		var t Token
		if err := rows.Scan(
			&t.ID, &t.Position, &t.Token, &t.LemmaText,
			&t.LemmaID, &t.UPOS, &t.XPOS, &t.Feats,
			&t.DependencyHead, &t.DepRel,
		); err != nil {
			rows.Close()
			return nil, err
		}
		tokens = append(tokens, t)
	}
	rows.Close()

	return tokens, rows.Err()
}
