.PHONY: dev api admin web build test clean

BIN := bin/bkkguide

# Run the Go API (seeded) and the Vite dev server together. Open http://localhost:5173
dev: web/node_modules web/dist
	@trap 'kill 0' EXIT; \
	BKK_DEV=1 go run ./cmd/server -seed & \
	(cd web && npm run dev) & \
	wait

api: web/dist
	BKK_DEV=1 go run ./cmd/server -seed

# Create an admin (the first one, or a way back in). Uses BKK_DB like the server.
admin: web/dist
	go run ./cmd/server -create-admin

web/node_modules: web/package.json
	cd web && npm install
	@touch $@

# go:embed needs web/dist to exist even before the first frontend build.
web/dist:
	mkdir -p web/dist && echo '<!doctype html><p>run make build</p>' > web/dist/index.html

# Single binary with the frontend embedded. Cross-compile for the VPS with:
#   make build GOOS=linux GOARCH=amd64
build: web/node_modules
	cd web && npm run build
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o $(BIN) ./cmd/server

test: web/dist
	go test ./...

clean:
	rm -rf bin web/dist
