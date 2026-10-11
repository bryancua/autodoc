package tui

import (
	"flag"
	"os"
	"strings"
	"testing"
)

var updateKeys = flag.Bool("update", false, "rewrite the files generated from the key catalog (keys.go)")

// TestKeysAreGenerated: main.qml's Window keys and its cards' declarations, and each card's
// views/<View>.qml, are the catalog's, the features' keys and cards included; -update rewrites them.
func TestKeysAreGenerated(t *testing.T) {
	h := newHost(NewSession("unused", nil), Options{})
	k := keyCatalog(h.features())
	if err := k.check(mainIDs); err != nil {
		t.Fatal(err)
	}
	mainQML, err := os.ReadFile("qml/main.qml")
	if err != nil {
		t.Fatal(err)
	}
	wantMain, err := k.withGenerated(string(mainQML))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"qml/main.qml": wantMain}
	for _, c := range k.cards {
		files["qml/views/"+c.View+".qml"] = cardQML(c, k.on(c.ID))
	}
	for path, want := range files {
		if *updateKeys {
			if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s is not the key catalog's: go test ./tui -run TestKeysAreGenerated -update", path)
		}
	}
}

// Every Window key is named in the help, as help names it; every card key is on its card, and
// every card is in the help under its heading.
func TestEveryKeyIsInTheHelp(t *testing.T) {
	k := keyCatalog(nil)
	help := helpText(k)
	for _, key := range k.on("") {
		if !strings.Contains(help, key.Show) {
			t.Errorf("the Window's %s (%s) is not in the help", key.Seq, key.Show)
		}
	}
	for _, c := range k.cards {
		text := cardText(k.on(c.ID))
		for _, key := range k.on(c.ID) {
			if !strings.Contains(text, key.Show+"  "+key.Does) {
				t.Errorf("%s is not on the card %s", key.Show, c.ID)
			}
		}
		if !strings.Contains(help, c.Help+"\n  "+strings.ReplaceAll(text, "\n", "\n  ")) {
			t.Errorf("the help does not hold the card %s", c.ID)
		}
	}
}

// A card a key opens: its own file and text, its keys closing it before they run, Backspace back
// to the card it came from, and its keys in the help.
func TestACardOpensFromAKey(t *testing.T) {
	agents := &keyedFake{fake: fake{name: "agents"},
		cards: []Card{{ID: "agentsCard", View: "AgentsCard", Title: "SPC a — agents", Back: leader, Help: "SPC a — AGENTS"}},
		keys: []Key{
			{Card: leader, Seq: "Shift+B", Show: "B", Does: "agents…", Opens: "agentsCard"},
			{Card: "agentsCard", Seq: "b", Show: "b", Does: "send the buffer", Run: `App.runCommand("agent.buffer")`},
			{Card: "agentsCard", Seq: "f", Show: "f", Does: "send the selection", Run: `App.runCommand("agent.selection")`},
		}}
	k := keyCatalog([]Feature{agents})
	if err := k.check(mainIDs); err != nil {
		t.Fatal(err)
	}
	c, _ := k.card("agentsCard")
	qml := cardQML(c, k.on("agentsCard"))
	for _, want := range []string{
		"id: agentsCard",
		`title: "SPC a — agents"`,
		"text: App.agentsCardText",
		`Shortcut { sequence: "Backspace"; onActivated: { agentsCard.close(); leader.open() } }`,
		`Shortcut { sequence: "b"; onActivated: { agentsCard.close(); App.runCommand("agent.buffer") } }`,
		"Backspace goes back",
	} {
		if !strings.Contains(qml, want) {
			t.Errorf("the card's QML lacks %q:\n%s", want, qml)
		}
	}
	if lq, _ := k.card(leader); !strings.Contains(cardQML(lq, k.on(leader)), `Shortcut { sequence: "Shift+B"; onActivated: { leader.close(); agentsCard.open() } }`) {
		t.Error("SPC B does not close SPC's card and open the agents'")
	}
	if got := k.cardState()["App.agentsCardText"]; got != "b  send the buffer  f  send the selection" {
		t.Errorf("the card's text: %q", got)
	}
	main, err := k.withGenerated(keysBegin + keysEnd + cardsBegin + cardsEnd)
	if err != nil || !strings.Contains(main, "    AgentsCard { id: agentsCard }\n") || !strings.Contains(main, "    Leader { id: leader }\n") {
		t.Errorf("the cards' declarations: %v\n%s", err, main)
	}
	if !strings.Contains(helpText(k), "SPC a — AGENTS\n  b  send the buffer") {
		t.Error("the help lacks the agents card")
	}
	if _, err := keyCatalog(nil).withGenerated("no markers"); err == nil {
		t.Error("a main.qml without the markers was rewritten")
	}
	// a title is a QML string, quoted whatever it holds
	if q := cardQML(Card{ID: "x", View: "X", Title: `say "hi" \ bye`}, nil); !strings.Contains(q, `title: "say \"hi\" \\ bye"`) {
		t.Errorf("the title is not quoted:\n%s", q)
	}
}

// The registry refuses a sequence two keys of one place take, a card no key opens or declared twice,
// and a key on or opening a card nobody declared.
func TestTheRegistryRefusesKeysThatDoNotFit(t *testing.T) {
	for want, f := range map[string]*keyedFake{
		"the card leader's x is taken twice": {keys: []Key{{Card: leader, Seq: "x", Show: "x", Does: "mine", Run: "App.x()"}}},
		"the Window's Ctrl+S is taken twice": {keys: []Key{{Seq: "F2", Also: []string{"Ctrl+S"}, Run: "App.x()"}}},
		"no key opens the card lonely":       {cards: []Card{{ID: "lonely", View: "Lonely"}}},
		"the card leader is declared twice":  {cards: []Card{{ID: leader, View: "Other"}}, keys: []Key{{Seq: "F3", Opens: leader}}},
		"which is not declared":              {keys: []Key{{Card: "nowhere", Seq: "z", Run: "App.x()"}}},
		"opens the card gone, which is not":  {keys: []Key{{Card: leader, Seq: "Shift+Z", Opens: "gone"}}},
		// what every card answers itself: q and Escape close it, Backspace goes back
		"the card leader's q is taken twice":      {keys: []Key{{Card: leader, Seq: "q", Run: "App.x()"}}},
		"the card leader's Escape is taken twice": {keys: []Key{{Card: leader, Seq: "F4", Also: []string{"Escape"}, Run: "App.x()"}}},
		"the card subCard's Backspace is taken twice": {cards: []Card{{ID: "subCard", View: "SubCard", Back: leader}},
			keys: []Key{{Card: leader, Seq: "Shift+Y", Opens: "subCard"}, {Card: "subCard", Seq: "Backspace", Run: "App.x()"}}},
		// a card's ID and View are QML names, and the ID is no other of main.qml's
		"needs a lower-camel ID":           {cards: []Card{{ID: "agents-card", View: "AgentsCard"}}, keys: []Key{{Seq: "F5", Opens: "agents-card"}}},
		"and an upper-camel View":          {cards: []Card{{ID: "agentsCard", View: "agentsCard"}}, keys: []Key{{Seq: "F5", Opens: "agentsCard"}}},
		"takes an id main.qml already has": {cards: []Card{{ID: "help", View: "HelpCard"}}, keys: []Key{{Seq: "F5", Opens: "help"}}},
		// Backspace goes where the card came from
		"whose Backspace goes back to the card leader's": {cards: []Card{{ID: "subCard", View: "SubCard", Back: leader}},
			keys: []Key{{Seq: "F6", Opens: "subCard"}}},
		"whose Backspace goes back to the Window's": {cards: []Card{{ID: "subCard", View: "SubCard"}},
			keys: []Key{{Card: leader, Seq: "Shift+Y", Opens: "subCard"}}},
	} {
		f.fake = fake{name: "b"}
		if _, err := newRegistry(f); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("want %q: %v", want, err)
		}
	}
	if _, err := newRegistry(&keyedFake{fake: fake{name: "b"}, keys: []Key{{Card: leader, Seq: "Shift+X", Show: "X", Does: "mine", Run: "App.x()"}}}); err != nil {
		t.Errorf("a key of a feature's own: %v", err)
	}
}

type keyedFake struct {
	fake
	keys  []Key
	cards []Card
}

func (f *keyedFake) Keys() []Key   { return f.keys }
func (f *keyedFake) Cards() []Card { return f.cards }
