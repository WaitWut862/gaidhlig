package db

import (
	"database/sql"
	"fmt"
)

type RuleEntry struct {
	ID         int
	Title      string
	Text       string
	Category   string
	Difficulty string
	Verified   int
}

type RuleListView struct {
	Items      []RuleEntry
	NextOffset int
	Query      string
	Category   string
	Difficulty string
}

func SearchRules(db *sql.DB, query, category, difficulty string, offset int) ([]RuleEntry, error) {
	rows, err := db.Query(`
		SELECT id, title, text, category, difficulty
		FROM rules
		WHERE (? = '' OR LOWER(title) LIKE '%' || LOWER(?) || '%'
			OR LOWER(text) LIKE '%' || LOWER(?) || '%')
		AND (? = '' OR category = ?)
		AND (? = '' OR difficulty = ?)
		ORDER BY category, difficulty, title
		LIMIT 50 OFFSET ?`,
		query, query, query,
		category, category,
		difficulty, difficulty,
		offset,
	)
	if err != nil {
		return nil, err
	}

	var results []RuleEntry
	for rows.Next() {
		var r RuleEntry
		if err := rows.Scan(&r.ID, &r.Title, &r.Text, &r.Category, &r.Difficulty); err != nil {
			rows.Close()
			return nil, err
		}
		results = append(results, r)
	}
	rows.Close()

	return results, rows.Err()
}

func GetRule(db *sql.DB, id int) (RuleEntry, error) {
	var entry RuleEntry

	err := db.QueryRow(`
		SELECT id, title, text, category, difficulty, verified
		FROM rules
		WHERE id = ?`, id).Scan(
		&entry.ID, &entry.Title, &entry.Text,
		&entry.Category, &entry.Difficulty, &entry.Verified,
	)
	if err == sql.ErrNoRows {
		return RuleEntry{}, fmt.Errorf("rule not found")
	}
	if err != nil {
		return RuleEntry{}, err
	}

	return entry, nil
}
