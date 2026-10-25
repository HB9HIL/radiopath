ifneq (,$(wildcard .env))
include .env
export $(shell sed -E 's/^[[:space:]]*\#.*//; s/=.*//' .env)
endif

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null)
ARCH := $(shell docker version -f '{{.Server.Arch}}' 2>/dev/null)
COMPOSE := docker compose -f docker/compose.yaml
# Node runs in the compose service of the same name, so the host needs only Go and Docker.
NODE_RUN := $(COMPOSE) run --rm --no-deps -T --user $$(id -u):$$(id -g)
NODE := $(NODE_RUN) node
WEB := $(NODE_RUN) -w /src/web node
TAILWIND := npx tailwindcss -i internal/web/ui/app.css -o internal/web/static/app.css
# The image follows @playwright/test in e2e/package.json, the one place Renovate updates.
PLAYWRIGHT_IMAGE := mcr.microsoft.com/playwright:v$(shell sed -n 's/.*"@playwright\/test": "\([^"]*\)".*/\1/p' e2e/package.json)-noble

.PHONY: dev test vet build docker css web check api e2e e2e-run FORCE

.env:
	cp .env.example .env

# Terrain and canopy tiles for dev and e2e, see https://github.com/radiopath/testfiles
tmp/testfiles:
	git clone --depth 1 https://github.com/radiopath/testfiles $@

# Postgres, Valkey, Mailpit and the stylesheet watcher in Docker, the server in the background and the Vite dev server
# (also in Docker) in the foreground; Ctrl-C stops everything.
dev: .env node_modules web/node_modules internal/webui/dist/index.html tmp/testfiles
	go build -ldflags="-X main.version=$(VERSION)" -o bin/radiopath-dev ./cmd/radiopath
	@$(COMPOSE) up -d --wait db valkey mailpit
	@set -a; . ./.env; set +a; \
		docker rm -f radiopath-css radiopath-vite >/dev/null 2>&1; \
		$(NODE_RUN) --name radiopath-css node $(TAILWIND) --watch=always --minify >/dev/null 2>&1 & \
		RADIOPATH_BASE_URL=http://localhost:5173 ./bin/radiopath-dev & pid=$$!; \
		trap 'kill $$pid 2>/dev/null; docker rm -f radiopath-css radiopath-vite >/dev/null 2>&1; $(COMPOSE) stop db valkey mailpit' EXIT INT TERM; \
		for i in $$(seq 30); do curl -sf -o /dev/null http://localhost:8080/healthz && break; sleep 0.5; done; \
		echo devpassword | ./bin/radiopath-dev useradd dev >/dev/null; \
		echo "App: http://localhost:5173  login dev / devpassword, admin panel http://localhost:5173/admin (token in .env), mails http://localhost:8025"; \
		echo "Ctrl-C stops everything incl. Postgres"; \
		$(COMPOSE) run --rm --service-ports --name radiopath-vite --user $$(id -u):$$(id -g) vite

test: internal/webui/dist/index.html
	go test ./...

vet: internal/webui/dist/index.html
	go vet ./...

build: css web
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/radiopath ./cmd/radiopath

# Tailwind source is internal/web/ui/app.css; the generated app.css is committed so go build needs no node.
node_modules: package-lock.json
	$(NODE) npm ci --no-audit --no-fund
	@touch node_modules

css: node_modules
	$(NODE) $(TAILWIND) --minify

web/node_modules: web/package-lock.json
	$(WEB) npm ci --no-audit --no-fund
	@touch web/node_modules

# The SPA into internal/webui/dist, which the binary embeds.
web: web/node_modules
	$(WEB) npm run build

# go:embed needs dist to exist; Go tools that do not need the SPA get this empty placeholder.
internal/webui/dist/index.html:
	mkdir -p internal/webui/dist && touch $@

check: web/node_modules
	$(WEB) npm run check

# The OpenAPI description and the TypeScript types the SPA builds on; run after changing an API operation and commit both.
api: web/node_modules internal/webui/dist/index.html
	go run ./cmd/radiopath openapi > web/openapi.json
	$(WEB) npm run api

# Static Linux binaries, packed by docker/ci.Dockerfile into the e2e and the release image.
bin/radiopath-linux-%: FORCE
	CGO_ENABLED=0 GOOS=linux GOARCH=$* go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o $@ ./cmd/radiopath

# Builds the SPA and the binary for the Docker host and runs the Playwright specs against the image packing it.
e2e: web bin/radiopath-linux-$(ARCH) e2e-run

# The specs against bin/radiopath-linux-* as it is; CI runs this with the binary of its binary job.
# Every service in its own container on the compose network.
e2e-run: tmp/testfiles
	$(COMPOSE) --profile e2e up --build --detach --wait --force-recreate e2e-app
	rm -rf e2e/test-results
	docker run --rm --network radiopath_default --ipc=host --user $$(id -u):$$(id -g) -e HOME=/tmp -e CI \
		-v $(CURDIR)/e2e:/e2e -w /e2e $(PLAYWRIGHT_IMAGE) sh -c 'npm ci --no-audit --no-fund && npx playwright test'; \
		status=$$?; mkdir -p e2e/test-results; $(COMPOSE) --profile e2e logs e2e-app > e2e/test-results/app.log 2>&1; \
		$(COMPOSE) --profile e2e rm --stop --force e2e-app e2e-db e2e-mail >/dev/null 2>&1; exit $$status

docker:
	docker build -f docker/Dockerfile -t radiopath:dev .
