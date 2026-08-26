import http from 'k6/http';
import { check, sleep } from 'k6';

// This test simulates 500 concurrent drivers pinging their location every 3 seconds,
// while 100 riders simultaneously request trips.
export const options = {
  scenarios: {
    driver_location_pings: {
      executor: 'constant-vus',
      vus: 500,
      duration: '30s',
      exec: 'driverLocation',
    },
    rider_trip_requests: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 },
        { duration: '15s', target: 100 },
        { duration: '5s', target: 0 },
      ],
      exec: 'riderTrip',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
    http_req_failed: ['rate<0.01'],   // Error rate should be less than 1%
  },
};

const BASE_URL = 'http://localhost:9080/api/v1';

export function driverLocation() {
  const driverId = `drv_${__VU}`;
  
  // Simulate slightly moving around
  const lat = 40.7484 + (Math.random() * 0.01);
  const lng = -73.9851 + (Math.random() * 0.01);
  
  const payload = JSON.stringify({
    driverId: driverId,
    latitude: lat,
    longitude: lng,
    heading: Math.floor(Math.random() * 360),
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      // In a real load test we'd need valid JWTs, assuming APISIX allows this or we bypass
      'Authorization': 'Bearer mock_driver_jwt',
    },
  };

  const res = http.post(`${BASE_URL}/location/driver`, payload, params);

  check(res, {
    'location ping status is 200': (r) => r.status === 200,
  });

  // Drivers ping every 3 seconds
  sleep(3);
}

export function riderTrip() {
  const riderId = `rid_${__VU}`;
  
  const payload = JSON.stringify({
    riderId: riderId,
    pickupAddress: "Times Square, NY",
    pickupLat: 40.7580,
    pickupLng: -73.9855,
    dropoffAddress: "Central Park, NY",
    dropoffLat: 40.7812,
    dropoffLng: -73.9665,
    vehicleType: "SEDAN",
    fareAmount: 2500, // $25.00
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer mock_rider_jwt',
    },
  };

  const res = http.post(`${BASE_URL}/trips`, payload, params);

  check(res, {
    'trip creation status is 200': (r) => r.status === 200,
  });

  // Riders don't spam trips, they request once and wait
  sleep(10);
}
