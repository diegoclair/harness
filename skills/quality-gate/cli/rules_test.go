package main

import (
	"strings"
	"testing"
)

// The regression this encodes: an import resolves to a package directory, so a
// layer pattern that only matched files left every ARC rule silently mute.
func TestGlobMatchesPackageDirectories(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"data/**", "data", true},
		{"data/**", "data/first.go", true},
		{"data/**", "database/first.go", false},
		{"internal/*/domain/**", "internal/provider/domain", true},
		{"internal/*/domain/**", "internal/provider/domain/entity/x.go", true},
		{"internal/*/domain/**", "internal/provider/service/x.go", false},
		{"**/mocks/**", "internal/provider/mocks/x.go", true},
		{"**/mocks/**", "mocks/x.go", true},
		{"**/*.gen.go", "internal/x.gen.go", true},
		{"**/*.gen.go", "internal/x.go", false},
	}
	for _, c := range cases {
		if got := globMatch(c.pattern, c.path); got != c.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestConstraintMarkersSeparateDescriptionFromConstraint(t *testing.T) {
	described := []string{
		"firstname addresses the provider the way a client speaks about them.",
		"materializeduntil is the last date already turned into bookings.",
	}
	constrained := []string{
		"counts the first try. values below 1 fall back to the default.",
		"the psp returns it once, at creation; unrecoverable if lost.",
		"nullable because system-seeded categories carry no owner.",
		"cents, not reais.",
		"the legal name, not the display one.",
	}
	for _, s := range described {
		if hasConstraint(s) {
			t.Errorf("%q should read as a description", s)
		}
	}
	for _, s := range constrained {
		if !hasConstraint(s) {
			t.Errorf("%q carries a constraint and should pass", s)
		}
	}
}

// "used to" and "no longer" also form ordinary English about the present. A
// history marker that fires on those trains the reader to ignore the rule.
func TestHistoryMarkersIgnorePresentTenseEnglish(t *testing.T) {
	present := []string{
		"a label+color tag used to group cost entries.",
		"the cdn custom domain used to serve reads.",
		"records a failure that can no longer change the http status.",
		// Runtime state, not code history — six of nine findings were these.
		"fresh url params win over a previously stored bag.",
		"so a previously expired session does not survive the login.",
		"the rule's own date and moved from there (spec 00036).",
		"the stamp is only the net for windows changed from outside.",
	}
	history := []string{
		"previously this was a pointer.",
		"we used to send this synchronously.",
		"renamed: it was renamed in the split.",
	}
	for _, s := range present {
		if _, ok := containsAny(s, historyMarkers); ok {
			t.Errorf("%q is present tense, not history", s)
		}
	}
	for _, s := range history {
		if _, ok := containsAny(s, historyMarkers); !ok {
			t.Errorf("%q is history and should be caught", s)
		}
	}
}

func TestRestatesName(t *testing.T) {
	cases := []struct {
		target, text string
		want         bool
	}{
		{"GetUser", "GetUser gets the user", true},
		{"helper", "helper returns the helper", true},
		{"Charge", "Charge debits the card and refuses below the minimum", false},
	}
	for _, c := range cases {
		got := restatesName(Comment{Target: c.target, Text: c.text, Line: 1, EndLine: 1})
		if got != c.want {
			t.Errorf("restatesName(%q, %q) = %v, want %v", c.target, c.text, got, c.want)
		}
	}
}

func TestSectionLabelIsNotADescription(t *testing.T) {
	label := Comment{Target: "ErrInstagramURLInvalid", Text: "Instagram Showcase.", Line: 1, EndLine: 1}
	if !isSectionLabel(label) {
		t.Error("a short noun phrase that never mentions the declaration is a section label")
	}
	description := Comment{Target: "MaterializedUntil", Text: "MaterializedUntil is the last date.", Line: 1, EndLine: 1}
	if isSectionLabel(description) {
		t.Error("a comment naming its declaration is describing it, not labelling a section")
	}
}

func TestNotEnglishNeedsMoreThanOneLooseWord(t *testing.T) {
	if _, ok := notEnglish("the parameters we pass to the handler"); ok {
		t.Error("plain English must not be flagged")
	}
	if _, ok := notEnglish("esse comentário está em português"); !ok {
		t.Error("an accented Portuguese comment must be flagged")
	}
	if _, ok := notEnglish("isso aqui nao deve passar"); !ok {
		t.Error("two Portuguese function words without accents must be flagged")
	}
}

func TestCommentedOutCode(t *testing.T) {
	if !commentedOutCode([]string{"x := oldBuild(name);"}) {
		t.Error("an assignment in a comment is commented-out code")
	}
	if commentedOutCode([]string{"see repo.Find(ctx, id) for the query"}) {
		t.Error("a prose reference to a call is not commented-out code")
	}
	// Every entry below was a finding on the Lybel front-ends before the fix.
	prose := [][]string{
		{"@example", `const isDesktop = useMediaQuery("(min-width: 1024px)");`},
		{"```tsx", "<EmptyState onAction={() => navigate(\"/x\")} />", "```"},
		{"POST /provider/profile/instagram/ — add instagram post { url }"},
		{"slots. Text is computed server-side (working hours minus bookings/blocks);"},
		{"@returns {Promise<void>}"},
	}
	for _, lines := range prose {
		if commentedOutCode(lines) {
			t.Errorf("%q is documentation, not code left behind", lines)
		}
	}
}

// A single capitalised accented word inside an English sentence is a proper
// noun. Firing on it teaches the reader to ignore CMT-04.
func TestNotEnglishAllowsProperNouns(t *testing.T) {
	english := []string{
		"Provider timezone — Lybel is Brazil-only, so slots render in São Paulo time.",
		"The backend pins everything to America/Sao_Paulo while we are single-timezone.",
	}
	portuguese := []string{
		"Conteúdo",
		"Card 3 - Standard (O Físico Reinventado)",
		"Componentes base com tema automático",
	}
	for _, s := range english {
		if reason, ok := notEnglish(s); ok {
			t.Errorf("%q is English (%s)", s, reason)
		}
	}
	for _, s := range portuguese {
		if _, ok := notEnglish(s); !ok {
			t.Errorf("%q is Portuguese and should be caught", s)
		}
	}
}

// Product copy inside a usage example is not a Portuguese comment.
func TestProseLinesDropExamplesAndFences(t *testing.T) {
	lines := []string{
		"Section title with optional label.",
		"@example",
		`<SectionTitle highlight="Nada que não precisa.">`,
		"  Tudo que você precisa.",
		"</SectionTitle>",
	}
	got := strings.Join(proseLines(lines), " ")
	if strings.Contains(got, "você") {
		t.Errorf("the example body survived into the prose: %q", got)
	}
	if !strings.Contains(got, "Section title") {
		t.Errorf("the prose was dropped along with the example: %q", got)
	}
}

func TestRollingHashesMatchDirectComputation(t *testing.T) {
	values := []uint64{7, 11, 13, 17, 19, 23}
	const window = 3
	got := rollingHashes(values, window)
	for i := range got {
		var want uint64
		for k := 0; k < window; k++ {
			want = want*hashBase + values[i+k]
		}
		if got[i] != want {
			t.Errorf("window %d: rolling hash %d, direct %d", i, got[i], want)
		}
	}
}

// CMT-03 asks whether a comment earns its place. Asked about every comment that
// already exists it is noise; asked about the ones a delivery adds it is the
// question the rule was written for.
func TestBodyCommentIsAskedAboutOnlyWhenTheDeliveryAddedIt(t *testing.T) {
	comment := Comment{
		Line: 10, EndLine: 10, Pos: PosBody,
		Lines: []string{"walk the slots and keep the free ones"},
		Text:  "walk the slots and keep the free ones",
	}
	run := func(added []lineRange) []Finding {
		f := &File{Path: "x.go", Lang: LangGo, Comments: []Comment{comment}, AddedLines: added}
		var out []Finding
		checkComments(&Config{}, f, func(fi Finding) { out = append(out, fi) })
		return out
	}
	if got := run(nil); len(got) != 0 {
		t.Errorf("a pre-existing comment must stay silent, got %v", got)
	}
	got := run([]lineRange{{from: 8, to: 12}})
	if len(got) != 1 || got[0].Rule != "CMT-03" {
		t.Errorf("a comment the delivery added must be questioned, got %v", got)
	}
}

// A Greek letter in a maths note is not Portuguese.
func TestNotEnglishIgnoresNonLatinLetters(t *testing.T) {
	if reason, ok := notEnglish("0..π → fan upward/outward"); ok {
		t.Errorf("maths notation read as Portuguese (%s)", reason)
	}
}
