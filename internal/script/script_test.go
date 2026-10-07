package script

import "testing"

func TestValidate(t *testing.T) {
	t.Parallel()
	notice := "taken from the `abs0/wargames` project"
	good := []Ls{Recon("HELLO."), Orig("WHICH GAME?"), Tag(ABS0, "#45"), {{Text: "OK", Prov: Film}}, User(Reconstructed, "Hello.")}
	if errs := Validate(good, notice); len(errs) != 0 {
		t.Errorf("valid lines rejected: %v", errs)
	}
	if errs := Validate([]Ls{Tag(ABS0, "#45")}, "see https://github.com/abs0/wargames"); len(errs) != 0 {
		t.Errorf("a GitHub URL credits the repo: %v", errs)
	}
	// Art from a web page is kept as drawn, lower case and all, once NOTICE.md links its page.
	art := []Ls{Tag(MatthewThomasMap, "  \\_.:--.   mt-2_")}
	if errs := Validate(art, "from <https://asciiart.website/art/3719>, used under"); len(errs) != 0 {
		t.Errorf("credited web art rejected: %v", errs)
	}
	if errs := Validate(art, "see asciiart.website for more"); len(errs) != 1 {
		t.Errorf("uncredited web art: want 1 error, got %v", errs)
	}
	if errs := Validate(art, "https://asciiart.website/art/37190"); len(errs) != 1 {
		t.Errorf("a longer URL does not credit the page: %v", errs)
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
		Tag(ABS0, "repository text is still in capitals"), // the exemption is for web art only
		User("", "Typed lines need a tag too"),
	}
	if errs := Validate(bad, notice); len(errs) != len(bad) {
		t.Errorf("want %d errors, got %d: %v", len(bad), len(errs), errs)
	}
}
