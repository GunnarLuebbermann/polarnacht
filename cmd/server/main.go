// Polarnacht-Server: spielbar per SSH und im Browser.
//
//	go run ./cmd/server               SSH auf :2222, Web auf :8080
//	ssh -p 2222 localhost
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"golang.org/x/term"

	"github.com/gunnarluebbermann/polarnacht/engine"
)

func main() {
	sshAddr := flag.String("ssh", ":2222", "Adresse für den SSH-Server (leer = aus)")
	webAddr := flag.String("web", ":8080", "Adresse für die Web-Version (leer = aus)")
	webDir := flag.String("webdir", "web", "Ordner mit index.html, game.wasm, wasm_exec.js")
	hostKey := flag.String("hostkey", ".ssh/polarnacht_ed25519", "Pfad zum SSH-Hostschlüssel (wird erzeugt)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *webAddr != "" {
		srv := &http.Server{Addr: *webAddr, Handler: http.FileServer(http.Dir(*webDir))}
		go func() {
			log.Printf("Web-Version: http://localhost%s", *webAddr)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatal(err)
			}
		}()
		defer srv.Shutdown(context.Background())
	}

	if *sshAddr != "" {
		s, err := wish.NewServer(
			wish.WithAddress(*sshAddr),
			wish.WithHostKeyPath(*hostKey),
			wish.WithIdleTimeout(15*time.Minute),
			wish.WithMaxTimeout(2*time.Hour),
			wish.WithMiddleware(func(next ssh.Handler) ssh.Handler {
				return func(s ssh.Session) { spiele(s); next(s) }
			}),
		)
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			log.Printf("SSH: ssh -p %s localhost", (*sshAddr)[1:])
			if err := s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
				log.Fatal(err)
			}
		}()
		defer s.Shutdown(context.Background())
	}

	<-ctx.Done()
	log.Println("Fahre herunter …")
}

func spiele(s ssh.Session) {
	pty, winCh, ok := s.Pty()
	if !ok {
		fmt.Fprintln(s, "Polarnacht braucht ein Terminal. Verbinde dich mit: ssh -t …")
		return
	}
	t := term.NewTerminal(s, "")
	t.SetSize(pty.Window.Width, pty.Window.Height)
	go func() {
		for w := range winCh {
			t.SetSize(w.Width, w.Height)
		}
	}()

	g := engine.New()
	fmt.Fprint(t, "\n"+engine.ANSI(g.Intro())+"\n\n")
	t.AutoCompleteCallback = func(line string, pos int, key rune) (string, int, bool) {
		if key != '\t' || pos != len(line) {
			return "", 0, false
		}
		neu, _ := g.Complete(line)
		return neu, len(neu), true
	}

	for {
		var (
			line string
			err  error
		)
		if g.Secret() {
			line, err = t.ReadPassword(g.Prompt())
		} else {
			t.SetPrompt(engine.ANSI("«" + g.Prompt() + "»"))
			line, err = t.ReadLine()
		}
		if err != nil {
			return
		}
		out := g.Exec(line)
		if out.Clear {
			fmt.Fprint(t, "\x1b[2J\x1b[H")
		}
		if out.Text != "" {
			fmt.Fprint(t, engine.ANSI(out.Text)+"\n")
		}
		if out.Won || out.Quit {
			fmt.Fprint(t, "\n")
			if out.Quit {
				return
			}
		}
	}
}
