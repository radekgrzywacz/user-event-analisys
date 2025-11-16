module http-ingestor

go 1.24.2

require (
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/twmb/franz-go v1.19.5 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.11.2 // indirect
	user-event-analisys/contracts v0.0.0
)

replace user-event-analisys/contracts => ../contracts
