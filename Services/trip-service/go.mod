module github.com/cab-booking/trip-service

go 1.22.0

require (
	github.com/cab-booking/pkg v0.0.0
	github.com/cab-booking/proto v0.0.0
	github.com/golang-migrate/migrate/v4 v4.17.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.5.4
	github.com/twmb/franz-go v1.16.1
	google.golang.org/grpc v1.62.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.19 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.7.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/crypto v0.20.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace (
	github.com/cab-booking/pkg => ../../pkg
	github.com/cab-booking/proto => ../../proto
)
