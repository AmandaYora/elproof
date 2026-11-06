package utils

import (
	"regexp"
	"testing"
)

var usernameRe = regexp.MustCompile(`^[a-z0-9_.]{4,32}$`)

func TestUsernameBase(t *testing.T) {
	cases := map[string]string{
		"Rina Kirana":                           "rina.kirana",
		"Rina.K@gmail.com":                      "rina.k",
		"  Dewi  -  Lestari ":                   "dewi.lestari",
		"Ö":                                     "fallback",
		"ab":                                    "fallback",
		"Muhammad Abdurrahman Al-Fatih Nugroho": "muhammad.abdurrahman.al",
	}
	for in, want := range cases {
		got := UsernameBase(in, "fallback")
		if got != want {
			t.Errorf("UsernameBase(%q) = %q, want %q", in, got, want)
		}
		if !usernameRe.MatchString(got) {
			t.Errorf("%q is not a valid username", got)
		}
		if s := UsernameWithSuffix(got); !usernameRe.MatchString(s) {
			t.Errorf("suffixed %q invalid", s)
		}
	}
}

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"a@b.co", "rina.kirana@gmail.com"} {
		if !ValidEmail(ok) {
			t.Errorf("%q rejected", ok)
		}
	}
	for _, bad := range []string{"", "a", "a@b", "a b@c.com", "@b.com", "a@.com", "a@b.com."} {
		if ValidEmail(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}
