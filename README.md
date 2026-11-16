# Polarnacht

Ein Escape-Room-Spiel im Terminal. Spielbar per **SSH** und im **Browser**, auch am Handy.

> Station Nordlicht, 78°55′ N, 03:12 Uhr. Die Stations-KI HALDOR hat die Heizung abgeschaltet.
> Draußen −38 °C, drinnen wird es mit jedem Befehl kälter. Finde den dreiteiligen Notfallcode
> und schalte HALDOR ab.

Unterwegs lernt man echte Shell-Befehle: `ls -a`, `cd`, `cat`, `grep`, `base64 -d`, `strings`,
`history`, `ps`, `kill`, `su`. Wer feststeckt, tippt `hinweis`.

## Spielen

**Im Browser / am Handy:** Dateien, Ordner und Befehle in der Ausgabe lassen sich antippen,
unten gibt es Schnelltasten (`ls`, `cd ..`, Tab, `hinweis` …). Der Spielstand wird lokal im
Browser gespeichert.

**Per SSH:**

```sh
make server             # baut die Web-Version und startet beides
ssh -p 2222 localhost   # SSH
open http://localhost:8080   # Browser
```

Optionen: `go run ./cmd/server -ssh :2222 -web :8080 -webdir web -hostkey .ssh/polarnacht_ed25519`
(eine Adresse leer lassen schaltet den jeweiligen Teil ab).

## Aufbau

```
engine/       Spiellogik (Dateisystem, Befehle, Rätsel, Hinweise) – ohne Ein-/Ausgabe
cmd/server/   SSH-Server (charmbracelet/wish) + statischer Webserver
cmd/wasm/     Brücke Engine ↔ Browser (Go → WebAssembly)
web/          Handy-taugliche Oberfläche (index.html)
```

Die Engine wird von beiden Oberflächen genutzt. Texte enthalten ein kleines Markup
(`⟦d|ordner⟧` antippbar, `«…»` hervorgehoben, `⟪…⟫` HALDOR spricht, `‹…›` gedämpft),
das der SSH-Server in ANSI-Farben und die Web-Version in HTML übersetzt.

## Entwicklung

```sh
make test   # spielt u. a. die komplette Rätselkette einmal durch
make web    # baut web/game.wasm + web/wasm_exec.js
```

Ein Push auf `main` baut die Web-Version und veröffentlicht sie über GitHub Pages
(Pages-Quelle in den Repo-Einstellungen auf „GitHub Actions“ stellen).

**Spoiler:** Die vollständige Lösung steht in `engine/engine_test.go`.
