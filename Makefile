ifneq (,$(wildcard .env))
include .env
export $(shell sed -E 's/^[[:space:]]*\#.*//; s/=.*//' .env)
endif

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null)
COMPOSE := docker compose -f docker/compose.yaml
# Node runs in the compose service of the same name, so the host needs only Go and Docker.
NODE_RUN := $(COMPOSE) run --rm --no-deps -T --user $$(id -u):$$(id -g)
NODE := $(NODE_RUN) node
TAILWIND := npx tailwindcss -i internal/web/ui/app.css -o internal/web/static/app.css

.PHONY: dev test vet build docker css e2e e2e-open

.env:
	cp .env.example .env

# Terrain and canopy tiles for dev, see https://github.com/radiopath/testfiles
tmp/testfiles:
	git clone --depth 1 https://github.com/radiopath/testfiles $@

# Postgres, Valkey, Mailpit and the stylesheet watcher in Docker, the server in the foreground; Ctrl-C stops everything.
dev: .env node_modules tmp/testfiles
	go build -ldflags="-X main.version=$(VERSION)" -o bin/radiopath-dev ./cmd/radiopath
	@$(COMPOSE) up -d --wait db valkey mailpit
	@set -a; . ./.env; set +a; \
		docker rm -f radiopath-css >/dev/null 2>&1; \
		$(NODE_RUN) --name radiopath-css node $(TAILWIND) --watch=always >/dev/null 2>&1 & \
		./bin/radiopath-dev & pid=$$!; \
		trap 'kill $$pid 2>/dev/null; docker rm -f radiopath-css >/dev/null 2>&1; $(COMPOSE) stop db valkey mailpit' EXIT INT TERM; \
		for i in $$(seq 30); do curl -sf -o /dev/null http://localhost:8080/healthz && break; sleep 0.5; done; \
		echo devpassword | ./bin/radiopath-dev useradd dev >/dev/null; \
		echo "App: http://localhost:8080  login dev / devpassword, admin panel http://localhost:8080/admin (token in .env), mails http://localhost:8025"; \
		echo "Ctrl-C stops everything incl. Postgres"; \
		wait $$pid

test:
	go test ./...

vet:
	go vet ./...

build: css
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/radiopath ./cmd/radiopath

# Tailwind source is internal/web/ui/app.css; the generated app.css is committed so go build needs no node.
node_modules: package-lock.json
	$(NODE) npm ci --no-audit --no-fund
	@touch node_modules

css: node_modules
	$(NODE) $(TAILWIND) --minify

e2e: build
	e2e/e2e.sh

e2e-open: build
	E2E_OPEN=1 e2e/e2e.sh

docker:
	docker build -f docker/Dockerfile -t radiopath:dev .
