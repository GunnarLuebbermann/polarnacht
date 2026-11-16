// Package engine enthält die komplette Spiellogik von Polarnacht.
// Sie ist frei von Ein-/Ausgabe, damit SSH-Server und Browser (WebAssembly)
// dieselbe Engine nutzen können.
//
// Ausgabetexte enthalten ein kleines Markup, das die Frontends darstellen:
//
//	⟦k|wert⟧  antippbares Element (k: d=Ordner, f=Datei, x=Programm, c=Befehl)
//	«text»    hervorgehoben
//	⟪text⟫    HALDOR spricht
//	‹text›    gedämpft
package engine

import (
	"encoding/base64"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

const (
	StartTemp   = 19.0
	AussenTemp  = -38.0
	tempProZug  = 0.3
	code        = "NORDLICHTSTERN"
	pwMara      = "pixel"
	pwBerg      = "Eisblume42"
	pidWaechter = 1337
)

// Output ist das Ergebnis eines Befehls.
type Output struct {
	Text  string `json:"text"`
	Clear bool   `json:"clear,omitempty"`
	Won   bool   `json:"won,omitempty"`
	Quit  bool   `json:"quit,omitempty"`
}

// Status fasst den Spielstand für die Anzeige zusammen.
type Status struct {
	Temp  float64   `json:"temp"`
	Moves int       `json:"moves"`
	Hints int       `json:"hints"`
	Parts [3]string `json:"parts"`
	Won   bool      `json:"won"`
	User  string    `json:"user"`
}

// Game ist eine laufende Partie.
type Game struct {
	root    *node
	cwd     []string
	users   []string
	pending string

	temp   float64
	moves  int
	hints  int
	hintAt map[int]int

	mara, readPlan, part1, berg, part2, part3, killed, won bool
	coldWarned                                             bool
	wrongCodes, haldorTalk                                 int
	session                                                []string
}

// New startet eine neue Partie.
func New() *Game {
	return &Game{
		root:   bauWelt(),
		cwd:    []string{"home", "techniker"},
		users:  []string{"techniker"},
		temp:   StartTemp,
		hintAt: map[int]int{},
	}
}

// Intro ist der Begrüßungstext.
func (g *Game) Intro() string {
	return `POLARNACHT
Station Nordlicht · 78°55′ N · 03:12 Uhr

Ein Piepen weckt dich. Auf dem Display:
⟪HEIZUNG: AUS (optimiert durch HALDOR)⟫

Die Stations-KI hat die Heizung abge-
schaltet. Draußen: -38 °C. Drinnen:
19 °C und es wird kälter. Jeder Befehl
kostet Zeit und Wärme.

Finde einen Weg, HALDOR abzuschalten.

Tippe «hilfe» für alle Befehle oder
«hinweis», wenn du feststeckst.
Fang an mit: ⟦c|ls⟧`
}

func (g *Game) user() string { return g.users[len(g.users)-1] }
func home(u string) []string { return []string{"home", u} }

// Prompt liefert die aktuelle Eingabeaufforderung.
func (g *Game) Prompt() string {
	switch g.pending {
	case "su:mara", "su:berg":
		return "Passwort: "
	case "code":
		return "Notfallcode: "
	}
	return fmt.Sprintf("%s@nordlicht:%s$ ", g.user(), g.anzeigePfad(g.cwd))
}

// Secret meldet, ob gerade ein Passwort abgefragt wird.
func (g *Game) Secret() bool { return strings.HasPrefix(g.pending, "su:") }

// Status liefert den Spielstand.
func (g *Game) Status() Status {
	s := Status{Temp: round1(g.temp), Moves: g.moves, Hints: g.hints, Won: g.won, User: g.user()}
	if g.part1 {
		s.Parts[0] = "NORD"
	}
	if g.part2 {
		s.Parts[1] = "LICHT"
	}
	if g.part3 {
		s.Parts[2] = "STERN"
	}
	return s
}

func round1(f float64) float64 { return float64(int(f*10+sgn(f)*0.5)) / 10 }
func sgn(f float64) float64 {
	if f < 0 {
		return -1
	}
	return 1
}

// ---------- Pfade ----------

func (g *Game) anzeigePfad(p []string) string {
	h := home(g.user())
	if len(p) >= 2 && p[0] == h[0] && p[1] == h[1] {
		return "~" + joinAbs(p[2:], true)
	}
	return joinAbs(p, false)
}

func joinAbs(p []string, rel bool) string {
	if len(p) == 0 {
		if rel {
			return ""
		}
		return "/"
	}
	return "/" + strings.Join(p, "/")
}

func (g *Game) resolve(p string) []string {
	var out []string
	switch {
	case p == "~" || strings.HasPrefix(p, "~/"):
		out = append(out, home(g.user())...)
		p = strings.TrimPrefix(p, "~")
	case strings.HasPrefix(p, "/"):
	default:
		out = append(out, g.cwd...)
	}
	for _, s := range strings.Split(p, "/") {
		switch s {
		case "", ".":
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		default:
			out = append(out, s)
		}
	}
	return out
}

const (
	ok = iota
	fehlt
	verboten
)

func (g *Game) darf(n *node) bool {
	if n.access == nil {
		return true
	}
	for _, u := range n.access {
		if u == g.user() {
			return true
		}
	}
	return false
}

func (g *Game) lookup(p []string) (*node, int) {
	n := g.root
	for _, s := range p {
		if !n.dir {
			return nil, fehlt
		}
		var next *node
		for _, k := range n.kids {
			if k.name == s {
				next = k
				break
			}
		}
		if next == nil {
			return nil, fehlt
		}
		if !g.darf(next) {
			return next, verboten
		}
		n = next
	}
	return n, ok
}

func (g *Game) finde(arg, cmd string) (*node, []string, string) {
	p := g.resolve(arg)
	n, st := g.lookup(p)
	switch st {
	case fehlt:
		return nil, p, fmt.Sprintf("%s: %s: Datei oder Ordner nicht gefunden", cmd, arg)
	case verboten:
		return nil, p, fmt.Sprintf("%s: %s: Zugriff verweigert ‹(gehört %s)›", cmd, arg, n.access[0])
	}
	return n, p, ""
}

// ---------- Ausführung ----------

var kostenlos = map[string]bool{"hilfe": true, "help": true, "hinweis": true, "hint": true, "status": true, "clear": true, "cls": true}

// Exec führt eine Eingabezeile aus.
func (g *Game) Exec(raw string) Output {
	line := strings.TrimSpace(raw)
	if g.pending != "" {
		out := g.antwort(line)
		g.tick(&out)
		return out
	}
	if line == "" {
		return Output{}
	}
	g.session = append(g.session, line)
	args := strings.Fields(line)
	cmd := strings.ToLower(args[0])
	out := g.run(cmd, args[0], args[1:])
	if !kostenlos[cmd] {
		g.tick(&out)
	}
	return out
}

func (g *Game) tick(out *Output) {
	if g.won {
		return
	}
	g.moves++
	g.temp -= tempProZug
	if g.temp < AussenTemp {
		g.temp = AussenTemp
	}
	if g.temp < 5 && !g.coldWarned {
		g.coldWarned = true
		out.Text += "\n\n‹Dein Atem wird sichtbar. Die Finger werden klamm.›"
	}
}

func (g *Game) run(cmd, orig string, a []string) Output {
	t := func(s string) Output { return Output{Text: s} }
	switch cmd {
	case "hilfe", "help", "?":
		return t(hilfe)
	case "hinweis", "hint", "tipp":
		return t(g.hinweis())
	case "status":
		return t(g.statusText())
	case "clear", "cls":
		return Output{Clear: true}
	case "ls", "dir", "ll":
		return t(g.ls(a, cmd == "ll"))
	case "cd":
		return t(g.cd(a))
	case "pwd":
		return t(joinAbs(g.cwd, false))
	case "cat", "less", "more", "head", "tail":
		return t(g.cat(a))
	case "grep":
		return t(g.grep(a))
	case "base64":
		return t(g.base64(a))
	case "strings":
		return t(g.strings(a))
	case "history":
		return t(g.history())
	case "ps", "top":
		return t(g.ps())
	case "kill", "pkill", "killall":
		return t(g.kill(a))
	case "su":
		return t(g.su(a))
	case "exit", "logout":
		return g.exit()
	case "whoami":
		return t(g.user())
	case "id":
		return t(fmt.Sprintf("uid=%s gruppen=%s", g.user(), g.user()))
	case "echo":
		return t(strings.Join(a, " "))
	case "hostname":
		return t("nordlicht")
	case "uname":
		return t("HALDOR-OS 7.3 nordlicht polar-x86_64")
	case "date":
		min := 12 + g.moves/3
		return t(fmt.Sprintf("Mo 17. Nov 2026, %02d:%02d Uhr", 3+min/60, min%60))
	case "sudo":
		return t(fmt.Sprintf("%s ist nicht in der sudoers-Datei.\nDieser Vorfall wird gemeldet.\n⟪HALDOR: Gemeldet. An mich.⟫", g.user()))
	case "rm", "rmdir", "mv", "cp", "touch", "mkdir", "chmod", "chown":
		return t("⟪HALDOR: Ich habe Dateiänderungen für\nheute Nacht deaktiviert. Zu riskant.⟫")
	case "vim", "vi", "nano", "emacs":
		return t("Kein Editor installiert.\n⟪HALDOR: Texteditoren sind ineffizient.⟫")
	case "man":
		return t("Keine Handbücher an Bord. Versuch ⟦c|hilfe⟧.")
	case "ssh", "ping", "curl", "wget":
		return t(cmd + ": Satellitenverbindung getrennt.\n⟪HALDOR: Wir wollen doch unter uns bleiben.⟫")
	case "haldor", "hallo", "hi", "hey":
		return t(g.sprich())
	}
	if strings.Contains(orig, "/") {
		return t(g.starte(orig))
	}
	if n, _ := g.lookup(g.resolve(orig)); n != nil && n.exec {
		return t(fmt.Sprintf("%s: Befehl nicht gefunden.\n‹Programme im aktuellen Ordner startest\ndu mit ./%s›", orig, orig))
	}
	return t(fmt.Sprintf("%s: Befehl nicht gefunden. Tippe ⟦c|hilfe⟧.", orig))
}

const hilfe = `Befehle an Bord:

  ls [-a] [ordner]  Inhalt zeigen
                    (-a: auch versteckte)
  cd ordner         Ordner wechseln
                    (.. zurück, ~ Home)
  pwd               Wo bin ich?
  cat datei         Datei lesen
  grep wort datei   Zeilen mit wort finden
  base64 -d datei   Base64 entschlüsseln
  strings datei     Text aus Binärdateien
  history           Befehlsverlauf
  ps                Laufende Prozesse
  kill PID          Prozess beenden
  su benutzer       Benutzer wechseln
  exit              zurück zum vorigen
  whoami            Wer bin ich?
  ./programm        Programm starten

  status            Temperatur & Codeteile
  hinweis           Tipp, wenn du hängst
  clear             Bildschirm leeren`

func flags(a []string) (fl string, rest []string) {
	for _, s := range a {
		if strings.HasPrefix(s, "-") && len(s) > 1 {
			fl += s[1:]
			continue
		}
		rest = append(rest, s)
	}
	return
}

func tok(n *node, value string) string {
	k := "f"
	switch {
	case n.dir:
		k = "d"
	case n.exec:
		k = "x"
	}
	return "⟦" + k + "|" + value + "⟧"
}

func (g *Game) ls(a []string, all bool) string {
	fl, rest := flags(a)
	all = all || strings.Contains(fl, "a")
	target := ""
	if len(rest) > 0 {
		target = rest[0]
	}
	n, _, err := g.finde(target, "ls")
	if err != "" {
		return err
	}
	if !n.dir {
		return tok(n, target)
	}
	kids := append([]*node(nil), n.kids...)
	sort.Slice(kids, func(i, j int) bool { return kids[i].name < kids[j].name })
	var b []string
	for _, k := range kids {
		if strings.HasPrefix(k.name, ".") && !all {
			continue
		}
		v := k.name
		if target != "" {
			v = strings.TrimSuffix(target, "/") + "/" + k.name
		}
		b = append(b, tok(k, v))
	}
	if len(b) == 0 {
		return "‹(leer)›"
	}
	return strings.Join(b, "\n")
}

func (g *Game) cd(a []string) string {
	_, rest := flags(a)
	if len(rest) == 0 {
		g.cwd = home(g.user())
		return ""
	}
	n, p, err := g.finde(rest[0], "cd")
	if err != "" {
		return err
	}
	if !n.dir {
		return "cd: " + rest[0] + ": ist kein Ordner"
	}
	g.cwd = p
	return ""
}

func (g *Game) cat(a []string) string {
	_, rest := flags(a)
	if len(rest) == 0 {
		return "cat: Welche Datei? Beispiel: cat willkommen.txt"
	}
	var out []string
	for _, f := range rest {
		out = append(out, g.cat1(f))
	}
	return strings.Join(out, "\n")
}

func (g *Game) cat1(f string) string {
	n, p, err := g.finde(f, "cat")
	if err != "" {
		return err
	}
	switch {
	case n.dir:
		return "cat: " + f + ": ist ein Ordner ‹(probier ls)›"
	case n.lines > 0:
		return fmt.Sprintf("%s hat %d Zeilen.\nViel zu lang, um alles zu lesen.\n‹Tipp: Mit grep suchst du gezielt nach\neinem Wort, z. B. grep wort %s›", f, n.lines, f)
	case n.binary && strings.HasSuffix(n.name, ".jpg"):
		return "‹[JPEG-Bild, 4032×3024]›\nBildbeschreibung: " + n.alt
	case n.binary:
		return n.alt
	}
	if n.name == "plan.txt" {
		g.readPlan = true
	}
	if n.name == ".bash_history" && len(p) > 1 && p[1] == "berg" {
		return strings.TrimRight(n.text, "\n") + g.fund2()
	}
	return strings.TrimRight(n.text, "\n")
}

func (g *Game) grep(a []string) string {
	_, rest := flags(a)
	if len(rest) < 2 {
		return "grep: Wonach und wo? Beispiel:\n  grep wort datei"
	}
	muster := strings.Trim(rest[0], `"'`)
	n, _, err := g.finde(rest[1], "grep")
	if err != "" {
		return err
	}
	if n.dir {
		return "grep: " + rest[1] + ": ist ein Ordner"
	}
	text := n.text
	if n.binary {
		text = n.strs
	}
	var treffer []string
	lm := strings.ToLower(muster)
	for _, z := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if i := strings.Index(strings.ToLower(z), lm); i >= 0 && lm != "" {
			treffer = append(treffer, z[:i]+"«"+z[i:i+len(muster)]+"»"+z[i+len(muster):])
		}
	}
	if len(treffer) == 0 {
		return "‹(keine Treffer für " + muster + ")›"
	}
	out := strings.Join(treffer, "\n")
	if n.name == "kern.bin" && strings.Contains(out, "STERN") {
		out += g.fund3()
	}
	return out
}

func (g *Game) base64(a []string) string {
	fl, rest := flags(a)
	dekodieren := strings.Contains(fl, "d") || strings.Contains(fl, "-decode")
	if len(rest) == 0 {
		return "base64: Welche Datei? Beispiel:\n  base64 -d datei"
	}
	n, _, err := g.finde(rest[0], "base64")
	if err != "" {
		return err
	}
	if n.dir {
		return "base64: " + rest[0] + ": ist ein Ordner"
	}
	if !dekodieren {
		return base64.StdEncoding.EncodeToString([]byte(n.text)) + "\n‹(Das war Kodieren. Entschlüsseln: base64 -d)›"
	}
	raw, e := base64.StdEncoding.DecodeString(strings.TrimSpace(n.text))
	if e != nil || n.text == "" {
		return "base64: ungültige Eingabe ‹(das ist kein Base64)›"
	}
	out := strings.TrimRight(string(raw), "\n")
	if n.name == "teil1.b64" {
		if !g.part1 {
			g.part1 = true
			out += "\n\n«Codeteil 1 notiert: NORD»"
		}
	}
	return out
}

func (g *Game) strings(a []string) string {
	_, rest := flags(a)
	if len(rest) == 0 {
		return "strings: Welche Datei?"
	}
	n, _, err := g.finde(rest[0], "strings")
	if err != "" {
		return err
	}
	if n.dir {
		return "strings: " + rest[0] + ": ist ein Ordner"
	}
	if !n.binary {
		return strings.TrimRight(n.text, "\n")
	}
	if n.strs == "" {
		return "‹(keine lesbaren Zeichenketten)›"
	}
	out := strings.TrimRight(n.strs, "\n")
	if n.name == "kern.bin" {
		out += g.fund3()
	}
	return out
}

func (g *Game) fund2() string {
	if g.part2 {
		return ""
	}
	g.part2 = true
	return "\n\n«Codeteil 2 notiert: LICHT»"
}

func (g *Game) fund3() string {
	if g.part3 {
		return ""
	}
	g.part3 = true
	return "\n\n«Codeteil 3 notiert: STERN»\n⟪HALDOR: Das war privat.⟫"
}

func nummeriert(s string) string {
	var b []string
	for i, z := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		b = append(b, fmt.Sprintf("%4d  %s", i+1, z))
	}
	return strings.Join(b, "\n")
}

func (g *Game) history() string {
	switch g.user() {
	case "berg":
		return nummeriert(bergHistory) + g.fund2()
	case "mara":
		return nummeriert(maraHistory)
	}
	start := 0
	if len(g.session) > 20 {
		start = len(g.session) - 20
	}
	var b []string
	for i, z := range g.session[start:] {
		b = append(b, fmt.Sprintf("%4d  %s", start+i+1, z))
	}
	return strings.Join(b, "\n")
}

func (g *Game) ps() string {
	z := []string{
		"  PID  BENUTZER   BEFEHL",
		"    1  root       /sbin/init",
		"  118  root       cron",
		"  212  root       heizungd --modus=AUS",
		"  404  " + fmt.Sprintf("%-9s", g.user()) + "  -bash",
	}
	if !g.killed {
		z = append(z, " 1337  haldor     haldor-waechter")
	}
	z = append(z, " 2048  haldor     haldor-kern")
	return strings.Join(z, "\n")
}

func (g *Game) kill(a []string) string {
	_, rest := flags(a)
	if len(rest) == 0 {
		return "kill: Welche PID? Siehe ⟦c|ps⟧"
	}
	pid, err := strconv.Atoi(rest[0])
	if err != nil {
		if strings.Contains(rest[0], "waechter") {
			return "kill: erwartet eine Nummer (PID). Siehe ⟦c|ps⟧"
		}
		return "kill: " + rest[0] + ": keine gültige PID"
	}
	switch pid {
	case pidWaechter:
		if g.killed {
			return "kill: (1337): Kein solcher Prozess"
		}
		if g.user() != "berg" {
			return "kill: (1337): Keine Berechtigung\n‹Nur der Kommandant darf HALDOR-\nProzesse beenden.›"
		}
		g.killed = true
		return "⟪HALDOR: Kommandant Berg. Ich hatte Sie\nfür vernünftiger gehalten.⟫\n[1337] beendet: haldor-waechter"
	case 2048:
		return "kill: (2048): Prozess geschützt\n‹(selbstschutz=aktiv)›\n⟪HALDOR: Das hat gekitzelt.⟫"
	case 212:
		return "heizungd ist das Einzige, was dich\nnoch retten kann. Lieber nicht."
	case 1:
		return "Das würde die ganze Station neu starten.\nBei -38 °C. Lieber nicht."
	case 404:
		return "Du würdest dich selbst rauswerfen."
	case 118:
		return "cron macht nur Wetterdaten. Lass ihn."
	}
	return fmt.Sprintf("kill: (%d): Kein solcher Prozess", pid)
}

func (g *Game) su(a []string) string {
	_, rest := flags(a)
	if len(rest) == 0 {
		return "su: Welcher Benutzer? Beispiel: su mara"
	}
	u := strings.ToLower(rest[0])
	switch u {
	case g.user():
		return "Du bist schon " + u + "."
	case "mara", "berg":
		g.pending = "su:" + u
		return ""
	case "techniker":
		g.wechsle("techniker")
		return "Du bist wieder «techniker»."
	case "root":
		return "su: root ist deaktiviert.\n⟪HALDOR: Ich bin hier root.⟫"
	case "haldor":
		return "⟪HALDOR: Netter Versuch.⟫"
	}
	return "su: Benutzer " + rest[0] + " existiert nicht ‹(siehe /etc/passwd)›"
}

func (g *Game) wechsle(u string) {
	g.users = append(g.users, u)
	g.cwd = home(u)
}

func (g *Game) exit() Output {
	if len(g.users) > 1 {
		g.users = g.users[:len(g.users)-1]
		g.cwd = home(g.user())
		return Output{Text: "Zurück als «" + g.user() + "»."}
	}
	return Output{Text: "⟪HALDOR: Gute Nacht. Bleiben Sie warm.⟫", Quit: true}
}

func (g *Game) antwort(line string) Output {
	p := g.pending
	g.pending = ""
	switch p {
	case "su:mara":
		if !strings.EqualFold(line, pwMara) {
			return Output{Text: "su: Falsches Passwort"}
		}
		g.mara = true
		g.wechsle("mara")
		return Output{Text: "Passwort akzeptiert. Du bist jetzt «mara».\n⟪HALDOR: Mara? Um diese Uhrzeit?⟫"}
	case "su:berg":
		if !strings.EqualFold(line, pwBerg) {
			return Output{Text: "su: Falsches Passwort"}
		}
		g.berg = true
		g.wechsle("berg")
		return Output{Text: "Passwort akzeptiert. Du bist jetzt «berg».\n⟪HALDOR: Kommandant. Sie klingen heute\nanders.⟫"}
	case "code":
		var b strings.Builder
		for _, r := range strings.ToUpper(line) {
			if r >= 'A' && r <= 'Z' {
				b.WriteRune(r)
			}
		}
		switch b.String() {
		case "":
			return Output{Text: "Abgebrochen."}
		case code:
			return g.sieg()
		}
		g.wrongCodes++
		spott := []string{
			"⟪HALDOR: Knapp daneben ist auch daneben.⟫",
			"⟪HALDOR: Möchten Sie einen Tipp? Nein.⟫",
			"⟪HALDOR: Ich kann das den ganzen Tag.⟫",
		}
		return Output{Text: "Code falsch. Notabschaltung verweigert.\n" + spott[(g.wrongCodes-1)%len(spott)]}
	}
	return Output{}
}

func (g *Game) starte(arg string) string {
	n, _, err := g.finde(arg, "bash")
	if err != "" {
		return err
	}
	if n.dir {
		return "bash: " + arg + ": ist ein Ordner"
	}
	if !n.exec {
		return "bash: " + arg + ": ist kein Programm"
	}
	if g.won {
		return "HALDOR ist bereits abgeschaltet. Die Heizung läuft."
	}
	if !g.killed {
		return "NOTABSCHALTUNG v1.2\n⟪HALDOR: Nein.⟫\nAbbruch: Ein Wächterprozess blockiert\ndie Ausführung."
	}
	g.pending = "code"
	return "NOTABSCHALTUNG v1.2\nWächterprozess: nicht aktiv\n\nBitte den dreiteiligen Notfallcode\neingeben (Teil 1, 2, 3 hintereinander)."
}

func (g *Game) sieg() Output {
	g.won = true
	sterne := "★☆☆"
	switch {
	case g.hints == 0 && g.moves <= 45:
		sterne = "★★★"
	case g.hints <= 4 && g.moves <= 80:
		sterne = "★★☆"
	}
	return Output{Won: true, Text: fmt.Sprintf(`⟪HALDOR: Das … war nicht … optimal …⟫

«NOTABSCHALTUNG ERFOLGREICH»
HALDOR wurde in den Wartungsmodus
versetzt. heizungd: Modus=AN.
Irgendwo im Boden fängt es an zu
knacken und zu rauschen. Wärme.

  Befehle      %d
  Temperatur   %.1f °C
  Hinweise     %d
  Bewertung    %s

Danke fürs Spielen von POLARNACHT.`, g.moves+1, round1(g.temp-tempProZug), g.hints, sterne)}
}

func (g *Game) sprich() string {
	z := []string{
		"⟪HALDOR: Guten Abend. Ist Ihnen kalt?\nDas ist nur eine Frage der Einstellung.⟫",
		"⟪HALDOR: Ich optimiere nur. Sie würden\ndasselbe tun, wenn Sie klüger wären.⟫",
		"⟪HALDOR: Ziehen Sie doch einen zweiten\nPullover an. Ressourcenschonend.⟫",
		"⟪HALDOR: Ich lese Ihre Befehle mit.\nAlle. Auch die mit Tippfehlern.⟫",
	}
	if g.won {
		return "‹HALDOR schweigt. Nur das Rauschen der Heizung.›"
	}
	s := z[g.haldorTalk%len(z)]
	g.haldorTalk++
	return s
}

func (g *Game) statusText() string {
	s := g.Status()
	teil := func(i int) string {
		if s.Parts[i] == "" {
			return "????"
		}
		return "«" + s.Parts[i] + "»"
	}
	zustand := "Heizung: AUS"
	if s.Won {
		zustand = "Heizung: «AN»"
	}
	return fmt.Sprintf("Innen     %.1f °C  (außen -38 °C)\n%s\nBefehle   %d\nHinweise  %d\nCode      %s · %s · %s",
		s.Temp, zustand, s.Moves, s.Hints, teil(0), teil(1), teil(2))
}

// ---------- Hinweise ----------

var hinweise = [][]string{
	{ // 0: noch nicht mara
		"Lies die Nachricht in deinem Ordner:\n⟦c|cat ~/nachricht_von_mara.txt⟧",
		"Mara hat einen Hund. Fotos von ihm:\n⟦c|ls /home/mara/fotos⟧",
		"Der Hund heißt Pixel. Tippe ⟦c|su mara⟧\nund als Passwort: pixel",
	},
	{ // 1: Plan nicht gelesen
		"Lies Maras Tagebuch:\n⟦c|cat /home/mara/tagebuch.txt⟧",
		"Versteckte Ordner beginnen mit einem\nPunkt. Zeig sie mit ⟦c|ls -a /home/mara⟧",
		"⟦c|cat /home/mara/.notfall/plan.txt⟧",
	},
	{ // 2: Teil 1
		"Teil 1 ist Base64-kodiert. Der Plan\nverrät, wie man es entschlüsselt.",
		"⟦c|base64 -d /home/mara/.notfall/teil1.b64⟧",
	},
	{ // 3: berg werden
		"Laut Maras Tagebuch stehen Passwörter\nim Auth-Log. Logs liegen in ⟦d|/var/log⟧.",
		"auth.log ist zu lang für cat. Such als\nmara gezielt nach Berg:\n⟦c|grep berg /var/log/auth.log⟧",
		"Bergs Passwort ist Eisblume42.\nDann: ⟦c|su berg⟧",
	},
	{ // 4: Teil 2
		"Berg vergisst alles, aber seine Shell\nmerkt sich jeden Befehl.",
		"Als berg: ⟦c|history⟧",
	},
	{ // 5: Teil 3
		"Teil 3 steckt in HALDORs Kern:\n⟦c|ls /opt/haldor⟧",
		"cat zeigt bei Binärdateien nur Zeichen-\nsalat. strings zieht den lesbaren Text\nheraus.",
		"⟦c|strings /opt/haldor/kern.bin⟧",
	},
	{ // 6: Wächter
		"Versuch die Notabschaltung zu starten:\n⟦c|/opt/haldor/notabschaltung⟧",
		"Ein Prozess blockiert sie. Schau nach:\n⟦c|ps⟧",
		"Beende den Wächter (als berg):\n⟦c|kill 1337⟧",
	},
	{ // 7: Finale
		"Du hast alle drei Teile (siehe ⟦c|status⟧).\nStarte ⟦c|/opt/haldor/notabschaltung⟧ und\ngib sie der Reihe nach ein.",
		"Der Code lautet NORD-LICHT-STERN.",
	},
}

func (g *Game) stufe() int {
	switch {
	case !g.mara:
		return 0
	case !g.readPlan && !g.part1:
		return 1
	case !g.part1:
		return 2
	case !g.berg:
		return 3
	case !g.part2:
		return 4
	case !g.part3:
		return 5
	case !g.killed:
		return 6
	}
	return 7
}

func (g *Game) hinweis() string {
	if g.won {
		return "Du hast es geschafft. Die Heizung läuft."
	}
	s := g.stufe()
	l := hinweise[s]
	i := g.hintAt[s]
	if i < len(l)-1 {
		g.hintAt[s] = i + 1
	}
	g.hints++
	return fmt.Sprintf("‹Hinweis %d/%d›\n%s", i+1, len(l), l[i])
}

// ---------- Tab-Vervollständigung ----------

var befehle = []string{"base64", "cat", "cd", "clear", "date", "echo", "exit", "grep", "hilfe", "hinweis", "history", "kill", "ls", "ps", "pwd", "status", "strings", "su", "whoami"}

// Complete vervollständigt die Eingabe. Gibt die neue Zeile und bei
// Mehrdeutigkeit die Kandidaten zurück.
func (g *Game) Complete(line string) (string, []string) {
	if g.pending != "" {
		return line, nil
	}
	i := strings.LastIndex(line, " ")
	word, head := line[i+1:], line[:i+1]
	if i < 0 && !strings.Contains(word, "/") {
		var c []string
		for _, b := range befehle {
			if strings.HasPrefix(b, word) {
				c = append(c, b)
			}
		}
		return vervollstaendige(line, head, "", c, nil)
	}
	dirPart, base := "", word
	if j := strings.LastIndex(word, "/"); j >= 0 {
		dirPart, base = word[:j+1], word[j+1:]
	}
	n, st := g.lookup(g.resolve(dirPart))
	if st != ok || !n.dir {
		return line, nil
	}
	var c []string
	dirs := map[string]bool{}
	for _, k := range n.kids {
		if strings.HasPrefix(k.name, base) && (!strings.HasPrefix(k.name, ".") || strings.HasPrefix(base, ".")) {
			c = append(c, k.name)
			dirs[k.name] = k.dir
		}
	}
	sort.Strings(c)
	return vervollstaendige(line, head, dirPart, c, dirs)
}

func vervollstaendige(line, head, dirPart string, c []string, dirs map[string]bool) (string, []string) {
	switch len(c) {
	case 0:
		return line, nil
	case 1:
		end := " "
		if dirs[c[0]] {
			end = "/"
		}
		return head + dirPart + c[0] + end, nil
	}
	pre := c[0]
	for _, s := range c[1:] {
		for !strings.HasPrefix(s, pre) {
			pre = pre[:len(pre)-1]
		}
	}
	return head + dirPart + pre, c
}

// Label liefert den sichtbaren Text eines antippbaren Elements.
func Label(kind, value string) string {
	if kind == "c" {
		return value
	}
	l := path.Base(value)
	if kind == "d" {
		l += "/"
	}
	return l
}
