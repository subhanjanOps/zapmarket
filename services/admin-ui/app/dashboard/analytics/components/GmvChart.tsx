'use client';

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

interface DailyGmv { date: string; amount_paise: number }

export function GmvChart({ data }: { data: DailyGmv[] }) {
    const formatted = data.map(d => ({
        date: d.date,
        gmv:  Math.round(d.amount_paise / 100),
    }));

    return (
        <div className="rounded-lg border p-4">
            <h3 className="mb-4 text-sm font-semibold text-gray-600">GMV (₹)</h3>
            <ResponsiveContainer width="100%" height={240}>
                <LineChart data={formatted}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" tick={{ fontSize: 11 }} />
                    <YAxis tick={{ fontSize: 11 }} tickFormatter={(v: number) => `₹${v.toLocaleString('en-IN')}`} />
                    <Tooltip formatter={(v) => [`₹${Number(v).toLocaleString('en-IN')}`, 'GMV']} />
                    <Line type="monotone" dataKey="gmv" stroke="#6366f1" strokeWidth={2} dot={false} />
                </LineChart>
            </ResponsiveContainer>
        </div>
    );
}
