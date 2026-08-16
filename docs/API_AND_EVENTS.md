# API Contracts & Event Streaming Specifications

---

## 1. gRPC Services

### Trip Service (`proto/trip/v1/trip.proto`)
- `CreateTrip(CreateTripRequest) returns (CreateTripResponse)`
- `GetTrip(GetTripRequest) returns (GetTripResponse)`
- `CancelTrip(CancelTripRequest) returns (CancelTripResponse)`

### Driver Service (`proto/driver/v1/driver.proto`)
- `MatchDriver(MatchDriverRequest) returns (MatchDriverResponse)`
- `UpdateDriverStatus(UpdateDriverStatusRequest) returns (UpdateDriverStatusResponse)`

### Location Service (`proto/location/v1/location.proto`)
- `UpdateLocation(UpdateLocationRequest) returns (UpdateLocationResponse)`
- `GetNearbyDrivers(GetNearbyDriversRequest) returns (GetNearbyDriversResponse)`

### Payment Service (`proto/payment/v1/payment.proto`)
- `AuthorizeHold(AuthorizeHoldRequest) returns (AuthorizeHoldResponse)`
- `ReleaseHold(ReleaseHoldRequest) returns (ReleaseHoldResponse)`
- `CapturePayment(CapturePaymentRequest) returns (CapturePaymentResponse)`

---

## 2. Kafka Event Topics

| Topic Name | Key | Payload | Description |
| :--- | :--- | :--- | :--- |
| `driver.location.v1` | `driver_id` | `{ driver_id, lat, lon, heading, timestamp }` | Firehose of driver 3s GPS location updates |
| `trip.events.v1` | `trip_id` | `{ trip_id, rider_id, driver_id, status, fare }` | Trip lifecycle state change events |
| `driver.match.v1` | `match_id` | `{ trip_id, driver_id, action: OFFERED|ACCEPTED|DECLINED }` | Match dispatch lifecycle events |
| `payment.events.v1` | `transaction_id` | `{ transaction_id, trip_id, status, amount }` | Payment hold / capture / refund events |

---

## 3. Centrifugo WebSocket Channels

- **Rider Channel**: `rider:{rider_id}`
  - Receives: `driver.location.updated`, `trip.status.changed`, `driver.assigned`.
- **Driver Channel**: `driver:{driver_id}`
  - Receives: `ride.offered`, `trip.cancelled`, `rider.location.updated`.
