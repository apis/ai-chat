set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

npm-version := `npm -v`
go-version := `go version`
set-echo-color := BOLD + "\\e[93m"
unset-echo-color := NORMAL

alias a := all
alias b := build

# full application build including dependencies
all: _npm-version _go-version web-full app

# build web assets and go binaries
build: web app

# build go binaries
app:
  @echo "{{set-echo-color}}*** building go binaries ***{{unset-echo-color}}"
  go fmt ./...
  go mod tidy
  go build -v ./cmd/ricky-bot
  go build -v ./cmd/calculator
  go build -v ./cmd/calculator-mcp

# just install web dependencies
[working-directory: "web/frontend"]
web-deps:
  @echo "{{set-echo-color}}*** installing npm libraries ***{{unset-echo-color}}"
  npm install

# web assets build
[working-directory: "web/frontend"]
web:
  @echo "{{set-echo-color}}*** running npm build ***{{unset-echo-color}}"
  npm run build

# full web assets build including dependencies
[working-directory: "web/frontend"]
web-full: web-deps web

_npm-version:
  @echo "{{set-echo-color}}*** npm version {{npm-version}} ***{{unset-echo-color}}"

_go-version:
  @echo "{{set-echo-color}}*** {{go-version}} ***{{unset-echo-color}}"
