package tui

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// KEYS — every key the Window answers, and every card of keys (SPC's, and a card a key on it
// opens), declared once. From this catalog come main.qml's Window shortcuts and its cards'
// declarations, each card's QML (views/<View>.qml), its text, and the cards as help shows them.
// The QML is generated: TestKeysAreGenerated fails when a file differs from the catalog, and
// `go test ./tui -run TestKeysAreGenerated -update` rewrites them. A feature adds keys and cards
// with the Keyed capability.

// Key is one key.
type Key struct {
	// Card is where it is answered: "" the Window itself, "leader" SPC's card, or the ID of a card a
	// key opens.
	Card string
	Seq  string   // the Shortcut's sequence: "Ctrl+G", "Shift+W", "`"
	Also []string // more sequences that do the same, shown nowhere: SPC SPC beside SPC /
	Show string   // how a card names it ("W"), or how help names a Window key ("Ctrl+h/j/k/l")
	Does string   // what it does, as its card says it; "" for a key no card lists
	Run  string   // the QML it runs: "App.openSearch()", "help.open()"
	// Opens is the card the key opens, in place of Run: its card closes and that one opens.
	Opens string
}

// Card is a card of keys: a Dialog listing its keys, each closing the card before it runs.
type Card struct {
	ID    string // the Dialog's id: "leader"
	View  string // its file, views/<View>.qml: "Leader"
	Title string // its frame's title: "SPC — commands"
	// Back is the card Backspace returns to; "" for none (SPC's card, opened from the Window).
	Back string
	// Help is how help heads the card's list: "THE LEADER CARD (…)".
	Help string
}

// Keyed is a feature with keys of its own, and cards of its own for keys to open.
type Keyed interface {
	Keys() []Key
	Cards() []Card
}

// leader is SPC's card: Space in Vim's Normal mode, Ctrl+Space in every editor mode.
const leader = "leader"

// cards are the host's cards.
var cards = []Card{{ID: leader, View: "Leader", Title: "SPC — commands",
	Help: "THE LEADER CARD (Space in Normal mode, Ctrl+Space in every mode)"}}

// windowKeys fire whichever widget has focus, when it leaves them. Space and Ctrl+h/j/k/l reach the
// Window from the page only in Vim's Normal mode: Insert mode, and the Text editor mode, type them.
var windowKeys = []Key{
	{Seq: "Space", Show: "Space", Opens: leader},
	{Seq: "Ctrl+Space", Show: "Ctrl+Space", Opens: leader}, // the leader in every editor mode
	{Seq: "Ctrl+H", Show: "Ctrl+h/j/k/l", Run: `App.movePane("h")`},
	{Seq: "Ctrl+J", Show: "Ctrl+h/j/k/l", Run: `App.movePane("j")`},
	{Seq: "Ctrl+K", Show: "Ctrl+h/j/k/l", Run: `App.movePane("k")`},
	{Seq: "Ctrl+L", Show: "Ctrl+h/j/k/l", Run: `App.movePane("l")`},
	{Seq: "Ctrl+S", Show: "Ctrl+S", Run: "App.save()"},
	{Seq: "Ctrl+O", Show: "Ctrl+O", Run: "App.openPicker()"},
	{Seq: "Ctrl+N", Show: "Ctrl+N", Run: "App.newFile()"},
	// as vim's search; the Vim keyset takes Ctrl+F and Ctrl+B to page
	{Seq: "Ctrl+G", Show: "Ctrl+G", Run: "App.openSearch()"},
	// the command line in every editor mode, on : with Ctrl: the Text mode types :; a terminal sends
	// Ctrl+; only with kitty's keyboard protocol, so Search › Command line… is always there
	{Seq: "Ctrl+;", Show: "Ctrl+;", Run: "App.openCommandLine()"},
	// / finds in the pane with the keyboard; n and N find again (find.go)
	{Seq: "/", Show: "/", Run: "App.openFind()"},
	{Seq: "n", Show: "n and N", Run: "App.findNext()"},
	{Seq: "Shift+N", Show: "n and N", Run: "App.findPrevious()"},
	// d in the Relations drawer shows the second ring, or hides it (relations.go); elsewhere nothing
	{Seq: "d", Show: "d in the relations", Run: "App.relationsDepth()"},
	{Seq: "Ctrl+W", Show: "Ctrl+W", Run: "App.pickWorkspace()"},
	{Seq: "Ctrl+Q", Show: "Ctrl+Q", Run: "App.quit()"},
	{Seq: "F1", Show: "F1", Run: "help.open()"},
	// the window's zoom (fonts.go); nothing in a terminal
	{Seq: "Ctrl+=", Show: "Ctrl+= Ctrl+- Ctrl+0", Run: `App.zoomWindow("in")`},
	{Seq: "Ctrl+-", Show: "Ctrl+= Ctrl+- Ctrl+0", Run: `App.zoomWindow("out")`},
	{Seq: "Ctrl+0", Show: "Ctrl+= Ctrl+- Ctrl+0", Run: "App.setZoom(100)"},
	// the Vim keys' card, in Normal mode (the Text mode types it); ? again closes it
	{Seq: "?", Show: "?", Run: "App.toggleVimKeys()"},
}

// leaderKeys are SPC's card's, in the card's reading order: left column, right column, row by row.
var leaderKeys = onCard(leader, []Key{
	{Seq: "/", Also: []string{"Space"}, Show: "/", Does: "search (or SPC again)", Run: "App.openSearch()"},
	{Seq: "e", Show: "e", Does: "the explorer", Run: "App.toggleExplorer()"},
	{Seq: "o", Show: "o", Does: "open a document", Run: "App.openPicker()"},
	{Seq: "l", Show: "l", Does: "the relations", Run: "App.toggleLinks()"},
	{Seq: "f", Show: "f", Does: "open a file anywhere", Run: "App.openFileDialog()"},
	{Seq: "j", Show: "j", Does: "related files, by digit", Run: "App.openJumpCard()"},
	{Seq: "n", Show: "n", Does: "new file", Run: "App.newFile()"},
	{Seq: "b", Show: "b", Does: "back", Run: "App.goBack()"},
	{Seq: "s", Show: "s", Does: "save", Run: "App.save()"},
	{Seq: "`", Show: "`", Does: "the terminal", Run: "App.toggleTerminal()"},
	{Seq: "r", Show: "r", Does: "recent files", Run: "App.openRecent()"},
	{Seq: "t", Show: "t", Does: "the status line", Run: "App.toggleStatusLine()"},
	{Seq: "w", Show: "w", Does: "switch workspace", Run: "App.pickWorkspace()"},
	{Seq: "g", Show: "g", Does: "AI agent", Run: "App.toggleAgent()"},
	{Seq: "Shift+W", Show: "W", Does: "manage workspaces", Run: "App.manageWorkspaces()"},
	{Seq: "m", Show: "m", Does: "show/focus or hide the menu", Run: "App.leaderMenuBar()"},
	{Seq: "?", Show: "?", Does: "help", Run: "help.open()"},
	{Seq: "k", Show: "k", Does: "the editor mode (Vim, Text)", Run: "App.toggleKeymap()"},
	{Seq: "h", Show: "h", Does: "notifications", Run: "App.openNotices()"},
	{Seq: ",", Show: ",", Does: "editor preferences", Run: "App.openPrefs()"},
	{Seq: "Shift+O", Show: "O", Does: "the file's outline", Run: "App.openOutline()"},
	{Seq: "a", Show: "a", Does: "search models", Run: "App.openAIModels()"},
	{Seq: "d", Show: "d", Does: "view the diagram", Run: "App.previewDiagram()"},
	{Seq: "Shift+A", Show: "A", Does: "about", Run: "about.open()"},
	{Seq: "Shift+G", Show: "G", Does: "agent profiles", Run: "App.openAgentProfiles()"},
	{Seq: "Shift+Q", Show: "Q", Does: "quit", Run: "App.quit()"},
	{Seq: "p", Show: "p", Does: "plugin commands", Run: "pluginKeys.open()"},
	{Seq: "Shift+H", Show: "H", Does: "preview HTML", Run: "App.previewHTML()"},
	{Seq: ".", Show: ".", Does: "the right-click menu", Run: "App.openContextMenu()"},
	{Seq: "Shift+L", Show: "L", Does: "arrange a panel", Run: "App.arrangePanel()"},
	{Seq: "Shift+T", Show: "T", Does: "the tutorial", Run: "App.openTutorial()"},
	{Seq: "x", Show: "x", Does: "todos", Run: "App.openTodos()"},
})

// onCard is keys, each on card.
func onCard(card string, keys []Key) []Key {
	for i := range keys {
		keys[i].Card = card
	}
	return keys
}

// keyset is the catalog: every card, and every key, the host's first and then each feature's.
type keyset struct {
	cards []Card
	keys  []Key
}

// keyCatalog is the catalog with every feature's keys and cards after the host's.
func keyCatalog(fs []Feature) keyset {
	k := keyset{cards: slices.Clone(cards), keys: slices.Concat(windowKeys, leaderKeys)}
	for _, f := range fs {
		if kf, ok := f.(Keyed); ok {
			k.cards, k.keys = append(k.cards, kf.Cards()...), append(k.keys, kf.Keys()...)
		}
	}
	return k
}

// on are the keys on card ("" the Window's).
func (k keyset) on(card string) []Key {
	var out []Key
	for _, key := range k.keys {
		if key.Card == card {
			out = append(out, key)
		}
	}
	return out
}

// card is the card of id.
func (k keyset) card(id string) (Card, bool) {
	i := slices.IndexFunc(k.cards, func(c Card) bool { return c.ID == id })
	if i < 0 {
		return Card{}, false
	}
	return k.cards[i], true
}

// check refuses a sequence two keys of one place take, a key on or opening a card that is not
// declared, a card declared twice, and a card nothing opens.
func (k keyset) check() error {
	var errs []error
	opened := map[string]bool{}
	for i, c := range k.cards {
		if slices.ContainsFunc(k.cards[:i], func(d Card) bool { return d.ID == c.ID || d.View == c.View }) {
			errs = append(errs, fmt.Errorf("tui: the card %s is declared twice", c.ID))
		}
	}
	for _, key := range k.keys {
		if _, ok := k.card(key.Card); key.Card != "" && !ok {
			errs = append(errs, fmt.Errorf("tui: %s is on the card %s, which is not declared", key.Seq, key.Card))
		}
		if key.Opens != "" {
			if _, ok := k.card(key.Opens); !ok {
				errs = append(errs, fmt.Errorf("tui: %s opens the card %s, which is not declared", key.Seq, key.Opens))
			}
			opened[key.Opens] = true
		}
	}
	for _, c := range k.cards {
		if !opened[c.ID] {
			errs = append(errs, fmt.Errorf("tui: no key opens the card %s", c.ID))
		}
	}
	for _, place := range append([]string{""}, cardIDs(k.cards)...) {
		seen := map[string]bool{}
		for _, key := range k.on(place) {
			for _, s := range append([]string{key.Seq}, key.Also...) {
				if seen[s] {
					errs = append(errs, fmt.Errorf("tui: %s %s is taken twice", placeName(place), s))
				}
				seen[s] = true
			}
		}
	}
	return errors.Join(errs...)
}

func cardIDs(cs []Card) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.ID
	}
	return out
}

// placeName is how a refusal names where a key is answered.
func placeName(card string) string {
	if card == "" {
		return "the Window's"
	}
	return "the card " + card + "'s"
}

// cardText is a card's body: two commands a row, the second column aligned, as the catalog orders
// them.
func cardText(keys []Key) string {
	var listed []string
	for _, k := range keys {
		if k.Does != "" {
			listed = append(listed, k.Show+"  "+k.Does)
		}
	}
	width := 0
	for i := 0; i < len(listed); i += 2 {
		width = max(width, utf8.RuneCountInString(listed[i]))
	}
	var b strings.Builder
	for i := 0; i < len(listed); i += 2 {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(listed[i])
		if i+1 < len(listed) {
			b.WriteString(strings.Repeat(" ", width+2-utf8.RuneCountInString(listed[i])))
			b.WriteString(listed[i+1])
		}
	}
	return b.String()
}

// textSource is the App.* source a card's body is: App.leaderText for SPC's.
func textSource(card string) string { return "App." + card + "Text" }

// cardState is every card's body, as the document reads it.
func (k keyset) cardState() map[string]any {
	m := map[string]any{}
	for _, c := range k.cards {
		m[textSource(c.ID)] = cardText(k.on(c.ID))
	}
	return m
}

// helpCards is every card as help lists it, under its heading.
func (k keyset) helpCards() string {
	var parts []string
	for _, c := range k.cards {
		parts = append(parts, c.Help+"\n  "+strings.ReplaceAll(cardText(k.on(c.ID)), "\n", "\n  "))
	}
	return strings.Join(parts, "\n\n")
}

// shortcuts are the QML Shortcuts for keys, indented: each closes the card it is on first, when
// close names it; a key that opens a card opens it.
func shortcuts(keys []Key, indent, close string) string {
	var b strings.Builder
	for _, k := range keys {
		run := k.Run
		if k.Opens != "" {
			run = k.Opens + ".open()"
		}
		if close != "" {
			run = "{ " + close + ".close(); " + run + " }"
		}
		for _, s := range append([]string{k.Seq}, k.Also...) {
			fmt.Fprintf(&b, "%sShortcut { sequence: %q; onActivated: %s }\n", indent, s, run)
		}
	}
	return b.String()
}

// The generated blocks of main.qml.
const (
	keysBegin  = "    // ---- the Window's keys: generated from tui/keys.go (go test ./tui -run TestKeysAreGenerated -update) ----\n"
	keysEnd    = "    // ---- end of the generated keys ----\n"
	cardsBegin = "    // ---- the cards of keys: generated from tui/keys.go (go test ./tui -run TestKeysAreGenerated -update) ----\n"
	cardsEnd   = "    // ---- end of the generated cards ----\n"
)

// cardQML is views/<View>.qml for card c and its keys.
func cardQML(c Card, keys []Key) string {
	back := ""
	if c.Back != "" {
		back = fmt.Sprintf("    Shortcut { sequence: \"Backspace\"; onActivated: { %s.close(); %s.open() } }\n", c.ID, c.Back)
	}
	help := "a key runs its command · q or Esc closes"
	if c.Back != "" {
		help = "a key runs its command · Backspace goes back · q or Esc closes"
	}
	return `// ` + c.View + `.qml — a card of keys, as AutoDB's: every command here, by key. GENERATED from
// tui/keys.go (go test ./tui -run TestKeysAreGenerated -update): edit the catalog there, not this
// file. The rows are TEXT, not a list: a list takes the keyboard and eats letters that are keys
// here. With nothing focusable in it, every key reaches the dialog's Shortcuts, one a command, and
// each closes the card before its command runs, so what the command opens is not under it. Esc
// closes it.
Dialog {
    id: ` + c.ID + `
    closeOnQ: true
    title: "` + c.Title + `"
    dim: false
    helpText: "` + help + `"
    Text { text: ` + textSource(c.ID) + `; wrapMode: Tui.WordWrap } // WordWrap keeps its lines: NoWrap is one
` + back + shortcuts(keys, "    ", c.ID) + "}\n"
}

// withGenerated is main.qml with its generated blocks replaced by the catalog's.
func (k keyset) withGenerated(mainQML string) (string, error) {
	var decls strings.Builder
	for _, c := range k.cards {
		fmt.Fprintf(&decls, "    %s { id: %s }\n", c.View, c.ID)
	}
	out, err := replaceBlock(mainQML, keysBegin, keysEnd, shortcuts(k.on(""), "    ", ""))
	if err != nil {
		return "", err
	}
	return replaceBlock(out, cardsBegin, cardsEnd, decls.String())
}

// replaceBlock is s with what lies between begin and end replaced by body.
func replaceBlock(s, begin, end, body string) (string, error) {
	a := strings.Index(s, begin)
	b := strings.Index(s, end)
	if a < 0 || b < a {
		return "", fmt.Errorf("tui: main.qml lacks the markers %q", strings.TrimSpace(begin))
	}
	return s[:a+len(begin)] + body + s[b:], nil
}
