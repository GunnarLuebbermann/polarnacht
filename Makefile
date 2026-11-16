GOROOT := $(shell go env GOROOT)

.PHONY: web server test clean

## web: baut die Browser-Version nach web/
web:
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o web/game.wasm ./cmd/wasm
	cp "$(GOROOT)/lib/wasm/wasm_exec.js" web/

## server: startet SSH (:2222) und Web (:8080)
server: web
	go run ./cmd/server

test:
	go test ./...

clean:
	rm -f web/game.wasm web/wasm_exec.js
