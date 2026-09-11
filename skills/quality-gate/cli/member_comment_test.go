package main

import "testing"

// Positions come from the AST, so the source is parsed for real: a Comment
// built by hand would already carry the position under test.
func TestMemberCommentsAreRareAndShort(t *testing.T) {
	src := `package x

import "context"

type Contract struct {
	// Amount is what the seller is charged for the whole order, including
	// the shipping the marketplace retains on its side.
	Amount int
	// Kept for the vendor's audit trail, never shown to the seller.
	Ref string
	Bare string
}

type Gateway interface {
	// Makes sure the gateway knows the seller, then opens the page
	// where the card is typed.
	Open(ctx context.Context) error
	// Idempotent: a second call with the same key is a no-op at the vendor.
	Charge(ctx context.Context) error
	Close() error
}

type Silent struct {
	Name string
	Age  int
}

const (
	// A decl keeps its own budget, which a wrapped constraint
	// is allowed to use.
	Limit = 3
)
`
	f := parseGoSource(t, src)
	type at struct {
		rule string
		line int
	}
	got := map[at]Severity{}
	checkComments(&Config{}, f, func(fi Finding) { got[at{fi.Rule, fi.Line}] = fi.Sev })

	for _, c := range []struct {
		what string
		key  at
		sev  Severity
	}{
		{"two-line field comment", at{"CMT-01", 6}, SevError},
		{"two-line interface method comment", at{"CMT-01", 15}, SevError},
		{"one-line field comment", at{"CMT-10", 9}, SevWarn},
		{"one-line interface method comment", at{"CMT-10", 18}, SevWarn},
	} {
		if sev, ok := got[c.key]; !ok || sev != c.sev {
			t.Errorf("%s: expected %v as %s, got %q (all: %v)", c.what, c.key, c.sev, sev, got)
		}
	}
	for _, key := range []at{{"CMT-10", 6}, {"CMT-10", 15}, {"CMT-01", 9}, {"CMT-01", 18}, {"CMT-01", 29}} {
		if _, ok := got[key]; ok {
			t.Errorf("%v must stay silent (all: %v)", key, got)
		}
	}
	for key := range got {
		if key.line >= 23 && key.line <= 26 {
			t.Errorf("a struct with no member comments was reported: %v", key)
		}
	}
}
