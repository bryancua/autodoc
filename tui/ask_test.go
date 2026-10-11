package tui

import "testing"

// The card asks one question at a time: Yes runs its OnYes once, No and a newer question drop it,
// and only the question still asked can have its text rewritten.
func TestAskRunsYesOnceAndOnlyForTheQuestionAsked(t *testing.T) {
	s := newScripted("kb")
	a := newAsker(s)
	if st := a.State(); st["App.askYes"] != "&Yes" || st["App.askNo"] != "&No" {
		t.Fatalf("first labels %v", st)
	}
	ran := 0
	asked := a.ask(Question{Title: "remove the thing?", Text: "counting…", Yes: "&Yes, remove it", OnYes: func() { ran++ }})
	if s.opened[len(s.opened)-1] != "confirm" || s.set["App.askTitle"] != "remove the thing?" ||
		s.set["App.askYes"] != "&Yes, remove it" || s.set["App.askNo"] != "&No" {
		t.Fatalf("the card: opened %v, set %v", s.opened, s.set)
	}
	asked.Say("3 things go")
	if s.set["App.askText"] != "3 things go" {
		t.Errorf("Say while asked: %v", s.set["App.askText"])
	}
	a.Commands()["App.askAccepted"].Run(nil)
	a.Commands()["App.askAccepted"].Run(nil)
	if ran != 1 {
		t.Errorf("Yes ran OnYes %d times, want once", ran)
	}
	asked.Say("late")
	if s.set["App.askText"] != "3 things go" {
		t.Errorf("Say after the answer rewrote the text: %v", s.set["App.askText"])
	}

	first := a.ask(Question{Title: "one?", OnYes: func() { ran += 10 }})
	a.ask(Question{Title: "two?", OnYes: func() { ran += 100 }})
	first.Say("stale")
	if s.set["App.askText"] == "stale" {
		t.Error("an older question's Say rewrote the newer one")
	}
	a.Commands()["App.askAccepted"].Run(nil)
	if ran != 101 {
		t.Errorf("Yes on the newer question: ran %d, want 101 (only the newer one's OnYes)", ran)
	}

	a.ask(Question{Title: "three?", OnYes: func() { ran += 1000 }})
	a.Commands()["App.askRejected"].Run(nil)
	a.Commands()["App.askAccepted"].Run(nil)
	if ran != 101 {
		t.Errorf("No, then a stray accept: ran %d, want 101", ran)
	}
	Asked{}.Say("no-op") // a zero Asked, as a test's Core returns, says nothing
}
