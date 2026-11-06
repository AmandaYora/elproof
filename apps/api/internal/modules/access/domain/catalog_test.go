package domain

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestTemplatesUseCatalogKeysOfTheirKind(t *testing.T) {
	seen := map[string]bool{}
	for _, tpl := range Templates {
		if seen[tpl.Key] {
			t.Fatalf("duplicate template %s", tpl.Key)
		}
		seen[tpl.Key] = true
		for _, k := range tpl.Permissions {
			p, ok := Lookup(k)
			if !ok || p.Kind != tpl.Kind {
				t.Errorf("template %s: %q unknown or wrong kind", tpl.Key, k)
			}
		}
	}
	keys := map[string]bool{}
	for _, p := range Catalog {
		if keys[p.Key] {
			t.Errorf("duplicate catalog key %s", p.Key)
		}
		keys[p.Key] = true
		if p.Label == "" || p.Description == "" || p.Group == "" {
			t.Errorf("%s needs a group, label and description for the Peran & Akses page", p.Key)
		}
	}
}

// The migration that seeded existing workspaces must match the templates new
// workspaces get, or old and new tenants would behave differently.
func TestMigrationMatchesTemplates(t *testing.T) {
	raw, err := os.ReadFile("../../../../migrations/000058_create_tenant_roles.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	re := regexp.MustCompile(`(?s)JOIN \((.*?)\) p WHERE r\.template_key = '(\w+)'`)
	found := map[string][]string{}
	for _, m := range re.FindAllStringSubmatch(sql, -1) {
		perms := regexp.MustCompile(`'([a-z_.]+)'`).FindAllStringSubmatch(m[1], -1)
		for _, p := range perms {
			found[m[2]] = append(found[m[2]], p[1])
		}
	}
	for _, tpl := range Templates {
		want := append([]string(nil), tpl.Permissions...)
		got := found[tpl.Key]
		sort.Strings(want)
		sort.Strings(got)
		if strings.Join(want, ",") != strings.Join(got, ",") {
			t.Errorf("template %s: migration has %v, code has %v", tpl.Key, got, want)
		}
		if !strings.Contains(sql, "'"+tpl.Name+"'") {
			t.Errorf("template %s: name %q missing from migration", tpl.Key, tpl.Name)
		}
	}
}
