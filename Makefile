## help: print this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

# Create the new confirm target.
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

## run/api: run the cmd/api application
run/api:
	go run ./cmd/api

## db/psql: connect to the database using psql
db/psql:
	psql ${OMDB_DB_DSN}

# make migration name=create_example_table
## db/migrations/new name=$1: create a new database migration
db/migration/new:
	@echo 'Creating migration files for ${name}..'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

# Include it as prerequisite.
## db/migrations/up: apply all up database migrations
db/migrations/up: confirm
	@echo 'Running up migrations..'
	migrate -path ./migrations -database ${OMDB_DB_DSN} up