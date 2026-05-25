.PHONY: dev web build test test-int docker run clean

dev: web
	go run ./cmd/leaderboard

web:
	cd web && pnpm install && pnpm build

build: web
	go build -o bin/leaderboard ./cmd/leaderboard

test:
	go test ./internal/... -race
	cd web && pnpm test

test-int:
	go test -tags=integration ./test/integration/... -v

docker:
	docker build -t newapi-leaderboard:local .

run: docker
	docker run --rm -p 8080:8080 \
		-e MYSQL_DSN="$$MYSQL_DSN" \
		-e ADMIN_TOKEN=test \
		newapi-leaderboard:local

clean:
	rm -rf bin web/dist internal/embed/dist/assets/
