up:
	docker compose -f deployments/docker-compose.yml up -d
down:
	docker compose -f deployments/docker-compose.yml down
run-server:
	go run ./cmd/server
run-worker:
	go run ./cmd/worker
test:
	go test ./...
swagger:
	swag init -g cmd/server/main.go -o docs/swagger

e2e:
	bash scripts/e2e.sh