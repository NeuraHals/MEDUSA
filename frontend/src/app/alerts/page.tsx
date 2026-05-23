"use client";
import { useState } from "react";
import { AppLayout } from "@/components/layout/app-layout";
import { SectionCard } from "@/components/ui/shared";
import { AlertTriangle, Clock, ShieldAlert, Zap } from "lucide-react";
import { ContradictionPanel } from "@/components/ui/contradiction-panel";
import { ContradictionEngine } from "@/services/contradiction-engine";
import { useIntelligenceStore } from "@/hooks/useIntelligenceStore";
import { useAlerts } from "@/hooks/useAlerts";

export default function AlertsPage() {
  const [filter, setFilter] = useState<'all' | 'critical' | 'system'>('all');
  const { data: alerts } = useAlerts();
  const { setContradictions, contradictions } = useIntelligenceStore();

  const handleRunContradictionScan = async () => {
    const found = await ContradictionEngine.runScenarios();
    setContradictions(found);
  };

  return (
    <AppLayout 
      title="System Alerts" 
      subtitle="Chronological event feed and intelligence verification"
      rightContent={
        <button 
          onClick={handleRunContradictionScan} 
          className="bg-red-600 text-white px-5 py-2 rounded-lg text-xs font-bold hover:bg-red-700 transition-colors shadow-sm flex items-center gap-2"
        >
          <ShieldAlert className="w-4 h-4" /> Run Contradiction Scan
        </button>
      }
    >
      <div className="p-4 lg:p-5 flex flex-col lg:flex-row gap-4 lg:gap-5 h-full overflow-y-auto lg:overflow-hidden">

        {/* ── Left: Live Alert Feed ─────────────────────── */}
        <SectionCard className="flex-1 overflow-visible lg:overflow-auto bg-card" noPadding>
          <div className="p-4 lg:p-6 border-b border-card-border flex items-center gap-2 bg-card sticky top-0 z-10 transition-colors overflow-x-auto whitespace-nowrap">
            <button onClick={() => setFilter('all')} className={`px-4 py-1.5 rounded-full text-xs font-bold shadow-sm transition-colors ${filter === 'all' ? 'bg-foreground text-background' : 'bg-card text-muted-foreground border border-card-border hover:bg-muted'}`}>All Events</button>
            <button onClick={() => setFilter('critical')} className={`px-4 py-1.5 rounded-full text-xs font-bold shadow-sm transition-colors ${filter === 'critical' ? 'bg-red-500 text-white border-transparent' : 'bg-card text-muted-foreground border border-card-border hover:bg-muted'}`}>Critical</button>
            <button onClick={() => setFilter('system')} className={`px-4 py-1.5 rounded-full text-xs font-bold shadow-sm transition-colors ${filter === 'system' ? 'bg-foreground text-background' : 'bg-card text-muted-foreground border border-card-border hover:bg-muted'}`}>System</button>
          </div>
          <div className="p-6 space-y-4">
            {/* Live alerts from hook */}
            {alerts?.filter(a => {
              if (filter === 'all') return true;
              if (filter === 'critical') return a.severity === 'critical' || a.severity === 'high';
              return a.severity === 'medium' || a.severity === 'low';
            }).map((alert, i) => {
              const isCritical = alert.severity === 'critical';
              const minsAgo = Math.floor((Date.now() - new Date(alert.created_at).getTime()) / 60000);
              return (
                <div key={alert.id} className="flex gap-5 p-5 rounded-2xl border border-card-border bg-card hover:shadow-md transition-shadow group">
                  <div className={`w-12 h-12 rounded-xl flex items-center justify-center shrink-0 border transition-colors ${isCritical ? 'bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-400 border-red-100 dark:border-red-500/20' : 'bg-amber-50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-100 dark:border-amber-500/20'}`}>
                    <AlertTriangle className="w-6 h-6" />
                  </div>
                  <div className="flex-1 pt-1">
                    <div className="flex justify-between items-start">
                      <h4 className="text-base font-bold text-foreground">{alert.title}</h4>
                      <span className="text-xs font-bold text-muted-foreground flex items-center gap-1.5">
                        <Clock className="w-3.5 h-3.5" /> {minsAgo}m ago
                      </span>
                    </div>
                    <p className="text-sm font-medium text-muted-foreground mt-1.5 leading-relaxed">{alert.location}</p>
                  </div>
                </div>
              );
            })}

            {/* Fallback mock entries */}
            {(!alerts || alerts.length === 0) && [1, 2, 3, 4, 5, 6].filter(i => {
              if (filter === 'all') return true;
              if (filter === 'critical') return i === 1;
              return i !== 1;
            }).map(i => (
              <div key={i} className="flex gap-5 p-5 rounded-2xl border border-card-border bg-card hover:shadow-md transition-shadow group">
                <div className={`w-12 h-12 rounded-xl flex items-center justify-center shrink-0 border transition-colors ${i === 1 ? 'bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-400 border-red-100 dark:border-red-500/20' : 'bg-amber-50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-100 dark:border-amber-500/20'}`}>
                  <AlertTriangle className="w-6 h-6" />
                </div>
                <div className="flex-1 pt-1">
                  <div className="flex justify-between items-start">
                    <h4 className="text-base font-bold text-foreground">{i === 1 ? 'Critical Capacity Warning' : 'Network Latency Spike'}</h4>
                    <span className="text-xs font-bold text-muted-foreground flex items-center gap-1.5"><Clock className="w-3.5 h-3.5" /> 10:4{i} AM</span>
                  </div>
                  <p className="text-sm font-medium text-muted-foreground mt-1.5 leading-relaxed">
                    {i === 1 
                      ? "ICU beds at Metro General dropped below 5% availability. Automatic allocation diverts suggested."
                      : "Telemetry cluster experiencing 400ms latency. Signal Agent is buffering data."}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </SectionCard>

        {/* ── Right: Intelligence Contradiction Panel ───── */}
        <div className="w-full lg:w-[420px] shrink-0 flex flex-col gap-4 overflow-visible lg:overflow-auto min-h-[300px] lg:min-h-0">
          {contradictions.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full gap-4 text-center p-8">
              <ShieldAlert className="w-12 h-12 text-muted-foreground/30" />
              <p className="text-sm font-bold text-muted-foreground">No Contradictions Detected</p>
              <p className="text-xs text-muted-foreground/70 leading-relaxed">
                Click <strong>Run Contradiction Scan</strong> to simulate conflicting intelligence scenarios and observe autonomous reasoning.
              </p>
              <button 
                onClick={handleRunContradictionScan} 
                className="mt-2 bg-card border-2 border-card-border text-foreground px-5 py-2.5 rounded-xl text-xs font-bold hover:border-red-400 hover:text-red-500 transition-colors flex items-center gap-2"
              >
                <Zap className="w-4 h-4" /> Trigger Scan Now
              </button>
            </div>
          ) : (
            <ContradictionPanel />
          )}
        </div>

      </div>
    </AppLayout>
  );
}
