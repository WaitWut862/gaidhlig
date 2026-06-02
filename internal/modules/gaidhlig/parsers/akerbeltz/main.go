package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var articles = []article{
	{url: "http://www.akerbeltz.org/index.php?title=VSO_and_Master_Yoda", category: "syntax"},
	{url: "http://www.akerbeltz.org/index.php?title=Existentials_or_I_think_therefore_I_am", category: "syntax"},
	{url: "http://www.akerbeltz.org/index.php?title=Experience_vs_Disposition_or_Tha_mi_sunndach", category: "syntax"},
	{url: "http://www.akerbeltz.org/index.php?title=To_bi_or_not_to_bi", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=The_Case_System_or_What_the_heck_is_a_vocative%3F", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Genitives_and_Possessives", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=How_to_gender_a_noun", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Plurals_shmurals_and_how_to_predict_them", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Possessives_and_syllabic_structure_or_Ar_n-Athair_a_tha_air_n%C3%A8amh", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Lenition_and_why_that_is_your_mothers_fault", category: "phonology"},
	{url: "http://www.akerbeltz.org/index.php?title=The_homo-organic_rule_or_When_not_to_lenite", category: "phonology"},
	{url: "http://www.akerbeltz.org/index.php?title=Nasalisation_or_When_to_speak_through_your_nose", category: "phonology"},
	{url: "http://www.akerbeltz.org/index.php?title=Nasalisation_2_or_Why_am_I_married_to_%C9%99_N%C9%AFN%CA%B2%C9%99_agam%3F", category: "phonology"},
	{url: "http://www.akerbeltz.org/index.php?title=Stress_placement_and_Why_going_up_is_a_bad_thing", category: "phonology"},
	{url: "http://www.akerbeltz.org/index.php?title=Broad_vs_Slender", category: "phonetics"},
	{url: "http://www.akerbeltz.org/index.php?title=Voiced_vs_Voiceless_or_Why_does_b_sound_like_p_but_not_really%3F", category: "phonetics"},
	{url: "http://www.akerbeltz.org/index.php?title=%C3%89iridh_e_is_ceannaidh_e%3F_or_the_Future_tense", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=The_Imperative_or_How_to_order_people_around", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Adjective_Ordering", category: "syntax"},
	{url: "http://www.akerbeltz.org/index.php?title=Aig,_air_agus_ann_an_or_The_severed_head", category: "syntax"},
	{url: "http://www.akerbeltz.org/index.php?title=PPP_-_Pronouns,_prepositions_and_their_pronunciation", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Interrogatives_or_Who_the_what_why%3F", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Numerals_or_How_to_write_a_cheque_in_Gaelic", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Adverbs_or_Thall_%27s_a-bhos", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Demonstratives_or_An_cat_ud_thall", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Expressions_of_Time", category: "syntax"},
	{url: "http://www.akerbeltz.org/index.php?title=Chaidh_e_dhan_ch%C3%A9ilidh_is_mi_cho_tinn_or_The_mystery_of_the_agus", category: "syntax"},
	{url: "http://www.akerbeltz.org/index.php?title=Epistemic_Modality_or_Do_I_HAVE_to_read_this%3F", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Stative_Verbs_or_How_to_run_in_suspended_animation", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=The_Fog_of_Terminology", category: "terminology"},
	{url: "http://www.akerbeltz.org/index.php?title=So_what%27s_one_of_those%3F", category: "typology"},
	{url: "http://www.akerbeltz.org/index.php?title=D%C3%A8an_moch-%C3%A9irigh_mh%C3%B3r_or_The_gender_of_verbal_nouns", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=As_t-samhradh_or_The_mysterious_t-", category: "phonology"},
	{url: "http://www.akerbeltz.org/index.php?title=A_h-uile%3F_Na_h-uile%3F%3F_C%C3%A0ch%3F%3F%3F_Gach%3F%3F%3F%3F", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Habemus_infinitivum_necne", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=The_many_functions_of_%C9%99", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Svarabhakti_or_The_Helping_Vowel", category: "phonology"},
	{url: "http://www.akerbeltz.org/index.php?title=Compensatory_lengthening_and_The_secret_of_time", category: "phonology"},
	{url: "http://www.akerbeltz.org/index.php?title=The_Three_Bs", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=M%C3%A0thraichean-c%C3%A9ile_or_Kinship", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Ball_gazing_genitives", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Genitives_in_-(th)rach", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=Hoigh,_an_dithis_agaibh!_or_Personal_numerals", category: "morphology"},
	{url: "http://www.akerbeltz.org/index.php?title=The_new-old_numerals_or_Why_this_sucks", category: "morphology"},
}

type article struct {
	url      string
	category string
}

type rule struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	Category   string `json:"category"`
	Difficulty string `json:"difficulty"`
}

func main() {
	db, err := sql.Open("sqlite3", "../../gaidhlig.db")
	if err != nil {
		fmt.Println("Error opening db:", err)
		return
	}
	defer db.Close()

	needsReviewID, err := ensureTag(db, "status", "needs-review",
		"AI-drafted content pending human verification")
	if err != nil {
		fmt.Println("Error ensuring tag:", err)
		return
	}

	stmt, err := db.Prepare(
		"INSERT INTO rules(title, text, category, difficulty, verified) VALUES(?, ?, ?, ?, 0)",
	)
	if err != nil {
		fmt.Println("Error preparing statement:", err)
		return
	}
	defer stmt.Close()

	tagStmt, err := db.Prepare(
		"INSERT OR IGNORE INTO tag_associations(tag_id, target_table, target_id, note) VALUES(?, 'rules', ?, 'AI-drafted from Akerbeltz')",
	)
	if err != nil {
		fmt.Println("Error preparing tag statement:", err)
		return
	}
	defer tagStmt.Close()

	totalInserted := 0
	totalFailed := 0

	for i, a := range articles {
		fmt.Printf("[%d/%d] Fetching: %s\n", i+1, len(articles), a.url)

		content, err := fetchArticle(a.url)

		//--debug--
		fmt.Printf("  Content length: %d bytes\n", len(content))
		//--debug--

		if err != nil {
			fmt.Printf("  Fetch error: %v\n", err)
			totalFailed++
			continue
		}

		if strings.TrimSpace(content) == "" {
			fmt.Println("  Empty article, skipping")
			continue
		}

		fmt.Println("  Sending to Claude...")
		rules, err := extractRules(content, a.category)
		if err != nil {
			fmt.Printf("  Claude error: %v\n", err)
			totalFailed++
			continue
		}

		if len(rules) == 0 {
			fmt.Println("  No rules extracted, skipping")
			continue
		}

		fmt.Printf("  Extracted %d rules, inserting...\n", len(rules))
		for _, r := range rules {
			res, err := stmt.Exec(r.Title, r.Text, r.Category, r.Difficulty)
			if err != nil {
				fmt.Printf("  Insert error for %q: %v\n", r.Title, err)
				continue
			}
			ruleID, _ := res.LastInsertId()
			tagStmt.Exec(needsReviewID, ruleID)
			totalInserted++
			fmt.Printf("  + %s [%s/%s]\n", r.Title, r.Category, r.Difficulty)
		}

		time.Sleep(5 * time.Second)
	}

	fmt.Printf("\nFinished.\n")
	fmt.Printf("  Rules inserted: %d\n", totalInserted)
	fmt.Printf("  Articles failed: %d\n", totalFailed)
}

func fetchArticle(url string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Identify as a browser to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:125.0) Gecko/20100101 Firefox/125.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	content := stripHTML(string(body))
	content = strings.TrimSpace(content)

	fmt.Printf("  Content: %d bytes\n", len(content))
	return content, nil
}

func stripHTML(s string) string {
	// Remove script and style blocks entirely
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	s = scriptRe.ReplaceAllString(s, "")
	s = styleRe.ReplaceAllString(s, "")

	// Replace block elements with newlines for readability
	blockRe := regexp.MustCompile(`(?i)<(br|p|div|h[1-6]|li|tr|td|th)[^>]*>`)
	s = blockRe.ReplaceAllString(s, "\n")

	// Strip all remaining tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	s = tagRe.ReplaceAllString(s, "")

	// Decode common HTML entities
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#039;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")

	// Collapse excessive whitespace and blank lines
	blankRe := regexp.MustCompile(`\n{3,}`)
	s = blankRe.ReplaceAllString(s, "\n\n")
	spaceRe := regexp.MustCompile(`[ \t]+`)
	s = spaceRe.ReplaceAllString(s, " ")

	return s
}

func extractRules(content, category string) ([]rule, error) {
	prompt := "You are processing a Scottish Gaelic grammar article for a language learning database.\n\n" +
		"Read the article below and produce a JSON array of rule entries. Each article may produce multiple rules if it covers distinct concepts — aim for 3-8 rules per article, capturing the most important learnable points.\n\n" +
		"Each entry must have exactly these fields:\n" +
		"- \"title\": short descriptive name, max 60 characters\n" +
		"- \"text\": clear plain English explanation suitable for a language learner, 2-5 sentences, no jargon without explanation\n" +
		"- \"category\": use \"" + category + "\" unless a specific rule clearly belongs to one of: phonetics, phonology, morphology, syntax, typology, terminology\n" +
		"- \"difficulty\": one of A1, A2, B1, B2, C1, C2 — be conservative, A1 is only for the most fundamental concepts\n\n" +
		"Return ONLY the JSON array. No markdown backticks, no explanation, no preamble. Start your response with [ and end with ].\n\n" +
		"Article:\n" + content

	cmd := exec.Command("claude")
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude exec: %w — stderr: %s", err, stderr.String())
	}

	raw := strings.TrimSpace(stdout.String())
	if raw == "" {
		return nil, fmt.Errorf("empty response from claude")
	}

	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON array found in response: %s", raw[:min(len(raw), 200)])
	}

	jsonStr := raw[start : end+1]

	var rules []rule
	if err := json.Unmarshal([]byte(jsonStr), &rules); err != nil {
		return nil, fmt.Errorf("json parse: %w — raw: %s", err, jsonStr[:min(len(jsonStr), 200)])
	}

	var valid []rule
	for _, r := range rules {
		if r.Title == "" || r.Text == "" {
			continue
		}
		if r.Category == "" {
			r.Category = category
		}
		if r.Difficulty == "" {
			r.Difficulty = "B1"
		}
		valid = append(valid, r)
	}

	return valid, nil
}

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

// func min(a, b int) int {
// 	if a < b {
// 		return a
// 	}
// 	return b
// }

func stripWikiMarkup(s string) string {
	scanner := bufio.NewScanner(strings.NewReader(s))
	var out strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "{{") ||
			strings.HasPrefix(line, "[[Category") ||
			strings.HasPrefix(line, "[[File") ||
			strings.HasPrefix(line, "[[Faidhle") {
			continue
		}
		line = strings.ReplaceAll(line, "'''", "")
		line = strings.ReplaceAll(line, "''", "")
		out.WriteString(line)
		out.WriteString("\n")
	}
	return out.String()
}
