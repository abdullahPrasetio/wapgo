.PHONY: run build cli-build cli-install test test-race coverage lint sec docker-up docker-down \
        docker-build docker-push migrate tidy check integration docs list release

# ── Service ────────────────────────────────────────────────────────────────────

run:
	go run cmd/api/main.go

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/api cmd/api/main.go

# ── CLI ────────────────────────────────────────────────────────────────────────

cli-build:
	CGO_ENABLED=0 go build -C cli \
		-ldflags="-s -w -X 'github.com/abdullahPrasetio/wapgo/cli/commands.Version=$(shell git describe --tags --match 'cli/v*' --always 2>/dev/null | sed 's|^cli/||' || echo dev)'" \
		-o ../bin/wapgo ./wapgo

cli-install:
	go install -C cli \
		-ldflags="-s -w -X 'github.com/abdullahPrasetio/wapgo/cli/commands.Version=$(shell git describe --tags --match 'cli/v*' --always 2>/dev/null | sed 's|^cli/||' || echo dev)'" \
		./wapgo

# ── Testing ────────────────────────────────────────────────────────────────────

test:
	go test ./...
	cd cli && go test ./...

test-race:
	go test -race ./...
	cd cli && go test -race ./...

integration:
	go test -tags=integration -v -timeout=120s ./internal/integration/...

coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@TOTAL=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,"",$$3); print $$3}'); \
	echo "Total coverage: $${TOTAL}%"; \
	if [ $$(echo "$${TOTAL} < 80" | bc) -eq 1 ]; then \
		echo "ERROR: coverage $${TOTAL}% is below 80% gate"; exit 1; \
	fi

# ── Code quality ───────────────────────────────────────────────────────────────

lint:
	golangci-lint run ./...
	cd cli && golangci-lint run ./...

sec:
	gosec -severity medium -confidence medium ./...
	govulncheck ./...

# ── Full local CI check (mirrors GitHub Actions pipeline) ─────────────────────

check: lint sec test-race coverage

# ── Docker ────────────────────────────────────────────────────────────────────

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker build \
		--tag wapgo:$(shell git describe --tags --always 2>/dev/null || echo dev) \
		--tag wapgo:latest \
		.

docker-push: docker-build
	docker push ghcr.io/abdullahprasetio/wapgo:latest

# ── Misc ───────────────────────────────────────────────────────────────────────

docs:
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
	@echo "Swagger docs generated → docs/swagger.json"

list:
	@echo "Generated domains:"
	@ls internal/usecase/*_usecase.go 2>/dev/null | sed 's|internal/usecase/||;s|_usecase.go||' | sort | sed 's/^/  • /'

migrate:
	go run cmd/api/main.go migrate

tidy:
	go mod tidy
	cd cli && go mod tidy

# ── Release ────────────────────────────────────────────────────────────────────
# Usage: make release V=v1.4.3
# Creates two tags required for go install to work:
#   v1.4.3     — root module tag  (used by docker, CI badge, etc.)
#   cli/v1.4.3 — CLI sub-module tag (required by: go install .../cli/wapgo@v1.4.3)
release:
	@[ -n "$(V)" ] || { echo "usage: make release V=vX.Y.Z"; exit 1; }
	@echo "Tagging $(V) and cli/$(V)..."
	git tag -a $(V) -m "release $(V)"
	git tag -a cli/$(V) -m "cli release $(V)"
	git push origin $(V) cli/$(V)
	@echo "Done. Verify: go install github.com/abdullahPrasetio/wapgo/cli/wapgo@$(V)"
