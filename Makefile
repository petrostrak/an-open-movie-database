run:
	go run ./cmd/api

psql:
	psql ${OMDB_DB_DSN}

up:
	@echo 'Running up migrations..'
	migrate -path ./migrations -database ${OMDB_DB_DSN} up