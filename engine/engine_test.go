package engine

import (
	"strings"
	"testing"
)

// TestDurchspielen spielt die komplette Rätselkette einmal durch.
func TestDurchspielen(t *testing.T) {
	g := New()
	schritte := []struct{ in, erwartet string }{
		{"ls", "nachricht_von_mara.txt"},
		{"cat nachricht_von_mara.txt", "Hundes"},
		{"ls /home/mara/fotos", "pixel_im_tiefschnee.jpg"},
		{"cat /home/mara/tagebuch.txt", "Zugriff verweigert"},
		{"su mara", ""},
		{"PIXEL", "Du bist jetzt mara"},
		{"cat tagebuch.txt", "ls -a"},
		{"ls", "tagebuch.txt"},
		{"ls -a", ".notfall/"},
		{"cd .notfall", ""},
		{"cat plan.txt", "base64 -d"},
		{"base64 -d teil1.b64", "Codeteil 1 notiert: NORD"},
		{"cat /var/log/auth.log", "2418 Zeilen"},
		{"grep berg /var/log/auth.log", "Eisblume42"},
		{"su berg", ""},
		{"falsch", "Falsches Passwort"},
		{"su berg", ""},
		{"Eisblume42", "Du bist jetzt berg"},
		{"history", "Codeteil 2 notiert: LICHT"},
		{"cd /opt/haldor", ""},
		{"cat kern.bin", "Binärdatei"},
		{"strings kern.bin", "Codeteil 3 notiert: STERN"},
		{"./notabschaltung", "blockiert"},
		{"ps", "haldor-waechter"},
		{"kill 1337", "beendet"},
		{"./notabschaltung", "Notfallcode"},
		{"nord licht mond", "Code falsch"},
		{"./notabschaltung", ""},
		{"Nord-Licht-Stern", "NOTABSCHALTUNG ERFOLGREICH"},
	}
	for _, s := range schritte {
		out := Plain(g.Exec(s.in).Text)
		if !strings.Contains(out, s.erwartet) {
			t.Fatalf("%q: erwartet %q in\n%s", s.in, s.erwartet, out)
		}
	}
	if st := g.Status(); !st.Won || st.Parts != [3]string{"NORD", "LICHT", "STERN"} {
		t.Fatalf("Status falsch: %+v", st)
	}
}

func TestRechte(t *testing.T) {
	g := New()
	for in, want := range map[string]string{
		"ls /opt/haldor":       "Zugriff verweigert",
		"kill 1337":            "Keine Berechtigung",
		"cd /home/berg":        "Zugriff verweigert",
		"notabschaltung":       "Befehl nicht gefunden",
		"cat /home/mara/fotos": "ist ein Ordner",
	} {
		if out := Plain(g.Exec(in).Text); !strings.Contains(out, want) {
			t.Errorf("%q: erwartet %q, bekam %q", in, want, out)
		}
	}
}

func TestComplete(t *testing.T) {
	g := New()
	cases := map[string]string{
		"ca":                    "cat ",
		"cat nach":              "cat nachricht_von_mara.txt ",
		"ls /home/ma":           "ls /home/mara/",
		"ls /home/mara/fotos/p": "ls /home/mara/fotos/p",
		"cat xyz":               "cat xyz",
	}
	for in, want := range cases {
		if got, _ := g.Complete(in); got != want {
			t.Errorf("Complete(%q) = %q, erwartet %q", in, got, want)
		}
	}
}

func TestHinweiseUndPrompt(t *testing.T) {
	g := New()
	if p := g.Prompt(); p != "techniker@nordlicht:~$ " {
		t.Fatalf("Prompt: %q", p)
	}
	h1 := g.Exec("hinweis").Text
	h2 := g.Exec("hinweis").Text
	if h1 == h2 || g.Status().Hints != 2 || g.Status().Moves != 0 {
		t.Fatalf("Hinweise zählen falsch: %+v", g.Status())
	}
	g.Exec("cd /var/log")
	if p := g.Prompt(); p != "techniker@nordlicht:/var/log$ " {
		t.Fatalf("Prompt: %q", p)
	}
}
