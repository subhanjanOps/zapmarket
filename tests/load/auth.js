import http from 'k6/http';
import { check, sleep } from 'k6';
import { thresholds, stages } from './k6.config.js';

export const options = { thresholds, stages };

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

export default function () {
    const loginRes = http.post(`${BASE_URL}/v1/auth/login`, JSON.stringify({
        email:    'loadtest@zapmarket.in',
        password: 'Load1234!',
    }), { headers: { 'Content-Type': 'application/json' } });

    check(loginRes, {
        'login status 200':  (r) => r.status === 200,
        'has access_token':  (r) => JSON.parse(r.body).access_token !== undefined,
    });

    sleep(1);
}
