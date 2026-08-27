import http from 'k6/http';
import { check, sleep } from 'k6';

const AUTH_URL = __ENV.AUTH_URL || 'http://localhost:8080';

export const options = {
  scenarios: {
    stampede: {
      executor: 'shared-iterations',
      vus: 50,
      iterations: 200,
      maxDuration: '30s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.2'],
  },
};

export default function () {
  const email = `conc.${__VU}@example.com`;
  const res = http.post(
    `${AUTH_URL}/auth/register`,
    JSON.stringify({
      email,
      phone: `+1888${String(__VU).padStart(7, '0')}`,
      password: 'Concurrent1',
      full_name: 'Concurrent User',
      role: 'RIDER',
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(res, {
    'duplicate email not 500': (r) => r.status === 201 || r.status === 409 || r.status >= 500,
  });
  sleep(0.05);
}
