package gla

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Token represents one CoNLL-U token row
type Token struct {
	ID     int
	Form   string // surface form as it appears in text
	Lemma  string // dictionary form
	UPOS   string // universal POS tag (VERB, NOUN, etc.)
	XPOS   string // language-specific POS tag (V-p, Ncsfd, etc.)
	Feats  string // morphological features
	Head   int    // dependency head token ID (0 = root)
	DepRel string // dependency relation (nsubj, obl, etc.)
	Misc   string // miscellaneous (SpaceAfter, etc.)
}

// Sentence represents one analysed sentence
type Sentence struct {
	ID     string
	Text   string
	Tokens []Token
}

// scriptDir returns the directory containing this file at runtime
// so analyse.py can always be found relative to the Go package
func scriptDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

// Analyse sends text to the UDPipe Python wrapper and returns
// parsed sentences in CoNLL-U format.
// Multiple sentences in the input are handled correctly.
func Analyse(text string) ([]Sentence, error) {
	scriptPath := filepath.Join(scriptDir(), "analyse.py")

	cmd := exec.Command("python3", scriptPath)
	cmd.Stdin = strings.NewReader(text)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gla exec: %w — stderr: %s", err, stderr.String())
	}

	return parseCoNLLU(stdout.String())
}

// parseCoNLLU parses raw CoNLL-U output into Sentence structs
func parseCoNLLU(raw string) ([]Sentence, error) {
	var sentences []Sentence
	var current *Sentence

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()

		// Blank line signals end of a sentence
		if strings.TrimSpace(line) == "" {
			if current != nil {
				sentences = append(sentences, *current)
				current = nil
			}
			continue
		}

		// Comment lines carry metadata
		if strings.HasPrefix(line, "#") {
			if current == nil {
				current = &Sentence{}
			}
			if strings.HasPrefix(line, "# sent_id = ") {
				current.ID = strings.TrimPrefix(line, "# sent_id = ")
			}
			if strings.HasPrefix(line, "# text = ") {
				current.Text = strings.TrimPrefix(line, "# text = ")
			}
			continue
		}

		// Skip multi-word token ranges (e.g. "1-2")
		if strings.Contains(strings.Split(line, "\t")[0], "-") {
			continue
		}

		// Token line — tab separated, 10 fields
		fields := strings.Split(line, "\t")
		if len(fields) != 10 {
			continue
		}

		id, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		head, _ := strconv.Atoi(fields[6])

		token := Token{
			ID:     id,
			Form:   fields[1],
			Lemma:  fields[2],
			UPOS:   fields[3],
			XPOS:   fields[4],
			Feats:  fields[5],
			Head:   head,
			DepRel: fields[7],
			Misc:   fields[9],
		}

		if current == nil {
			current = &Sentence{}
		}
		current.Tokens = append(current.Tokens, token)
	}

	// Flush final sentence if file doesn't end with blank line
	if current != nil && len(current.Tokens) > 0 {
		sentences = append(sentences, *current)
	}

	return sentences, scanner.Err()
}
