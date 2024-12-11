include .env

.PHONY: migrate-create migrate-up migrate-down

MYSQL_URL="mysql://${DB_USER}:${DB_PASSWORD}@tcp(${DB_HOST}:${DB_PORT})/${DB_NAME}?multiStatements=true"

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

migrate-up:
	migrate -path migrations -database ${MYSQL_URL} up

migrate-down:
	migrate -path migrations -database ${MYSQL_URL} down