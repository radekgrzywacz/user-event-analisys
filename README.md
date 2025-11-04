# user-event-analisys

## Project goal
End-to-end platform for analyzing user behaviour in digital channels. Events are synthetically generated, ingested over HTTP into Kafka, processed for anomalies, persisted in PostgreSQL, and visualised in Grafana.

## Event flow architecture
- The synthetic generator submits HTTP requests to `/ingestor`, simulating both regular activity and adversarial scenarios.
- The HTTP ingestor transforms each payload into a record in the Kafka topic `events`.
- The `analyser` service consumes records, validates them using Redis (cache and statistics) and PostgreSQL (long-term storage).
- Redis keeps per-user context (recent events, Markov transitions, EMA); PostgreSQL stores historical events and anomalies.
- Grafana connects to the analytical database and exposes pre-provisioned dashboards for monitoring and investigations.

## Application services (Docker)
### synthetic-data-generator (`synthetic-data-generator`)
- Produces user traffic and anomalies, manages the user database in Postgres (`user_event_generator_db`), and exposes CLI flags for simulation parameters (`users`, `duration`, `concurrency`, `anomaly-rate`).
- The main orchestration and anomaly sampling loop lives in `synthetic-data-generator/cmd/main/main.go:18`.
- Individual scenarios (logins, brute force, account takeover, fraud, long sessions) are defined in `synthetic-data-generator/internal/event/scenarios_creator.go:10`.

### http-ingestor (`http-ingestor`)
- Lightweight HTTP service that accepts payloads and asynchronously publishes Kafka records to the `events` topic.
- The handler that builds Kafka records and applies timeouts is defined in `http-ingestor/cmd/main/main.go:44`.

### analyser (`analyser`)
- Kafka consumer performing a two-stage analysis per event: fast cache checks (known IPs, Markov) followed by statistical heuristics (EMA, time deviation).
- The central processing pipeline is `analyser/internal/analyser/analyser.go:29`.
- Cache updates, Markov histograms, and statistical feature preparation happen in `analyser/internal/analyser/cachedData.go:16`.
- EMA frequency deviation and hour-of-day heuristics are implemented in `analyser/internal/analyser/cachedData.go:131` and `analyser/internal/analyser/cachedData.go:215`.
- Markov transition anomaly detection (1st- and 2nd-order, with global fallbacks) is located in `analyser/internal/analyser/cachedData.go:247`.

### grafana (`grafana`)
- Grafana instance with provisioned datasources and dashboards (`grafana/provisioning`), default credentials `admin/admin`.
- The default dashboard (`grafana/provisioning/dashboards/default/events.json`) shows counts of events and anomalies from the analytical database.

## Supporting components
- PostgreSQL (`db`): generator database (`user_event_generator_db`) and analytical database (`user_event_analysis_db`). Creation script: `db/initdb/create_databases.sql`.
- Redis (`redis`): cache used by `analyser` for recent events, behavioural features, and metrics.
- Kafka + Zookeeper (`kafka`, `zookeeper`): event backbone; topic `events` is created by the `init-kafka` job.
- Database migrations: `migrate-generator` runs automatically for the generator; `migrate-system` is triggered manually via the Makefile.
- Grafana (`grafana`) and persistent volumes (`pgdata`, `redis-data`, `grafana-data`) preserve state across restarts.

## Docker & Makefile
### Prerequisites
- Docker Desktop / Docker Engine with Docker Compose v2.
- `make`.
- (Optional) Go 1.22+ if you want to run services locally outside Docker.

### First run
1. Ensure ports `5432`, `6379`, `8080-8082`, `3000`, `9092` are available.
2. Start the full developer stack:
   ```bash
   make up-dev
   ```
   *Launches all services tagged with the `dev` profile in `docker-compose.yaml`.*
3. (Optional) Apply analytical DB migrations if the database is empty:
   ```bash
   make migrate-system-up
   ```
4. Generate an initial batch of events (the generator starts with the stack; trigger it manually with `make seed`).
5. Verify the services: Grafana (`http://localhost:3000`, `admin/admin`), HTTP ingestor (`http://localhost:8081/healthcheck`), generator API (`http://localhost:8080`), analyser logs (`docker compose logs -f analyser`).

### Useful commands
- `make up-dev` / `make down-dev` – start/stop the developer stack.
- `make up-all` / `make down-all` – start/stop the full profile set.
- `make migrate-system-up` / `make migrate-system-down` – apply or rollback a single analytical migration step.
- `make migrate-system-reset` – roll all migrations down and back up from scratch.
- `make migrate-system-version` – show the current analytical schema version.
- `make seed` – run the generator once (handy when the stack is down).
- `make clean` – stop the stack and remove volumes (data loss warning).

## Local development
- Each service has its own `.env` file (for example `synthetic-data-generator/.env`); when `RUNNING_IN_DOCKER` is unset, configs are loaded from these files.
- Run the generator locally with `go run ./cmd/main -duration 30 -users 50 -anomaly-rate 0.1` to make the duration infinite just set it to 0.
- Running the analyser locally requires Kafka, Redis, and Postgres; start them via Docker (`make up-dev`) and launch the Go binary from your host.
- If you change Kafka topics/ports, update both `docker-compose.yaml` and the `.env` files in `http-ingestor` and `analyser`.

## Key code locations
- `synthetic-data-generator/cmd/main/main.go:18` – scenario selection and concurrency control for the simulation.
- `synthetic-data-generator/internal/event/scenarios_creator.go:10` – definitions of normal and anomalous scenarios (brute force, takeover, fraud, long sessions).
- `http-ingestor/cmd/main/main.go:44` – HTTP handler that converts requests into Kafka records with timeout handling.
- `analyser/internal/analyser/analyser.go:29` – main processing pipeline (cache → statistics → persistence).
- `analyser/internal/analyser/cachedData.go:16` – user cache updates (72h history, IP/UA/country sets, Markov histograms).
- `analyser/internal/analyser/cachedData.go:131` – EMA-based frequency deviation per event type.
- `analyser/internal/analyser/cachedData.go:215` – unusual hour-of-day detection.
- `analyser/internal/analyser/cachedData.go:247` – first/second-order Markov anomaly detection with global fallbacks.
- `grafana/provisioning/dashboards/default/events.json:1` – prebuilt dashboard for monitoring event and anomaly counts.
