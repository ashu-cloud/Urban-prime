module github.com/cab-booking/location-service

go 1.22.0

require (
	github.com/cab-booking/pkg v0.0.0
	github.com/cab-booking/proto v0.0.0
	github.com/redis/go-redis/v9 v9.5.1
	google.golang.org/grpc v1.62.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace (
	github.com/cab-booking/pkg => ../../pkg
	github.com/cab-booking/proto => ../../proto
)
