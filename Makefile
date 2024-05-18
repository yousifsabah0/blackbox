build:
	@go build -o ./bin/main ./cmd/api/

run: build
	@./bin/main

up:
	goose -dir migrations postgres "user=postgres password=pa55word dbname=blackbox sslmode=disable host=localhost port=5432" up