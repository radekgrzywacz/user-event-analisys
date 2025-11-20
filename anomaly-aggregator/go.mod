module anomaly-aggregator

go 1.24.2

require (
	github.com/jackc/pgx/v5 v5.7.6
	github.com/twmb/franz-go v1.20.4
	user-event-analisys/contracts v0.0.0
)

require github.com/pkg/errors v0.9.1 // indirect

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.12.0 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.30.0 // indirect
)

replace user-event-analisys/contracts => ../contracts
