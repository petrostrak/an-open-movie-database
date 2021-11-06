# Create the new confirm target.
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

run/api:
	go run ./cmd/api

db/psql:
	psql ${OMDB_DB_DSN}

# make migration name=create_example_table
db/migration/new:
	@echo 'Creating migration files for ${name}..'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

# Include it as prerequisite.
db/migrations/up: confirm
	@echo 'Running up migrations..'
	migrate -path ./migrations -database ${OMDB_DB_DSN} up