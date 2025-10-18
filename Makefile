MIGRATE_SYSTEM = docker compose run --rm migrate-system -path /migrations -database "postgres://postgres:postgres@db:5432/user_event_analysis_db?sslmode=disable"

up:
	docker compose --profile dev up -d

down:
	docker compose --profile dev down

migrate-system-up:
	$(MIGRATE_SYSTEM) up 1

migrate-system-down:
	$(MIGRATE_SYSTEM) down 1

migrate-system-reset:
	$(MIGRATE_SYSTEM) down -all && $(MIGRATE_SYSTEM) up

migrate-system-force:
	$(MIGRATE_SYSTEM) force 1

migrate-system-version:
	$(MIGRATE_SYSTEM) version

seed:
	docker compose run --rm synthetic-data-generator

clean:
	docker compose down --volumes --remove-orphans
