import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

const AUTH_URL = __ENV.AUTH_URL || 'http://localhost:8080';
// Location and trip hit their HTTP REST ports directly (APISIX routes not yet configured locally)
const LOCATION_URL = __ENV.LOCATION_URL || 'http://localhost:8053';
const TRIP_URL = __ENV.TRIP_URL || 'http://localhost:8051';

const loginDuration = new Trend('auth_login_duration');
const registerFail = new Rate('auth_register_fail');
const locationPings = new Counter('location_pings');

export const options = {
  scenarios: {
    health_probes: {
      executor: 'constant-vus',
      vus: 10,
      duration: '20s',
      exec: 'health',
    },
    auth_burst: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5s', target: 20 },
        { duration: '12s', target: 20 },
        { duration: '3s', target: 0 },
      ],
      exec: 'authFlow',
    },
    driver_location_pings: {
      executor: 'constant-vus',
      vus: 50,
      duration: '20s',
      exec: 'driverLocation',
    },
    rider_trip_requests: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5s', target: 20 },
        { duration: '12s', target: 20 },
        { duration: '3s', target: 0 },
      ],
      exec: 'riderTrip',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.15'],
    auth_register_fail: ['rate<0.25'],
  },
};

export function health() {
  const res = http.get(`${AUTH_URL}/health`);
  check(res, { 'auth health 200': (r) => r.status === 200 });
  sleep(1);
}

export function authFlow() {
  const id = `${__VU}-${__ITER}-${Date.now()}`;
  const email = `load.${id}@example.com`;
  const payload = JSON.stringify({
    email,
    phone: `+1555${String(1000000 + (__VU * 1000 + __ITER)).slice(-7)}`,
    password: 'LoadTest99',
    full_name: 'Load Rider',
    role: 'RIDER',
  });
  const params = { headers: { 'Content-Type': 'application/json' } };

  group('register', () => {
    const res = http.post(`${AUTH_URL}/auth/register`, payload, params);
    registerFail.add(res.status !== 201);
    check(res, { 'register created or conflict': (r) => r.status === 201 || r.status === 409 });
  });

  group('login', () => {
    const start = Date.now();
    const res = http.post(
      `${AUTH_URL}/auth/login`,
      JSON.stringify({ email, password: 'LoadTest99' }),
      params
    );
    loginDuration.add(Date.now() - start);
    check(res, { 'login ok or unauthorized': (r) => r.status === 200 || r.status === 401 });
  });

  sleep(0.5);
}

export function driverLocation() {
  const payload = JSON.stringify({
    driverId: `drv_${__VU}`,
    latitude: 12.9716 + Math.random() * 0.01,
    longitude: 77.5946 + Math.random() * 0.01,
    heading: Math.floor(Math.random() * 360),
  });
  const res = http.post(`${LOCATION_URL}/api/v1/location/driver`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  locationPings.add(1);
  check(res, { 'location accepted': (r) => r.status < 500 });
  sleep(1);
}

export function riderTrip() {
  const pad = String(__VU).padStart(12, '0');
  const payload = JSON.stringify({
    riderId: `00000000-0000-0000-0000-${pad}`,
    pickupAddress: 'MG Road, Bengaluru',
    pickupLat: 12.9716,
    pickupLng: 77.5946,
    dropoffAddress: 'Koramangala, Bengaluru',
    dropoffLat: 12.9352,
    dropoffLng: 77.6245,
    vehicleType: 'SEDAN',
    paymentMethodId: 'pm_card_visa',
  });
  const res = http.post(`${TRIP_URL}/api/v1/trips`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res, { 'trip accepted': (r) => r.status < 500 });
  sleep(2);
}
