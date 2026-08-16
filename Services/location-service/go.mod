module github.com/cab-booking/location-service

go 1.22.0

require (
	github.com/cab-booking/pkg v0.0.0
	github.com/cab-booking/proto v0.0.0
	github.com/redis/go-redis/v9 v9.5.1
	github.com/twmb/franz-go v1.16.1
	google.golang.org/grpc v1.62.0
)

replace (
	github.com/cab-booking/pkg => ../../pkg
	github.com/cab-booking/proto => ../../proto
)
