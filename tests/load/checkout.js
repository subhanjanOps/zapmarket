import http from 'k6/http';
import { check, sleep } from 'k6';
import { thresholds } from './k6.config.js';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Lower VUs for checkout — it hits payment gateway
export const options = {
    thresholds,
    stages: [
        { duration: '30s', target: 5  },
        { duration: '60s', target: 10 },
        { duration: '30s', target: 0  },
    ],
};

const BASE_URL     = __ENV.BASE_URL          || 'http://localhost:8000';
const TOKEN        = __ENV.LOAD_TEST_TOKEN   || '';
const SKU_ID       = __ENV.LOAD_TEST_SKU_ID  || '';

export default function () {
    const headers = {
        'Content-Type':    'application/json',
        'Authorization':   `Bearer ${TOKEN}`,
        'Idempotency-Key': uuidv4(),
    };

    const orderRes = http.post(`${BASE_URL}/v1/orders`, JSON.stringify({
        items:    [{ sku_id: SKU_ID, quantity: 1 }],
        currency: 'INR',
    }), { headers });

    check(orderRes, { 'order created 201 or 202': (r) => r.status === 201 || r.status === 202 });
    sleep(2);
}
