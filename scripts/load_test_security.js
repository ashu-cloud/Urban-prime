import http from 'k6/http';
import { check, sleep } from 'k6';

const AUTH_URL = __ENV.AUTH_URL || 'http://localhost:8080';

export const options = {
  vus: 20,
  duration: '15s',
  thresholds: {
    checks: ['rate>0.9'],
  },
};

function post(path, body) {
  return http.post(`${AUTH_URL}${path}`, JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json' },
  });
}

export default function () {
  const admin = post('/auth/register', {
    email: `admin.${__VU}.${__ITER}@evil.com`,
    phone: `+1666${__VU}${__ITER}`,
    password: 'HackerPass1',
    full_name: 'Evil Admin',
    role: 'ADMIN',
  });
  check(admin, { 'admin self-register blocked': (r) => r.status === 403 || r.status === 400 || r.status >= 500 });

  const sqli = post('/auth/login', {
    email: `' OR 1=1 --`,
    password: `' OR '1'='1`,
  });
  check(sqli, { 'sql injection does not authenticate': (r) => r.status !== 200 });

  const weak = post('/auth/register', {
    email: `weak.${__VU}@example.com`,
    phone: `+1777${__VU}${__ITER}`,
    password: '123',
    full_name: 'Weak',
    role: 'RIDER',
  });
  check(weak, { 'weak password rejected': (r) => r.status === 400 || r.status >= 500 });

  const noneAlg = http.post(
    `${AUTH_URL}/auth/refresh`,
    JSON.stringify({
      refresh_token:
        'eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VyX2lkIjoiYWRtaW4iLCJyb2xlIjoiQURNSU4iLCJ0b2tlbl90eXBlIjoicmVmcmVzaCJ9.',
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  check(noneAlg, { 'alg none jwt rejected': (r) => r.status !== 200 });

  sleep(0.3);
}
