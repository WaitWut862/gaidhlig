package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rivo/tview"
)

var db *sql.DB

type Rule struct {
	ID         int
	Title      string
	Text       string
	Category   string
	Difficulty string
	Verified   int
}

var (
	app         *tview.Application
	pages       *tview.Pages
	ruleList    *tview.List
	searchInput *tview.InputField
	statusBar   *tview.TextView
	allRules    []Rule
	filtered    []Rule
)

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./internal/gaidhlig/gaidhlig.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	app = tview.NewApplication()
	pages = tview.NewPages()

	buildMainPage()

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func buildMainPage() {
	// Search input
	searchInput = tview.NewInputField().
		SetLabel(" 🔍 Search: ").
		SetFieldWidth(40).
		SetChangedFunc(func(text string) {
			filterRules(text)
		})
	searchInput.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	searchInput.SetLabelColor(tcell.ColorYellow)

	// Rule list
	ruleList = tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true).
		SetSelectedFunc(func(index int, main, secondary string, shortcut rune) {
			if index < len(filtered) {
				showRuleDetail(filtered[index])
			}
		})
	ruleList.SetBorder(true).SetTitle(" Rules — Enter to view, N to new, D to delete ").SetTitleColor(tcell.ColorYellow)
	ruleList.SetSelectedBackgroundColor(tcell.ColorDarkSlateGray)

	// Status bar
	statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetText(" [yellow]Tab[white]: focus search  [yellow]N[white]: new rule  [yellow]D[white]: delete  [yellow]Q[white]: quit")
	statusBar.SetBackgroundColor(tcell.ColorDarkSlateGray)

	// Category filter buttons
	categories := []string{"all", "syntax", "morphology", "phonology", "phonetics", "typology", "terminology"}
	categoryBar := tview.NewFlex()
	for _, cat := range categories {
		c := cat // capture
		btn := tview.NewButton(c).SetSelectedFunc(func() {
			if c == "all" {
				searchInput.SetText("")
			} else {
				searchInput.SetText("cat:" + c)
			}
		})
		btn.SetStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorYellow))
		categoryBar.AddItem(btn, len(c)+4, 0, false)
	}
	categoryBar.SetBorder(false)

	// Layout
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(searchInput, 3, 0, true).
		AddItem(categoryBar, 1, 0, false).
		AddItem(ruleList, 0, 1, false).
		AddItem(statusBar, 1, 0, false)

	// Key bindings on the main layout
	layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			if app.GetFocus() == searchInput {
				app.SetFocus(ruleList)
			} else {
				app.SetFocus(searchInput)
			}
			return nil
		case tcell.KeyCtrlN:
			showNewRuleForm(nil)
			return nil
		case tcell.KeyCtrlD:
			idx := ruleList.GetCurrentItem()
			if idx >= 0 && idx < len(filtered) {
				confirmDelete(filtered[idx])
			}
			return nil
		case tcell.KeyCtrlQ:
			app.Stop()
			return nil
		}
		return event
	})

	pages.AddPage("main", layout, true, true)
	loadRules()
}

func loadRules() {
	rows, err := db.Query(`
		SELECT id, title, text, category, difficulty, verified
		FROM rules
		ORDER BY category, difficulty, title
	`)
	if err != nil {
		setStatus("[red]Error loading rules: " + err.Error())
		return
	}
	defer rows.Close()

	allRules = nil
	for rows.Next() {
		var r Rule
		rows.Scan(&r.ID, &r.Title, &r.Text, &r.Category, &r.Difficulty, &r.Verified)
		allRules = append(allRules, r)
	}

	filterRules(searchInput.GetText())
	setStatus(fmt.Sprintf("[green]Loaded %d rules", len(allRules)))
}

func filterRules(query string) {
	query = strings.TrimSpace(strings.ToLower(query))
	filtered = nil

	// Support cat:category filter
	catFilter := ""
	textFilter := query
	if strings.HasPrefix(query, "cat:") {
		parts := strings.SplitN(query, " ", 2)
		catFilter = strings.TrimPrefix(parts[0], "cat:")
		if len(parts) > 1 {
			textFilter = parts[1]
		} else {
			textFilter = ""
		}
	}

	for _, r := range allRules {
		if catFilter != "" && !strings.EqualFold(r.Category, catFilter) {
			continue
		}
		if textFilter != "" {
			if !strings.Contains(strings.ToLower(r.Title), textFilter) &&
				!strings.Contains(strings.ToLower(r.Text), textFilter) {
				continue
			}
		}
		filtered = append(filtered, r)
	}

	ruleList.Clear()
	for _, r := range filtered {
		verifiedMark := "○"
		if r.Verified == 1 {
			verifiedMark = "●"
		}
		title := fmt.Sprintf("%s [%s/%s] %s", verifiedMark, r.Category, r.Difficulty, r.Title)
		preview := r.Text
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		ruleList.AddItem(title, "  "+preview, 0, nil)
	}

	setStatus(fmt.Sprintf("[yellow]Showing %d of %d rules", len(filtered), len(allRules)))
}

func showRuleDetail(r Rule) {
	verifiedLabel := "Unverified"
	verifiedColor := "red"
	if r.Verified == 1 {
		verifiedLabel = "Verified"
		verifiedColor = "green"
	}

	detail := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetText(fmt.Sprintf(
			"[yellow]%s[white]\n\n"+
				"[gray]Category:[white]   %s\n"+
				"[gray]Difficulty:[white] %s\n"+
				"[gray]Status:[white]     [%s]%s[white]\n"+
				"[gray]ID:[white]         %d\n\n"+
				"[white]%s",
			r.Title, r.Category, r.Difficulty,
			verifiedColor, verifiedLabel,
			r.ID, r.Text,
		))
	detail.SetBorder(true).SetTitle(fmt.Sprintf(" Rule #%d ", r.ID)).SetTitleColor(tcell.ColorYellow)

	// Buttons
	editBtn := tview.NewButton("Edit").SetSelectedFunc(func() {
		showNewRuleForm(&r)
	})
	editBtn.SetStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorYellow))

	verifyBtn := tview.NewButton("Toggle Verified").SetSelectedFunc(func() {
		toggleVerified(r)
		pages.RemovePage("detail")
		loadRules()
	})
	verifyBtn.SetStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorGreen))

	deleteBtn := tview.NewButton("Delete").SetSelectedFunc(func() {
		pages.RemovePage("detail")
		confirmDelete(r)
	})
	deleteBtn.SetStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorRed))

	backBtn := tview.NewButton("Back (Esc)").SetSelectedFunc(func() {
		pages.RemovePage("detail")
		app.SetFocus(ruleList)
	})
	backBtn.SetStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite))

	btnRow := tview.NewFlex().
		AddItem(editBtn, 8, 0, false).
		AddItem(tview.NewBox(), 2, 0, false).
		AddItem(verifyBtn, 18, 0, false).
		AddItem(tview.NewBox(), 2, 0, false).
		AddItem(deleteBtn, 10, 0, false).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(backBtn, 14, 0, true)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(detail, 0, 1, false).
		AddItem(btnRow, 3, 0, true)

	layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.RemovePage("detail")
			app.SetFocus(ruleList)
			return nil
		}
		return event
	})

	pages.AddPage("detail", layout, true, true)
	app.SetFocus(backBtn)
}

func showNewRuleForm(existing *Rule) {
	title := tview.NewInputField().SetLabel("Title:      ").SetFieldWidth(60)
	category := tview.NewInputField().SetLabel("Category:   ").SetFieldWidth(20)
	difficulty := tview.NewInputField().SetLabel("Difficulty: ").SetFieldWidth(5)
	textArea := tview.NewTextArea().SetLabel("Text:\n")
	verified := tview.NewCheckbox().SetLabel("Verified:   ")

	// Prefill if editing
	formTitle := " New Rule "
	if existing != nil {
		title.SetText(existing.Title)
		category.SetText(existing.Category)
		difficulty.SetText(existing.Difficulty)
		textArea.SetText(existing.Text, true)
		verified.SetChecked(existing.Verified == 1)
		formTitle = fmt.Sprintf(" Edit Rule #%d ", existing.ID)
	}

	for _, f := range []*tview.InputField{title, category, difficulty} {
		f.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
		f.SetLabelColor(tcell.ColorYellow)
	}

	hint := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[gray]Categories: syntax, morphology, phonology, phonetics, typology, terminology\n" +
			"Difficulty: A1, A2, B1, B2, C1, C2")

	saveBtn := tview.NewButton("Save").SetSelectedFunc(func() {
		t := strings.TrimSpace(title.GetText())
		cat := strings.TrimSpace(category.GetText())
		diff := strings.TrimSpace(difficulty.GetText())
		txt := strings.TrimSpace(textArea.GetText())

		if t == "" || cat == "" || diff == "" || txt == "" {
			setStatus("[red]All fields required")
			return
		}

		v := 0
		if verified.IsChecked() {
			v = 1
		}

		var err error
		if existing != nil {
			_, err = db.Exec(
				"UPDATE rules SET title=?, text=?, category=?, difficulty=?, verified=? WHERE id=?",
				t, txt, cat, diff, v, existing.ID,
			)
		} else {
			_, err = db.Exec(
				"INSERT INTO rules(title, text, category, difficulty, verified) VALUES(?,?,?,?,?)",
				t, txt, cat, diff, v,
			)
		}

		if err != nil {
			setStatus("[red]Save error: " + err.Error())
			return
		}

		pages.RemovePage("form")
		if existing != nil {
			pages.RemovePage("detail")
		}
		loadRules()
		setStatus("[green]Rule saved")
	})
	saveBtn.SetStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorGreen))

	cancelBtn := tview.NewButton("Cancel (Esc)").SetSelectedFunc(func() {
		pages.RemovePage("form")
		app.SetFocus(ruleList)
	})
	cancelBtn.SetStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite))

	btnRow := tview.NewFlex().
		AddItem(saveBtn, 8, 0, false).
		AddItem(tview.NewBox(), 2, 0, false).
		AddItem(cancelBtn, 14, 0, true)

	form := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(title, 2, 0, true).
		AddItem(category, 2, 0, false).
		AddItem(difficulty, 2, 0, false).
		AddItem(verified, 2, 0, false).
		AddItem(hint, 3, 0, false).
		AddItem(textArea, 0, 1, false).
		AddItem(btnRow, 3, 0, false)
	form.SetBorder(true).SetTitle(formTitle).SetTitleColor(tcell.ColorYellow)

	// Tab between fields
	fields := []tview.Primitive{title, category, difficulty, verified, textArea, saveBtn, cancelBtn}
	fieldIdx := 0
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			pages.RemovePage("form")
			app.SetFocus(ruleList)
			return nil
		case tcell.KeyTab:
			fieldIdx = (fieldIdx + 1) % len(fields)
			app.SetFocus(fields[fieldIdx])
			return nil
		case tcell.KeyBacktab:
			fieldIdx = (fieldIdx - 1 + len(fields)) % len(fields)
			app.SetFocus(fields[fieldIdx])
			return nil
		}
		return event
	})

	pages.AddPage("form", form, true, true)
	app.SetFocus(title)
}

func confirmDelete(r Rule) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Delete rule #%d?\n\n\"%s\"", r.ID, r.Title)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Delete" {
				_, err := db.Exec("DELETE FROM rules WHERE id=?", r.ID)
				if err != nil {
					setStatus("[red]Delete error: " + err.Error())
				} else {
					setStatus(fmt.Sprintf("[green]Rule #%d deleted", r.ID))
				}
				loadRules()
			}
			pages.RemovePage("confirm")
			app.SetFocus(ruleList)
		})

	pages.AddPage("confirm", modal, false, true)
}

func toggleVerified(r Rule) {
	newVal := 1
	if r.Verified == 1 {
		newVal = 0
	}
	db.Exec("UPDATE rules SET verified=? WHERE id=?", newVal, r.ID)
}

func setStatus(msg string) {
	statusBar.SetText(" " + msg + "  [yellow]Tab[white]: focus  [yellow]N[white]: new  [yellow]D[white]: delete  [yellow]Q[white]: quit")
}
