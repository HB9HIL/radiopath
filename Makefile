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
# The image follows @playwright/test in e2e/package.json, the one place Renovate updates.
PLAYWRIGHT_IMAGE := mcr.microsoft.com/playwright:v$(shell sed -n 's/.*"@playwright\/test": "\([^"]*\)".*/\1/p' e2e/package.json)-noble

.PHONY: dev test vet build docker css e2e

.env:
	cp .env.example .env

# Terrain and canopy tiles for dev and e2e, see https://github.com/radiopath/testfiles
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

# Playwright specs against the real image, every service in its own container on the compose network.
e2e: tmp/testfiles
	$(COMPOSE) --profile e2e up --build --detach --wait --force-recreate e2e-app
	rm -rf e2e/test-results
	docker run --rm --network radiopath_default --ipc=host --user $$(id -u):$$(id -g) -e HOME=/tmp -e CI \
		-v $(CURDIR)/e2e:/e2e -w /e2e $(PLAYWRIGHT_IMAGE) sh -c 'npm ci --no-audit --no-fund && npx playwright test'; \
		status=$$?; mkdir -p e2e/test-results; $(COMPOSE) --profile e2e logs e2e-app > e2e/test-results/app.log 2>&1; \
		$(COMPOSE) --profile e2e rm --stop --force e2e-app e2e-db e2e-mail >/dev/null 2>&1; exit $$status

docker:
	docker build -f docker/Dockerfile -t radiopath:dev .
