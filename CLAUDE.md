# Choixpeau

API minimaliste d'assignation de cohortes A/B, écrite en Go.

## Stack

- Go (net/http standard library, pas de framework)

## Build & Run

```bash
go build -o choixpeau .
./choixpeau
```

Le serveur écoute sur `:8080`.

## Tests

```bash
go test ./...
```
