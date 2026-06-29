export const thresholds = {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed:   ['rate<0.01'],
};

export const stages = [
    { duration: '30s', target: 10  },
    { duration: '60s', target: 50  },
    { duration: '30s', target: 100 },
    { duration: '30s', target: 0   },
];
