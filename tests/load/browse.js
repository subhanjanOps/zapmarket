import http from 'k6/http';
import { check, sleep } from 'k6';
import { thresholds, stages } from './k6.config.js';

export const options = { thresholds, stages };

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

export default function () {
    const listRes = http.get(`${BASE_URL}/v1/products?limit=20&offset=0`);
    check(listRes, { 'product list 200': (r) => r.status === 200 });

    const searchRes = http.get(`${BASE_URL}/v1/products/search?q=phone&per_page=20`);
    check(searchRes, { 'search 200': (r) => r.status === 200 });

    sleep(0.5);
}
