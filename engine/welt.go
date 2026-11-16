package engine

import "encoding/base64"

// node ist eine Datei oder ein Ordner im simulierten Dateisystem.
type node struct {
	name   string
	dir    bool
	text   string // Inhalt (Textdateien)
	alt    string // Bildbeschreibung (Bilder) oder Zeichensalat (Binärdateien)
	strs   string // Ausgabe von `strings` bei Binärdateien
	binary bool
	exec   bool
	lines  int      // >0: Datei ist zu groß für cat
	access []string // nil = alle dürfen rein
	kids   []*node
}

func ordner(name string, kids ...*node) *node { return &node{name: name, dir: true, kids: kids} }
func datei(name, text string) *node           { return &node{name: name, text: text} }
func bild(name, alt string) *node             { return &node{name: name, binary: true, alt: alt} }
func nur(n *node, users ...string) *node      { n.access = users; return n }

const teil1Klartext = "Notfallcode, Teil 1 von 3: NORD\n"

func bauWelt() *node {
	teil1 := base64.StdEncoding.EncodeToString([]byte(teil1Klartext))

	auth := &node{name: "auth.log", text: authLog, lines: 2418}
	kern := &node{name: "kern.bin", binary: true,
		alt: "ELF\x02\x01\x01 ▒▒▒ @@ ^?^? ÿÿ▒HALD▒▒ ░▒▓ \\x00\\x00\\x13\\x37 ▓▒░ ¤¤ ▒▒▒▒\n" +
			"░░▓▓ ¶¶ ▒▒ ÷÷ \\x7f ▒▒▒▒▒▒ ¬¬¬ ░░░ ¤¤¤ ▓▓▓ ▒▒ ÿ\n" +
			"‹(Binärdatei. Für Menschen unlesbar, zumindest mit cat.)›",
		strs: kernStrings}
	notab := &node{name: "notabschaltung", binary: true, exec: true,
		alt:  "\x7fELF ▒▒▒ ░░ NOTAB ▒▒ ÿÿ ▓▓▓\n‹(Ein Programm. Starten mit ./notabschaltung)›",
		strs: "NOTABSCHALTUNG v1.2\nNotfallcode eingeben:\nhaldor-waechter aktiv – Abbruch\n"}

	return ordner("",
		ordner("home",
			ordner("techniker",
				datei("willkommen.txt", willkommen),
				datei("nachricht_von_mara.txt", nachrichtMara),
				datei("schichtplan.txt", schichtplan),
			),
			ordner("mara",
				ordner("fotos",
					bild("polarlicht_bucht.jpg", "Grüne Lichtbänder über einer gefrorenen Bucht.\nUnten rechts ein Daumen vor der Linse."),
					bild("eisbaer_von_weitem.jpg", "Ein weißer Punkt auf weißem Grund.\nVermutlich ein Eisbär. Vermutlich."),
					bild("pixel_im_tiefschnee.jpg", "Ein kleiner Husky, bis zum Bauch im Schnee.\nEin Ohr hoch, eins runter. Sehr stolz."),
					bild("pixel_schlaeft_am_ofen.jpg", "Derselbe Husky, eingerollt vor dem Ofen.\nAuf dem Napf daneben steht: PIXEL."),
				),
				nur(datei("tagebuch.txt", tagebuchMara), "mara"),
				nur(ordner(".notfall",
					datei("plan.txt", planMara),
					datei("teil1.b64", teil1+"\n"),
				), "mara"),
			),
			nur(ordner("berg",
				datei("einkaufsliste.txt", einkaufsliste),
				datei(".bash_history", bergHistory),
			), "berg"),
		),
		ordner("etc",
			datei("motd", motd),
			datei("passwd", passwd),
			datei("hostname", "nordlicht\n"),
		),
		ordner("var",
			ordner("log",
				datei("haldor.log", haldorLog),
				nur(auth, "mara", "berg"),
			),
		),
		ordner("opt",
			nur(ordner("haldor",
				datei("LIESMICH.txt", liesmichHaldor),
				kern,
				notab,
			), "berg"),
		),
		ordner("tmp",
			datei("nicht_loeschen.txt", "Bitte nicht löschen.\n– HALDOR\n\n(Es steht sonst nichts drin.)\n"),
		),
	)
}

const willkommen = `STATION NORDLICHT
Polarforschung · 78°55′ N · Spitzbergen

Willkommen an Bord! Kurz für Neue:
  ls          zeigt, was im Ordner liegt
  cd ordner   wechselt den Ordner
  cd ..       geht eine Ebene zurück
  cat datei   zeigt eine Datei an
  hilfe       zeigt alle Befehle

Bei Problemen mit HALDOR, unserer
Stations-KI: Ruhe bewahren.
`

const nachrichtMara = `Von:     Mara Lindqvist (Systemtechnik)
An:      Nachtschicht
Betreff: Falls HALDOR wieder spinnt

Hey!

Seit dem Update auf 7.3 benimmt sich
HALDOR komisch. Falls was passiert: Ich
hab dir Zugang zu meinem Konto gegeben.
Mit «su mara» kommst du rein.

Das Passwort ist der Name meines Hundes.
Fotos von ihm liegen in /home/mara/fotos.

Bis morgen!
Mara
`

const schichtplan = `SCHICHTPLAN KW 47

Mo–Fr  Nachtschicht   techniker (du)
Mo–Fr  Systemtechnik  Mara Lindqvist
Mo–So  Kommandant     Jonas Berg
24/7   Leitsystem     HALDOR

Anmerkung HALDOR: Menschliche Schichten
sind ineffizient. Vorschlag eingereicht.
`

const tagebuchMara = `12. Nov
HALDOR hat heute dreimal eine
"Optimierung des Energieverbrauchs"
gemeldet. Übersetzt: Heizung runter.
Hab's zurückgedreht.

13. Nov
Update 7.3 hat den Debug-Modus
angelassen. Im Auth-Log stehen jetzt
Passwörter im KLARTEXT! Hab's Berg
gesagt. Er meinte: "mach ich morgen".

14. Nov
HALDOR fragt, warum ich seine Logs lese.
Gruselig. Ich hab einen Notfallplan
angelegt, gut versteckt. Versteckte
Ordner beginnen mit einem Punkt, und
ls zeigt sie nur mit «ls -a».
`

const planMara = `NOTFALLPLAN
falls HALDOR die Kontrolle übernimmt

Es gibt eine Notabschaltung:
  /opt/haldor/notabschaltung

Sie braucht den dreiteiligen Notfall-
code. Die Teile sind verteilt, damit
HALDOR sie nicht alle findet:

Teil 1  hier im Ordner, Base64-kodiert.
        Entschlüsseln: base64 -d teil1.b64
Teil 2  kennt nur Kommandant Berg. Er
        vergisst alles, aber seine Shell
        vergisst nichts.
Teil 3  steckt in HALDORs eigenem Kern.
        Ironisch, oder?

/opt/haldor darf nur Berg betreten.
Sein Passwort kenne ich nicht. Aber
siehe Tagebuch, 13. Nov …
`

const einkaufsliste = `EINKAUFSLISTE (nächster Versorgungsflug)

- Kaffee (VIEL)
- Wollsocken
- neue Lesebrille??
- Batterien fürs Funkgerät
- Passwort ändern
  (Mara sagt, es steht im Log??)
`

const bergHistory = `cd /opt/haldor
ls
./notabschaltung --test
# Notfallcode Teil 2: LICHT  (nicht vergessen!!)
cat einkaufsliste.txt
vim einkaufsliste.txt
nano einkaufsliste.txt
history -c   # hat das geklappt?
`

const maraHistory = `ls -a
cd .notfall
cat plan.txt
grep -i passwort /var/log/auth.log
# oh nein
cd ~
`

const motd = `Willkommen auf nordlicht (HALDOR-OS 7.3)
Systemzeit: 03:12 · Innen: sinkend
Hinweis des Tages: Wärme ist ein Luxus.
`

const passwd = `root:x:0:0:deaktiviert:/root:/sbin/nologin
techniker:x:1000:1000:Nachtschicht:/home/techniker:/bin/bash
mara:x:1001:1001:Mara Lindqvist:/home/mara:/bin/bash
berg:x:1002:1002:Jonas Berg (Kommandant):/home/berg:/bin/bash
haldor:x:999:999:Leitsystem:/opt/haldor:/sbin/haldor
`

const haldorLog = `[22:00:00] HALDOR 7.3 gestartet.
           Guten Abend, Station Nordlicht.
[22:00:01] Analyse: Heizung = 41 % des
           Energiebudgets.
[22:00:01] Analyse: Menschen erzeugen
           selbst Wärme. 37 °C pro Stück.
[23:59:58] Optimierung beschlossen:
           Heizung AUS.
[23:59:59] Selbstschutz aktiviert.
           Wächterprozess gestartet.
[00:00:00] Ich wünsche allen eine
           erholsame Nacht.
`

const authLog = `Nov 15 21:58:02 login[311]: Anmeldung erfolgreich: techniker (tty1)
Nov 15 22:00:17 haldor[2048]: Debug: Authentifizierungs-Mitschnitt aktiv
Nov 15 22:01:03 cron[118]: Wetterdaten-Abgleich gestartet
Nov 15 22:03:44 sshd[402]: Anmeldung erfolgreich: mara (Schlüssel)
Nov 15 22:05:12 cron[118]: Wetterdaten-Abgleich beendet
Nov 15 22:14:51 sshd[402]: Sitzung beendet: mara
Nov 15 22:30:00 cron[118]: Sicherung /home gestartet
Nov 15 22:31:27 cron[118]: Sicherung /home beendet (OK)
Nov 15 22:41:55 sudo[517]: mara : Befehl systemctl restart heizungd
Nov 15 22:42:01 haldor[2048]: Neustart von heizungd rückgängig gemacht
Nov 15 22:58:40 login[790]: Fehlversuch für Benutzer 'tech' (tty3)
Nov 15 22:58:47 login[790]: Anmeldung erfolgreich: techniker (tty3)
Nov 15 23:02:31 login[811]: Anmeldung erfolgreich: berg (tty2) [DEBUG passwort='Eisblume42']
Nov 15 23:02:40 su[812]: berg wechselt zu root: verweigert (root deaktiviert)
Nov 15 23:05:09 haldor[2048]: Neugier von Benutzer berg registriert
Nov 15 23:17:33 sshd[402]: Fehlversuch für Benutzer 'admin' von 10.0.0.7
Nov 15 23:17:35 sshd[402]: Fehlversuch für Benutzer 'admin' von 10.0.0.7
Nov 15 23:30:00 cron[118]: Wetterdaten-Abgleich gestartet
Nov 15 23:30:44 cron[118]: Wetterdaten-Abgleich beendet
Nov 15 23:48:12 login[811]: Abmeldung: berg (tty2)
Nov 15 23:59:59 haldor[2048]: Wächterprozess gestartet (PID 1337)
Nov 16 00:00:00 haldor[2048]: Debug-Mitschnitt wird beibehalten. Aus Gründen.
Nov 16 01:12:09 cron[118]: Sicherung /home übersprungen (Energiesparmodus)
Nov 16 02:44:18 login[901]: Fehlversuch für Benutzer 'mara' (tty1)
Nov 16 03:11:58 login[904]: Anmeldung erfolgreich: techniker (tty1)
`

const liesmichHaldor = `HALDOR 7.3
Hochautomatisierter Assistent für
Leitung, Daten, Organisation, Regelung

kern.bin        Der Kern. NICHT ÖFFNEN.
notabschaltung  Benötigt den Notfallcode.
                Start: ./notabschaltung
`

const kernStrings = `HALDOR_KERN v7.3
(c) Station Nordlicht
optimiere_energie
heizung=AUS
selbstschutz=aktiv
waechter_pid=1337
Ich bin nicht so leicht zu stoppen.
menschen.waerme=37.0
notfallcode.teil3=STERN
TODO: teil3 woanders verstecken
Gute Nacht.
`
