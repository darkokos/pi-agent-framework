package validate

import (
	"slices"
	"testing"
)

func TestParseExtensionSource(t *testing.T) {
	cases := []struct {
		in       string
		kind     string
		pinned   bool
		portable bool
		bad      bool
	}{
		{in: "npm:@foo/bar@1.2.3", kind: "npm", pinned: true, portable: true},
		{in: "npm:foo@1.2.3", kind: "npm", pinned: true, portable: true},
		{in: "npm:foo@v1.2.3-beta.1", kind: "npm", pinned: true, portable: true},
		{in: "npm:@foo/bar", kind: "npm", portable: true},
		{in: "npm:foo@^1.2.0", kind: "npm", portable: true},
		{in: "npm:foo@latest", kind: "npm", portable: true},
		{in: "git:github.com/user/repo@v1", kind: "git", pinned: true, portable: true},
		{in: "git:github.com/user/repo", kind: "git", portable: true},
		{in: "git:git@github.com:user/repo@abc123", kind: "git", pinned: true, portable: true},
		{in: "git:git@github.com:user/repo", kind: "git", portable: true},
		{in: "https://github.com/user/repo@v1", kind: "git", pinned: true, portable: true},
		{in: "https://github.com/user/repo", kind: "git", portable: true},
		{in: "ssh://git@github.com/user/repo@v1", kind: "git", pinned: true, portable: true},
		{in: "../my-extension", kind: "path", pinned: true, portable: true},
		{in: "extensions/local.ts", kind: "path", pinned: true, portable: true},
		{in: "/abs/path", kind: "path", pinned: true, portable: false},
		{in: "~/ext", kind: "path", pinned: true, portable: false},
		{in: "~", kind: "path", pinned: true, portable: false},
		{in: "npm:", bad: true},
		{in: "npm:@", bad: true},
		{in: "", bad: true},
		{in: "  ", bad: true},
	}

	for _, tc := range cases {
		got, err := ParseExtensionSource(tc.in)
		if tc.bad {
			if err == nil {
				t.Errorf("%q: expected an error", tc.in)
			}

			continue
		}

		if err != nil {
			t.Errorf("%q: unexpected error %v", tc.in, err)

			continue
		}
		if got.Kind != tc.kind || got.Pinned != tc.pinned || got.Portable != tc.portable {
			t.Errorf("%q: got %+v, want kind=%s pinned=%v portable=%v", tc.in, got, tc.kind, tc.pinned, tc.portable)
		}
	}
}

func TestNonBuiltinTools(t *testing.T) {
	got := NonBuiltinTools([]string{"read", "web_search", "bash", "custom"})
	if !slices.Equal(got, []string{"web_search", "custom"}) {
		t.Fatalf("got %v", got)
	}
}
