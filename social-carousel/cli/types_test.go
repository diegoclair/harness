package main

import "testing"

// resolvePlatform must keep linkedin-4x5 and instagram-1x1 at SafePadX=0
// (base.css default 80px) and only widen instagram-4x5's horizontal safe
// area for the profile grid's 4:5→3:4 crop. Empty/unknown platforms fall
// back to instagram-4x5 (existing behavior), so they inherit the same padding.
func TestResolvePlatformSafePadX(t *testing.T) {
	cases := []struct {
		platform string
		wantPadX int
	}{
		{"instagram-4x5", gridSafePadX},
		{"instagram-1x1", 0},
		{"linkedin-4x5", 0},
		{"", gridSafePadX},
		{"unknown", gridSafePadX},
	}

	for _, tc := range cases {
		spec := resolvePlatform(&Carousel{Platform: tc.platform})
		if spec.SafePadX != tc.wantPadX {
			t.Errorf("platform %q: SafePadX = %d, want %d", tc.platform, spec.SafePadX, tc.wantPadX)
		}
	}
}

func TestLinkedInPaddingUnchanged(t *testing.T) {
	spec := resolvePlatform(&Carousel{Platform: "linkedin-4x5"})
	if spec.SafePadX != 0 {
		t.Fatalf("linkedin-4x5 must keep SafePadX=0 so base.css's 80px default applies unchanged, got %d", spec.SafePadX)
	}
}
