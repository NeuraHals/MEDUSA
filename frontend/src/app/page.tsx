"use client";
import { AppLayout } from "@/components/layout/app-layout";
import { StatCard, SectionCard } from "@/components/ui/shared";
import { Activity, AlertTriangle, Ambulance, Hospital, Zap, Brain, Car, WifiOff, Shuffle, AlertTriangle as AlertTriangleIcon } from "lucide-react";
import { LiveMap } from "@/components/ui/live-map";
import { useIncidents } from "@/hooks/useIncidents";
import { useAmbulances } from "@/hooks/useAmbulances";
import { useHospitals } from "@/hooks/useHospitals";
import { useAlerts } from "@/hooks/useAlerts";
import { useIntelligenceStore } from "@/hooks/useIntelligenceStore";
import { useNetworkStore } from "@/hooks/useNetworkStore";
import { ScenarioEngine, ScenarioType } from "@/services/scenario-engine";
import { useQueryClient } from "@tanstack/react-query";


export default function DashboardPage() {
  const { data: incidents, isLoading: incLoading } = useIncidents();
  const { data: ambulances, isLoading: ambLoading } = useAmbulances();
  const { data: hospitals, isLoading: hospLoading } = useHospitals();
  const { data: alerts, isLoading: alertsLoading } = useAlerts();
  const queryClient = useQueryClient();

  const { reasoningLogs, addReasoningLog, clearLogs, setConflictActive } = useIntelligenceStore();
  const { toggleSimulateFailure, isDegraded, toggleTrafficCongestion, isTrafficCongested } = useNetworkStore();

  const activeIncidents = incidents?.filter(i => i.status === 'active').length || 0;
  const criticalIncidents = incidents?.filter(i => i.severity === 'critical' && i.status === 'active').length || 0;
  
  const totalAmbulances = ambulances?.length || 0;
  const availableAmbulances = ambulances?.filter(a => a.status === 'available').length || 0;
  
  const totalHospitalLoad = hospitals && hospitals.length > 0
    ? Math.round(hospitals.reduce((acc, h) => acc + h.current_load_pct, 0) / hospitals.length)
    : 0;

  // --- SCENARIO TRIGGERS ---
  const triggerScenario = async (type: ScenarioType) => {
    clearLogs();
    await ScenarioEngine.execute(type, queryClient);
  };

  const cycleRandomDisaster = async () => {
    const scenarios: ScenarioType[] = ['flood', 'multi-vehicle', 'mci', 'hospital-overload', 'wildfire'];
    const random = scenarios[Math.floor(Math.random() * scenarios.length)];
    await triggerScenario(random);
  };

  return (
    <AppLayout title="Operations Overview" subtitle="System-wide metrics and status">
      <div className="p-4 lg:p-5 flex-1 overflow-auto flex flex-col gap-4 lg:gap-5">
        {/* KPI Row */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 lg:gap-5 shrink-0">
          <StatCard title="Active Incidents" value={incLoading ? "-" : activeIncidents.toString()} subtitle={`${criticalIncidents} Critical, ${activeIncidents - criticalIncidents} High`} icon={AlertTriangle} color="red" />
          <StatCard title="Hospital Load" value={hospLoading ? "-" : `${totalHospitalLoad}%`} subtitle="System-wide capacity" icon={Hospital} color="amber" />
          <StatCard title="Available Units" value={ambLoading ? "-" : availableAmbulances.toString()} subtitle={`Out of ${totalAmbulances} total fleet`} icon={Ambulance} color="emerald" />
          <StatCard title="Pending Transfers" value={incLoading ? "-" : "8"} subtitle="Awaiting routing" icon={Activity} color="blue" />
        </div>

        {/* Phase 7: Demo Scenario Engine */}
        <SectionCard title="MEDUSA Intelligence Demonstrations" className="shrink-0 border-indigo-500/30 shadow-lg shadow-indigo-500/10">
          <div className="flex gap-3 mt-2 overflow-x-auto pb-2 custom-scrollbar">
            <button onClick={cycleRandomDisaster} className="bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/50 px-5 py-2.5 rounded-lg text-xs font-black hover:bg-indigo-500/20 transition-all flex items-center gap-2 whitespace-nowrap shadow-sm">
              <Shuffle className="w-4 h-4" /> Cycle Random Disaster
            </button>
            <div className="w-px bg-card-border mx-1" />
            <button onClick={() => triggerScenario('multi-vehicle')} className="bg-amber-500/10 text-amber-600 border border-amber-500/50 px-4 py-2.5 rounded-lg text-xs font-bold hover:bg-amber-500/20 transition-colors flex items-center gap-2 whitespace-nowrap">
              <Zap className="w-4 h-4" /> Multi-Crisis Conflict
            </button>
            <button onClick={() => triggerScenario('mci')} className="bg-red-500/10 text-red-600 border border-red-500/50 px-4 py-2.5 rounded-lg text-xs font-bold hover:bg-red-500/20 transition-colors flex items-center gap-2 whitespace-nowrap">
              <AlertTriangleIcon className="w-4 h-4" /> Mass Casualty Incident
            </button>
            <button onClick={() => triggerScenario('contradiction')} className="bg-blue-500/10 text-blue-600 border border-blue-500/50 px-4 py-2.5 rounded-lg text-xs font-bold hover:bg-blue-500/20 transition-colors flex items-center gap-2 whitespace-nowrap">
              <Brain className="w-4 h-4" /> Intel Contradiction
            </button>
            <button onClick={() => triggerScenario('api-outage')} className="bg-slate-500/10 text-slate-500 dark:text-slate-400 border border-slate-500/50 px-4 py-2.5 rounded-lg text-xs font-bold hover:bg-slate-500/20 transition-colors flex items-center gap-2 whitespace-nowrap">
              <WifiOff className="w-4 h-4" /> Simulate API Outage
            </button>
            <button onClick={() => triggerScenario('evacuation')} className="bg-orange-500/10 text-orange-600 border border-orange-500/50 px-4 py-2.5 rounded-lg text-xs font-bold hover:bg-orange-500/20 transition-colors flex items-center gap-2 whitespace-nowrap">
              <Car className="w-4 h-4" /> Evac Congestion
            </button>
            <button onClick={() => triggerScenario('false-alarm')} className="bg-emerald-500/10 text-emerald-600 border border-emerald-500/50 px-4 py-2.5 rounded-lg text-xs font-bold hover:bg-emerald-500/20 transition-colors flex items-center gap-2 whitespace-nowrap">
              <AlertTriangleIcon className="w-4 h-4" /> False Alarm Retract
            </button>
          </div>
        </SectionCard>

        {/* Map + Phase 6 AI Reasoning Feed */}
        <div className="flex flex-col lg:flex-row gap-4 lg:gap-5 flex-1 min-h-0 lg:min-h-0">
          <SectionCard title="Live Operational Map" className="flex-1 min-h-[350px] sm:min-h-[450px] lg:min-h-0 overflow-hidden" noPadding>
            <div className="flex-1 min-h-0 relative rounded-b-2xl overflow-hidden h-full">
              <LiveMap center={[40.7306, -73.9852]} zoom={12} />
            </div>
          </SectionCard>

          <div className="w-full lg:w-[420px] flex flex-col gap-4 lg:gap-5 shrink-0 min-h-[300px] lg:min-h-0">
            <SectionCard title="AI Reasoning Feed" className="flex-1 min-h-0 overflow-auto border-emerald-500/30" noPadding>
              <div className="px-4 lg:px-6 pb-4 lg:pb-6 space-y-3 lg:space-y-4 pt-4 lg:pt-6">
                {reasoningLogs?.length === 0 && (
                  <p className="text-center text-sm text-muted-foreground font-bold mt-4">Awaiting intelligence events...</p>
                )}
                {reasoningLogs?.map((log, i) => (
                  <div key={i} className="flex items-start gap-3 bg-muted/30 p-3 rounded-lg border border-card-border/60 shadow-sm transition-all animate-in slide-in-from-right-4">
                    <Brain className="w-5 h-5 text-emerald-500 shrink-0 mt-0.5" />
                    <div className="flex-1">
                      <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider mb-1">Trigger: {log.trigger}</p>
                      <p className="text-xs sm:text-sm font-semibold text-foreground/90">{log.action}</p>
                      <p className="text-[10px] text-muted-foreground mt-2 text-right">{new Date(log.timestamp).toLocaleTimeString()}</p>
                    </div>
                  </div>
                ))}
              </div>
            </SectionCard>
          </div>
        </div>
      </div>
    </AppLayout>
  );
}
