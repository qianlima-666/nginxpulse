.PHONY: frontend frontend-mobile backend single dev clean

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)
BUILD_TIME ?= $(shell date "+%Y-%m-%d %H:%M:%S")
GIT_COMMIT ?= $(shell git rev-parse --short=7 HEAD 2>/dev/null || echo unknown)
LDFLAGS = -s -w -X 'github.com/qianlima-666/nginxpulse/internal/version.Version=$(VERSION)' -X 'github.com/qianlima-666/nginxpulse/internal/version.BuildTime=$(BUILD_TIME)' -X 'github.com/qianlima-666/nginxpulse/internal/version.GitCommit=$(GIT_COMMIT)'

frontend:
	cd webapp && pnpm install --frozen-lockfile && pnpm run build
	cd webapp_mobile && pnpm install --frozen-lockfile && pnpm run build

frontend-mobile:
	cd webapp_mobile && pnpm install --frozen-lockfile && pnpm run build

backend:
	go build -ldflags="$(LDFLAGS)" -o bin/nginxpulse ./cmd/nginxpulse/main.go

single:
	VERSION="$(VERSION)" BUILD_TIME="$(BUILD_TIME)" GIT_COMMIT="$(GIT_COMMIT)" ./scripts/build_single.sh

dev:
	./scripts/dev_local.sh

clean:
	rm -rf bin/nginxpulse internal/webui/dist internal/webui/dist_mobile webapp/dist webapp_mobile/dist
