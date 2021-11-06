run/api:
	go run ./cmd/api

db/psql:
	psql ${OMDB_DB_DSN}

# make migration name=create_example_table
db/migration/new:
	@echo 'Creating migration files for ${name}..'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

db/migrations/up:
	@echo 'Running up migrations..'
	migrate -path ./migrations -database ${OMDB_DB_DSN} up