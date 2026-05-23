"use client";

import { useState } from "react";
import { AppLayout } from "@/components/layout/app-layout";
import { AlertTriangle, Hospital, CheckCircle2, Circle, Clock, User, Navigation, Activity } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTheme } from "@/components/theme-provider";
import { LiveMap } from "@/components/ui/live-map";

const TIMELINE_STEPS = [
  { label: "Dispatched", time: "10:14 AM", done: true },
  { label: "En Route", time: "10:16 AM", active: true, done: false },
  { label: "Arrived", time: "—", done: false },
  { label: "Transfer Complete", time: "—", done: false },
];

export default function AmbulancesPage() {
  const [progress] = useState(65);

  return (
    <AppLayout 
      title="Ambulance Tracking" 
      subtitle="Live route monitoring — Unit AMB-047"
      rightContent={
        <>
          <span className="flex items-center gap-1.5 text-xs font-bold text-emerald-700 bg-emerald-50 border border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20 px-4 py-2 rounded-full shadow-sm transition-colors duration-300">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
            Live Tracking Active
          </span>
          <span className="text-xs font-semibold text-muted-foreground transition-colors duration-300">Updated just now</span>
        </>
      }
    >
      <div className="flex flex-col lg:flex-row flex-1 overflow-y-auto lg:overflow-hidden p-4 lg:p-5 gap-4 lg:gap-5">
        {/* ─── MAP AREA ─── */}
        <div className="flex-1 flex flex-col gap-4 lg:gap-5 overflow-hidden min-w-0 min-h-0">
          <div className="flex-1 min-h-[300px] lg:min-h-0 rounded-2xl overflow-hidden relative shadow-md border border-card-border transition-colors duration-300">
            <LiveMap center={[40.7306, -73.9852]} zoom={13}>
              {/* Map overlay badges */}
              <div className="absolute top-5 left-5 flex gap-3 z-[400] pointer-events-none">
                <div className="bg-black/60 backdrop-blur-md border border-white/10 text-white text-xs px-4 py-2 rounded-full font-bold flex items-center gap-2 shadow-lg">
                  <Navigation className="w-3.5 h-3.5" /> Downtown Metro → Metro General
                </div>
                <div className="bg-red-500/90 backdrop-blur-md text-white text-xs px-4 py-2 rounded-full font-bold shadow-lg shadow-red-500/20">
                  P1 Critical
                </div>
              </div>
              <div className="absolute top-5 right-5 bg-black/60 backdrop-blur-md border border-white/10 text-white text-xs px-4 py-2 rounded-full font-bold flex items-center gap-2 shadow-lg z-[400] pointer-events-none">
                <Clock className="w-3.5 h-3.5 text-amber-400" />
                <span className="text-amber-400">ETA 8 min</span>
              </div>
            </LiveMap>
          </div>

          {/* ─── TIMELINE ─── */}
          <div className="bg-card rounded-2xl px-4 lg:px-8 py-4 lg:py-6 shadow-sm border border-card-border shrink-0 transition-colors duration-300">
            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest mb-4 lg:mb-5 transition-colors duration-300">Dispatch Timeline</p>
            <div className="overflow-x-auto w-full">
            <div className="flex items-start gap-0 min-w-[360px]">
              {TIMELINE_STEPS.map((step, i) => (
                <div key={step.label} className="flex-1 flex flex-col items-center relative">
                  {i < TIMELINE_STEPS.length - 1 && (
                    <div className={cn(
                      "absolute left-1/2 top-4 h-0.5 w-full -translate-y-1/2 transition-colors duration-300",
                      step.done ? "bg-emerald-400" : "bg-muted"
                    )} />
                  )}
                  <div className={cn(
                    "relative z-10 w-8 h-8 rounded-full border-2 flex items-center justify-center mb-3 transition-all duration-300",
                    step.done
                      ? "bg-emerald-500 border-emerald-500 shadow-md shadow-emerald-500/20"
                      : step.active
                        ? "bg-card border-blue-500 shadow-lg shadow-blue-500/30"
                        : "bg-card border-card-border"
                  )}>
                    {step.done
                      ? <CheckCircle2 className="w-4.5 h-4.5 text-white" strokeWidth={3} />
                      : step.active
                        ? <span className="w-3 h-3 rounded-full bg-blue-500 animate-pulse" />
                        : <Circle className="w-4 h-4 text-muted" />
                    }
                  </div>
                  <p className={cn(
                    "text-xs font-bold text-center transition-colors duration-300",
                    step.done ? "text-emerald-600 dark:text-emerald-500" : step.active ? "text-blue-600 dark:text-blue-400" : "text-muted-foreground"
                  )}>{step.label}</p>
                  <p className="text-[10px] font-semibold text-muted-foreground mt-1 transition-colors duration-300">{step.time}</p>
                </div>
              ))}
            </div>
            </div>
          </div>
        </div>

        {/* ─── RIGHT INFO PANEL ─── */}
        <div className="w-full lg:w-[320px] flex flex-col gap-4 lg:gap-5 shrink-0">
          <div className="bg-card rounded-2xl p-5 lg:p-7 shadow-sm border border-card-border flex flex-col gap-6 lg:gap-8 transition-colors duration-300">
            <div>
              <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest mb-1.5 transition-colors duration-300">Active Unit</p>
              <h2 className="text-2xl lg:text-3xl font-black text-foreground tracking-tight transition-colors duration-300">AMB-047</h2>
              <p className="text-sm font-medium text-muted-foreground mt-1 transition-colors duration-300">Ford Transit Type B</p>
            </div>

            <div className="space-y-5">
              <InfoRow icon={User} label="Driver" value="James Carter" />
              <InfoRow
                icon={Activity}
                label="Status"
                value={
                  <span className="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400 font-bold text-sm transition-colors duration-300">
                    <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                    En Route
                  </span>
                }
              />
              <InfoRow icon={Clock} label="ETA" value={<span className="font-bold text-amber-500 dark:text-amber-400 transition-colors duration-300">8 min</span>} />
              <InfoRow icon={Hospital} label="Destination" value="Metro General" className="text-blue-600 dark:text-blue-400 font-bold transition-colors duration-300" />
              <InfoRow
                icon={AlertTriangle}
                label="Priority"
                value={<span className="bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-400 text-[10px] font-bold px-3 py-1.5 rounded-full border border-red-200 dark:border-red-500/20 uppercase tracking-wider transition-colors duration-300">P1 Critical</span>}
              />
            </div>

            <div className="mt-auto">
              <div className="flex justify-between items-center mb-2.5">
                <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest transition-colors duration-300">Route Progress</p>
                <p className="text-sm font-black text-foreground transition-colors duration-300">{progress}%</p>
              </div>
              <div className="h-3 bg-muted rounded-full overflow-hidden transition-colors duration-300">
                <div
                  className="h-full bg-gradient-to-r from-blue-500 to-blue-400 rounded-full transition-all duration-700 ease-in-out shadow-sm"
                  style={{ width: `${progress}%` }}
                />
              </div>
              <div className="flex justify-between mt-2">
                <span className="text-[10px] font-bold text-muted-foreground transition-colors duration-300">Origin</span>
                <span className="text-[10px] font-bold text-blue-500 dark:text-blue-400 transition-colors duration-300">Destination</span>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            {[
              { label: "Speed", value: "72 km/h" },
              { label: "Dist Left", value: "4.2 km" },
              { label: "Dispatch #", value: "D-1029" },
              { label: "Crew", value: "2 On Board" },
            ].map((s) => (
              <div key={s.label} className="bg-card rounded-xl p-4 shadow-sm border border-card-border text-center transition-colors duration-300">
                <p className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold mb-1.5 transition-colors duration-300">{s.label}</p>
                <p className="text-sm font-black text-foreground transition-colors duration-300">{s.value}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </AppLayout>
  );
}

function InfoRow({ icon: Icon, label, value, className }: any) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="flex items-center gap-2.5 min-w-0">
        <Icon className="w-[18px] h-[18px] text-muted-foreground shrink-0 transition-colors duration-300" />
        <span className="text-sm font-bold text-muted-foreground uppercase tracking-wide truncate transition-colors duration-300">{label}</span>
      </div>
      <div className={cn("text-sm font-bold text-foreground text-right transition-colors duration-300", className)}>
        {value}
      </div>
    </div>
  );
}
