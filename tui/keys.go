package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// KEYS — every key the Window answers and every key on the leader card (SPC), declared once. From
// this catalog come main.qml's Window shortcuts, the leader card's Shortcuts (views/Leader.qml),
// the card's text, and the card as help shows it. The QML is generated: TestKeysAreGenerated
// fails when a file differs from the catalog, and `go test ./tui -run TestKeysAreGenerated
// -update` rewrites them. A feature adds keys with the Keyed capability.

// Key is one key.
type Key struct {
	Seq  string   // the Shortcut's sequence: "Ctrl+G", "Shift+W", "`"
	Also []string // more sequences that do the same, shown nowhere: SPC SPC beside SPC /
	Show string   // how the card names it ("W"), or how help names a Window key ("Ctrl+h/j/k/l")
	Does string   // what it does, as the card says it; "" for a key the card does not list
	Run  string   // the QML it runs: "App.openSearch()", "help.open()"
}

// Keyed is a feature with keys of its own: on the leader card, and the Window's.
type Keyed interface {
	LeaderKeys() []Key
	WindowKeys() []Key
}

// windowKeys fire whichever widget has focus, when it leaves them. Space and Ctrl+h/j/k/l reach the
// Window from the page only in Vim's Normal mode: Insert mode, and the Text editor mode, type them.
var windowKeys = []Key{
	{Seq: "Space", Show: "Space", Run: "leader.open()"},
	{Seq: "Ctrl+Space", Show: "Ctrl+Space", Run: "leader.open()"}, // the leader in every editor mode
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

// leaderKeys are the leader card's, in the card's reading order: left column, right column, row by
// row. Each closes the card before its command runs, so what the command opens is not under it.
var leaderKeys = []Key{
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
}

// keyCatalog is the catalog with every feature's keys after the host's.
func keyCatalog(fs []Feature) (window, leader []Key) {
	window, leader = append([]Key(nil), windowKeys...), append([]Key(nil), leaderKeys...)
	for _, f := range fs {
		if k, ok := f.(Keyed); ok {
			window, leader = append(window, k.WindowKeys()...), append(leader, k.LeaderKeys()...)
		}
	}
	return window, leader
}

// checkKeys refuses a sequence two keys of one place both take.
func checkKeys(place string, keys []Key) error {
	seen := map[string]bool{}
	for _, k := range keys {
		for _, s := range append([]string{k.Seq}, k.Also...) {
			if seen[s] {
				return fmt.Errorf("tui: %s %s is taken twice", place, s)
			}
			seen[s] = true
		}
	}
	return nil
}

// cardText is the leader card's body: two commands a row, the second column aligned, as the
// catalog orders them.
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

// shortcuts are the QML Shortcuts for keys, indented: each closes the card first when close names it.
func shortcuts(keys []Key, indent, close string) string {
	var b strings.Builder
	for _, k := range keys {
		run := k.Run
		if close != "" {
			run = "{ " + close + ".close(); " + k.Run + " }"
		}
		for _, s := range append([]string{k.Seq}, k.Also...) {
			fmt.Fprintf(&b, "%sShortcut { sequence: %q; onActivated: %s }\n", indent, s, run)
		}
	}
	return b.String()
}

// The generated files.
const (
	keysBegin = "    // ---- the Window's keys: generated from tui/keys.go (go test ./tui -run TestKeysAreGenerated -update) ----\n"
	keysEnd   = "    // ---- end of the generated keys ----\n"
)

// leaderQML is views/Leader.qml for keys.
func leaderQML(keys []Key) string {
	return `// Leader.qml — SPC's card, as AutoDB's: every command here, by key. GENERATED from tui/keys.go
// (go test ./tui -run TestKeysAreGenerated -update): edit the catalog there, not this file. The rows
// are TEXT, not a list: a list takes the keyboard and eats letters that are keys here. With nothing
// focusable in it, every key reaches the dialog's Shortcuts, one a command, and each closes the card
// before its command runs, so what the command opens is not under it. Esc closes it.
Dialog {
    id: leader
    closeOnQ: true
    title: "SPC — commands"
    dim: false
    helpText: "a key runs its command · q or Esc closes"
    Text { text: App.leaderText; wrapMode: Tui.WordWrap } // WordWrap keeps its lines: NoWrap is one
` + shortcuts(keys, "    ", "leader") + "}\n"
}

// withWindowKeys is main.qml with its generated block replaced by keys'.
func withWindowKeys(mainQML string, keys []Key) (string, error) {
	a := strings.Index(mainQML, keysBegin)
	b := strings.Index(mainQML, keysEnd)
	if a < 0 || b < a {
		return "", fmt.Errorf("tui: main.qml lacks the generated keys' markers")
	}
	return mainQML[:a+len(keysBegin)] + shortcuts(keys, "    ", "") + mainQML[b:], nil
}
