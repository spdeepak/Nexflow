# Project Instructions

IMPORTANT: Before doing anything else, read and follow this file.

## Go Backend (`backend/`)

- Go 1.26.5
- Run `gofmt` and `go vet` on modified Go files.
- Run `go build ./...` to verify compilation.
- Do not introduce new dependencies without asking.
- When changing database code, explain the migration impact.
- If you need new fields from the structs in the `schema` package you can add those fields for those structs in the `schema.json` file, look carefully on how other structs are designed and use the same format to add new fields if custom data type is necessary.
- After modifying `schema.json` or SQL schemas or SQL queries, run `make generate` to regenerate code (sqlc, go-jsonschema, mockery).
- Run the tests at the end to make sure all tests are passing

## Wails Desktop App (`backend/cmd/app/`)

- Run GoLand/debug with the `wails-app` run config (uses `-tags production` + `CGO_LDFLAGS=-framework UniformTypeIdentifiers`).
- Run in dev mode from `make wals-dev`.
- Build the binary with `make wails-build`.