package script

import "testing"

func TestValidate(t *testing.T) {
	t.Parallel()
	notice := "credited: abs0/wargames"
	good := []Ls{Recon("HELLO."), Orig("WHICH GAME?"), Tag(ABS0, "#45"), {{Text: "OK", Prov: Film}}}
	if errs := Validate(good, notice); len(errs) != 0 {
		t.Errorf("valid lines rejected: %v", errs)
	}
	bad := []Ls{
		{{Text: "NO TAG"}},
		{{Text: "X", Prov: "prompt"}},
		Tag("third-party:elfuska/wargames@1:x", "UNCREDITED"),
		Orig("lower case"),
	}
	if errs := Validate(bad, notice); len(errs) != 4 {
		t.Errorf("want 4 errors, got %d: %v", len(errs), errs)
	}
}
