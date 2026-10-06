package script

import "testing"

func TestValidate(t *testing.T) {
	t.Parallel()
	notice := "taken from the `abs0/wargames` project"
	good := []Ls{Recon("HELLO."), Orig("WHICH GAME?"), Tag(ABS0, "#45"), {{Text: "OK", Prov: Film}}}
	if errs := Validate(good, notice); len(errs) != 0 {
		t.Errorf("valid lines rejected: %v", errs)
	}
	if errs := Validate([]Ls{Tag(ABS0, "#45")}, "see https://github.com/abs0/wargames"); len(errs) != 0 {
		t.Errorf("a GitHub URL credits the repo: %v", errs)
	}
	bad := []Ls{
		{{Text: "NO TAG"}},
		{{Text: "X", Prov: "prompt"}},
		Tag("third-party:elfuska/wargames@1:x", "UNCREDITED"),
		Orig("lower case"),
		Tag("third-party:a", "A SUBSTRING OF THE NOTICE"),
		Tag("third-party:abs0/wargames", "NO COMMIT OR PATH"),
		Tag("third-party:abs0/wargames@010ed92", "NO PATH"),
		Tag("third-party:wargames@010ed92:wargames.sh", "NO OWNER"),
	}
	if errs := Validate(bad, notice); len(errs) != len(bad) {
		t.Errorf("want %d errors, got %d: %v", len(bad), len(errs), errs)
	}
}
