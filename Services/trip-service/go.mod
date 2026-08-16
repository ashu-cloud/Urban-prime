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

replace (
	github.com/cab-booking/pkg => ../../pkg
	github.com/cab-booking/proto => ../../proto
)
