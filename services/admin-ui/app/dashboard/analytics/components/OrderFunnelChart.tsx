'use client';

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

interface FunnelStage { stage: string; count: number }

export function OrderFunnelChart({ data }: { data: FunnelStage[] }) {
    return (
        <div className="rounded-lg border p-4">
            <h3 className="mb-4 text-sm font-semibold text-gray-600">Order Funnel</h3>
            <ResponsiveContainer width="100%" height={240}>
                <BarChart data={data}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="stage" tick={{ fontSize: 11 }} />
                    <YAxis tick={{ fontSize: 11 }} />
                    <Tooltip />
                    <Bar dataKey="count" fill="#6366f1" radius={[4, 4, 0, 0]} />
                </BarChart>
            </ResponsiveContainer>
        </div>
    );
}
