.PHONY: clean generate build run lint test dev complete wails-dev wails-build wails-run lint-backend test-backend test-backend-coverage

clean:
	find . -not -path './vendor/*' -name "*.gen.go" -type f -delete

generate: clean
	go tool sqlc generate
	go tool mockery

complete: clean
	go mod tidy
	go mod vendor
	generate

build:
	go build -o ../bin/server ./cmd/server

run:
	go run ./cmd/server

lint:
	GOFLAGS= go tool golangci-lint run ./...

test: lint
	go test -p 1 ./...

dev:
	go run ./cmd/server

lint-backend:
	GOFLAGS= go tool golangci-lint run ./...

test-backend: lint-backend
	go test -p 1 ./...

test-backend-coverage:
	go test -p 1 -coverprofile=coverage.out ./... && { head -n1 coverage.out; grep -vE '\.gen\.go:' coverage.out | tail -n +2; } > coverage.filtered.out && go tool cover -func=coverage.filtered.out | tail -n1 && rm coverage.out coverage.filtered.out

# Run the Wails desktop app in dev mode (with hot reload). Works from anywhere
# without depending on the working directory.
wails-dev:
	rm -rf app.db && rm -rf app.db-shm && rm -rf app.db-wal
	cd cmd/app/frontend && rm -rf wailsjs
	cd cmd/app && rm -rf build && mkdir -p build && cp frontend/appicon.png build/appicon.png
	cd cmd/app && go tool wails dev

# Build the Wails desktop app into a packaged macOS .app bundle (with the custom
# app icon from frontend/appicon.png). The output bundle is produced by the Wails
# CLI in cmd/app/build/bin/ and can be opened from anywhere.
wails-build:
	rm -rf app.db && rm -rf app.db-shm && rm -rf app.db-wal
	cd cmd/app/frontend && rm -rf wailsjs
	cd cmd/app && rm -rf build
	cd cmd/app && rm -rf build && mkdir -p build && cp frontend/appicon.png build/appicon.png
	cd cmd/app && CGO_LDFLAGS="-framework UniformTypeIdentifiers" go tool wails build
	mkdir -p cmd/app/build/bin/Nexflow.app/Contents/Resources/configs
	cp configs/config.json cmd/app/build/bin/Nexflow.app/Contents/Resources/configs/
	cp configs/secrets.json cmd/app/build/bin/Nexflow.app/Contents/Resources/configs/
	codesign --force --deep --sign - cmd/app/build/bin/Nexflow.app

# Launch the built Wails app binary directly, bypassing the macOS Gatekeeper
# check that silently blocks the packaged .app. Free of charge and needs no
# Apple Developer signing. Run `make wails-build` first.
wails-run:
	@test -x cmd/app/build/bin/Nexflow.app/Contents/MacOS/nexflow || { echo "Build first: make wails-build"; exit 1; }
	./cmd/app/build/bin/Nexflow.app/Contents/MacOS/nexflow
