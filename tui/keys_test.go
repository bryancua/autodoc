package tui

import (
	"flag"
	"os"
	"strings"
	"testing"
)

var updateKeys = flag.Bool("update", false, "rewrite the files generated from the key catalog (keys.go)")

// TestKeysAreGenerated: main.qml's Window keys and views/Leader.qml are the catalog's, the
// features' keys included; -update rewrites them.
func TestKeysAreGenerated(t *testing.T) {
	h := newHost(NewSession("unused", nil), Options{})
	window, leader := keyCatalog(h.features())
	mainQML, err := os.ReadFile("qml/main.qml")
	if err != nil {
		t.Fatal(err)
	}
	wantMain, err := withWindowKeys(string(mainQML), window)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{"qml/main.qml": wantMain, "qml/views/Leader.qml": leaderQML(leader)} {
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

// Every Window key is named in the help, as help names it; every leader key is on the card, and
// the card is in the help.
func TestEveryKeyIsInTheHelp(t *testing.T) {
	help := helpText(leaderKeys)
	for _, k := range windowKeys {
		if !strings.Contains(help, k.Show) {
			t.Errorf("the Window's %s (%s) is not in the help", k.Seq, k.Show)
		}
	}
	card := cardText(leaderKeys)
	for _, k := range leaderKeys {
		if !strings.Contains(card, k.Show+"  "+k.Does) {
			t.Errorf("SPC %s is not on the card", k.Show)
		}
	}
	if !strings.Contains(help, strings.ReplaceAll(card, "\n", "\n  ")) {
		t.Error("the help does not hold the leader card")
	}
}

// A sequence two keys of one place take is refused when the registry is built.
func TestTheRegistryRefusesAKeyTakenTwice(t *testing.T) {
	twice := &keyedFake{fake: fake{name: "b"}, leader: []Key{{Seq: "x", Show: "x", Does: "mine", Run: "App.x()"}}}
	if _, err := newRegistry(twice); err == nil || !strings.Contains(err.Error(), "the leader card's x is taken twice") {
		t.Errorf("SPC x twice: %v", err)
	}
	also := &keyedFake{fake: fake{name: "b"}, window: []Key{{Seq: "F2", Also: []string{"Ctrl+S"}, Run: "App.x()"}}}
	if _, err := newRegistry(also); err == nil || !strings.Contains(err.Error(), "the Window's Ctrl+S is taken twice") {
		t.Errorf("Ctrl+S twice: %v", err)
	}
	if _, err := newRegistry(&keyedFake{fake: fake{name: "b"}, leader: []Key{{Seq: "Shift+X", Show: "X", Does: "mine", Run: "App.x()"}}}); err != nil {
		t.Errorf("a key of a feature's own: %v", err)
	}
}

type keyedFake struct {
	fake
	window, leader []Key
}

func (f *keyedFake) WindowKeys() []Key { return f.window }
func (f *keyedFake) LeaderKeys() []Key { return f.leader }
