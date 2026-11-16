//go:build js && wasm

// Brücke zwischen Engine und Browser. Stellt drei globale Funktionen bereit:
//
//	pnStart()          neue Partie, liefert JSON mit Intro
//	pnExec(zeile)      führt eine Zeile aus, liefert JSON
//	pnComplete(zeile)  Tab-Vervollständigung, liefert JSON
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/gunnarluebbermann/polarnacht/engine"
)

var g *engine.Game

type antwort struct {
	engine.Output
	Prompt string        `json:"prompt"`
	Secret bool          `json:"secret"`
	Status engine.Status `json:"status"`
}

func paket(o engine.Output) string {
	b, _ := json.Marshal(antwort{Output: o, Prompt: g.Prompt(), Secret: g.Secret(), Status: g.Status()})
	return string(b)
}

func main() {
	js.Global().Set("pnStart", js.FuncOf(func(js.Value, []js.Value) any {
		g = engine.New()
		return paket(engine.Output{Text: g.Intro()})
	}))
	js.Global().Set("pnExec", js.FuncOf(func(_ js.Value, a []js.Value) any {
		return paket(g.Exec(a[0].String()))
	}))
	js.Global().Set("pnComplete", js.FuncOf(func(_ js.Value, a []js.Value) any {
		line, opts := g.Complete(a[0].String())
		b, _ := json.Marshal(map[string]any{"line": line, "options": opts})
		return string(b)
	}))
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("polarnacht-bereit"))
	select {}
}
