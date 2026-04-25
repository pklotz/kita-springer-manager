.PHONY: run build frontend-build frontend-dev dev

run:
	go run ./cmd/server

build:
	go build -o bin/kita-springer ./cmd/server

frontend-build:
	cd frontend && npm run build

frontend-dev:
	cd frontend && npm run dev

# Full dev workflow: backend + frontend in parallel
dev:
	@echo "Starting backend on :9092 and frontend dev server on :5173"
	@trap 'kill 0' INT; \
	  go run ./cmd/server & \
	  cd frontend && npm run dev & \
	  wait
