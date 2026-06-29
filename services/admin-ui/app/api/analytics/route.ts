import { NextRequest, NextResponse } from 'next/server';

const ORDER_SERVICE_URL = process.env.ORDER_SERVICE_URL || 'http://order-management-service:8084';

export async function GET(req: NextRequest) {
    const range = req.nextUrl.searchParams.get('range') || '7d';
    const days = range === '90d' ? 90 : range === '30d' ? 30 : 7;

    const [gmvRes, funnelRes] = await Promise.all([
        fetch(`${ORDER_SERVICE_URL}/v1/admin/analytics/gmv?days=${days}`),
        fetch(`${ORDER_SERVICE_URL}/v1/admin/analytics/funnel?days=${days}`),
    ]);

    const gmv    = gmvRes.ok    ? await gmvRes.json()    : { daily: [] };
    const funnel = funnelRes.ok ? await funnelRes.json() : { stages: [] };

    return NextResponse.json({ gmv, funnel });
}
