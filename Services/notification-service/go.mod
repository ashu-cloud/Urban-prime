module github.com/cab-booking/notification-service

go 1.22.0

require (
	github.com/cab-booking/pkg v0.0.0
	github.com/twmb/franz-go v1.16.1
)

require (
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/pierrec/lz4/v4 v4.1.19 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.7.0 // indirect
	golang.org/x/crypto v0.20.0 // indirect
)

replace github.com/cab-booking/pkg => ../../pkg
