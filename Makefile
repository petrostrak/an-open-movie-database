run:
	go run ./cmd/api

psql:
	psql ${OMDB_DB_DSN}

# make migration name=create_example_table
migration:
	@echo 'Creating migration files for ${name}..'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

up:
	@echo 'Running up migrations..'
	migrate -path ./migrations -database ${OMDB_DB_DSN} up