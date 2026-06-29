import { GmvChart } from './components/GmvChart';
import { OrderFunnelChart } from './components/OrderFunnelChart';

async function getAnalytics(range: string) {
    const res = await fetch(`${process.env.NEXT_PUBLIC_APP_URL}/api/analytics?range=${range}`, {
        next: { revalidate: 300 },
    });
    return res.ok ? res.json() : { gmv: { daily: [] }, funnel: { stages: [] } };
}

export default async function AnalyticsPage({ searchParams }: { searchParams: Promise<{ range?: string }> }) {
    const params = await searchParams;
    const range = params.range || '7d';
    const data = await getAnalytics(range);

    return (
        <div className="space-y-6 p-6">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-bold">Analytics</h1>
                <div className="flex gap-2">
                    {['7d', '30d', '90d'].map(r => (
                        <a key={r} href={`?range=${r}`}
                           className={`rounded px-3 py-1 text-sm ${range === r ? 'bg-indigo-600 text-white' : 'bg-gray-100 text-gray-700'}`}>
                            {r}
                        </a>
                    ))}
                </div>
            </div>
            <GmvChart data={data.gmv.daily ?? []} />
            <OrderFunnelChart data={data.funnel.stages ?? []} />
        </div>
    );
}
