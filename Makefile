.PHONY: help build run migrate-up migrate-down migrate-status migrate-create db-up db-down db-reset seed

help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Собрать приложение
	go build -o bin/rag-iishka ./cmd/rag-iishka

run: ## Запустить приложение
	go run ./cmd/rag-iishka/main.go

migrate-up: ## Применить миграции
	@if command -v goose >/dev/null 2>&1; then \
		goose -dir migrations postgres "postgres://postgres:postgres@localhost:5433/rag_iishka" up; \
	else \
		echo "goose не установлен. Установите: go install github.com/pressly/goose/v3/cmd/goose@latest"; \
	fi

migrate-down: ## Откатить миграции
	@if command -v goose >/dev/null 2>&1; then \
		goose -dir migrations postgres "postgres://postgres:postgres@localhost:5433/rag_iishka" down; \
	else \
		echo "goose не установлен. Установите: go install github.com/pressly/goose/v3/cmd/goose@latest"; \
	fi

migrate-status: ## Показать статус миграций
	@if command -v goose >/dev/null 2>&1; then \
		goose -dir migrations postgres "postgres://postgres:postgres@localhost:5433/rag_iishka" status; \
	else \
		echo "goose не установлен. Установите: go install github.com/pressly/goose/v3/cmd/goose@latest"; \
	fi


db-up: ## Запустить PostgreSQL в Docker
	docker-compose up -d postgres

db-down: ## Остановить PostgreSQL в Docker
	docker-compose down

db-reset: ## Сбросить базу данных (удалить и создать заново)
	docker-compose down -v
	docker-compose up -d postgres
	sleep 5
	$(MAKE) migrate-up

seed: ## Наполнить базу знаний примерами данных
	go run ./cmd/seed/main.go
