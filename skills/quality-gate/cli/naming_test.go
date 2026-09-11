package main

import "testing"

// The signature is read from real source, so the parse and the rule are proven
// together: a Func built by hand would only prove the rule agrees with itself.
func TestPureSoundingNameOnIOIsReported(t *testing.T) {
	src := `package x

import "context"

type repo struct{}

func profileOf(ctx context.Context, id string) string { return id }
func chargeOf(id string) (int, error)                 { return 0, nil }
func OldestOutstandingOf(ctx context.Context) (int, error) { return 0, nil }
func (r *repo) accountOf(ctx context.Context, id string) string { return id }

func totalOf(prices []int) int { return len(prices) }
func Proof(ctx context.Context) error { return nil }
func Thereof() error { return nil }
func loadProfile(ctx context.Context, id string) (string, error) { return id, nil }
`
	f := parseGoSource(t, src)

	got := map[string]bool{}
	checkNaming(&Config{}, f, func(fi Finding) {
		if fi.Rule == "NAM-01" {
			got[nameAt(f, fi.Line)] = true
		}
	})

	for _, name := range []string{"profileOf", "chargeOf", "OldestOutstandingOf", "accountOf"} {
		if !got[name] {
			t.Errorf("%s does I/O behind a pure-sounding name and must be reported", name)
		}
	}
	for _, name := range []string{"totalOf", "Proof", "Thereof", "loadProfile"} {
		if got[name] {
			t.Errorf("%s must stay silent", name)
		}
	}
}

func nameAt(f *File, line int) string {
	for _, fn := range f.Funcs {
		if fn.Line == line {
			return fn.Name
		}
	}
	return ""
}
