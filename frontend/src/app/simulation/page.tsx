"use client";
import { AppLayout } from "@/components/layout/app-layout";
import { SectionCard, StatCard } from "@/components/ui/shared";
import { Cpu, TrendingUp } from "lucide-react";
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from "recharts";
import { useTheme } from "@/components/theme-provider";

const mockData = [
  { time: "T+0h", demand: 40 },
  { time: "T+1h", demand: 60 },
  { time: "T+2h", demand: 85 },
  { time: "T+3h", demand: 98 },
  { time: "T+4h", demand: 70 },
];

export default function SimulationPage() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  
  const gridColor = isDark ? '#1f2937' : '#f3f4f6';
  const textColor = isDark ? '#94a3b8' : '#9ca3af';
  const tooltipBg = isDark ? '#111827' : '#ffffff';
  const tooltipBorder = isDark ? '#1f2937' : '#e5e7eb';

  return (
    <AppLayout title="Simulation & Forecasting" subtitle="Predictive demand models">
      <div className="p-4 lg:p-5 flex flex-col gap-4 lg:gap-5 overflow-auto">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 lg:gap-5 shrink-0">
          <StatCard title="Forecasted Overload" value="T+3 hrs" subtitle="Metro General ER" icon={TrendingUp} color="red" />
          <StatCard title="Model Confidence" value="94%" subtitle="10,000 iterations" icon={Cpu} color="emerald" />
        </div>
        <SectionCard title="Demand Projection (Next 4 Hours)" className="shrink-0">
          <div style={{ height: 340 }}>
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={mockData} margin={{ top: 8, right: 16, bottom: 8, left: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={gridColor} vertical={false} />
                <XAxis dataKey="time" stroke={textColor} fontSize={12} tickLine={false} axisLine={false} dy={8} />
                <YAxis stroke={textColor} fontSize={12} tickLine={false} axisLine={false} width={36} tickFormatter={(v) => `${v}%`} />
                <Tooltip
                  contentStyle={{ backgroundColor: tooltipBg, borderRadius: "12px", border: `1px solid ${tooltipBorder}`, fontWeight: "bold", fontSize: 12, color: isDark ? '#f8fafc' : '#0f172a' }}
                  itemStyle={{ color: "#ef4444" }}
                  formatter={(v: number) => [`${v}%`, "Demand"]}
                />
                <Line type="monotone" dataKey="demand" stroke="#ef4444" strokeWidth={3}
                  dot={{ r: 5, fill: "#ef4444", strokeWidth: 2, stroke: isDark ? '#111827' : '#fff' }}
                  activeDot={{ r: 7, stroke: "#ef4444", strokeWidth: 2, fill: isDark ? '#111827' : '#fff' }}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </SectionCard>
      </div>
    </AppLayout>
  );
}
