package domain

import "testing"

func TestDocumentCodeFromName(t *testing.T) {
	cases := map[string]string{
		"Melati Organizer":           "MO",
		"Bunga Wedding Organizer":    "BWO",
		"Lavanya":                    "LAV",
		"  sinar  & bulan wedding  ": "SBW",
		"A B C D E F G":              "ABCDE",
		"Yo":                         "YO",
		"X":                          "WO",
		"":                           "WO",
		"Rumah 99 Wedding":           "R9W",
		"Éclat Mariage":              "CM",
	}
	for in, want := range cases {
		if got := DocumentCodeFromName(in); got != want {
			t.Errorf("DocumentCodeFromName(%q) = %q, want %q", in, got, want)
		}
		if !ValidDocumentCode(DocumentCodeFromName(in)) {
			t.Errorf("derived code for %q is not valid", in)
		}
	}
}

func TestValidDocumentCode(t *testing.T) {
	for _, ok := range []string{"MO", "BWO", "LAV12", "99"} {
		if !ValidDocumentCode(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "M", "ABCDEF", "mo", "M-O", "MÖ"} {
		if ValidDocumentCode(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
