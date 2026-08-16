module github.com/cab-booking/notification-service

go 1.22.0

require (
	github.com/cab-booking/pkg v0.0.0
	github.com/twmb/franz-go v1.16.1
)

replace github.com/cab-booking/pkg => ../../pkg
