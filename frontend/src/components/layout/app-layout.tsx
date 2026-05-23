"use client";
import { usePathname } from "next/navigation";
import { useState, useEffect } from "react";
import { LayoutDashboard, AlertTriangle, Ambulance, Hospital, GitMerge, Bell, Settings, Moon, Sun, WifiOff, Activity, Car, Menu, X } from "lucide-react";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { useTheme } from "@/components/theme-provider";
import { useIntelligenceStore } from "@/hooks/useIntelligenceStore";
import { useNetworkStore } from "@/hooks/useNetworkStore";

const NAV_ITEMS = [
  { icon: LayoutDashboard, label: "Dashboard", href: "/" },
  { icon: AlertTriangle, label: "Incidents", href: "/incidents" },
  { icon: Ambulance, label: "Ambulances", href: "/ambulances" },
  { icon: Hospital, label: "Hospitals", href: "/hospitals" },
  { icon: GitMerge, label: "Allocations", href: "/allocation" },
  { icon: Bell, label: "Alerts", href: "/alerts" },
  { icon: Settings, label: "Settings", href: "/settings" },
];

export function AppLayout({ children, title, subtitle, rightContent }: any) {
  const [mounted, setMounted] = useState(false);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  useEffect(() => setMounted(true), []);

  const pathname = usePathname();
  const { theme, toggleTheme } = useTheme();
  const { conflictActive, reasoningLogs } = useIntelligenceStore();
  const { isDegraded, failedServices, forceSimulateFailure, toggleSimulateFailure, isTrafficCongested, toggleTrafficCongestion, isPaused, togglePause, resetSimulations } = useNetworkStore();
  
  return (
    <div className="flex h-screen w-full bg-background text-foreground font-sans overflow-hidden transition-colors duration-300">
      {/* ─── LEFT SIDEBAR (DESKTOP) ─── */}
      <aside className="w-[72px] bg-card border-r border-card-border hidden lg:flex flex-col items-center py-6 gap-1 shadow-sm shrink-0 relative z-20 transition-colors duration-300">
        <div className="w-9 h-9 rounded-xl bg-primary flex items-center justify-center mb-6 shadow-md shadow-primary/20">
          <span className="text-primary-foreground font-black text-xs tracking-tight">M</span>
        </div>
        {NAV_ITEMS.map((item) => {
          const active = pathname === item.href;
          return (
            <Link
              key={item.label}
              href={item.href}
              className={cn(
                "w-12 h-12 rounded-xl flex flex-col items-center justify-center gap-0.5 transition-all group",
                active ? "bg-red-50 dark:bg-red-500/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"
              )}
              title={item.label}
            >
              <item.icon className="w-5 h-5 shrink-0" strokeWidth={active ? 2.2 : 1.8} />
              <span className="text-[8px] font-semibold leading-none tracking-wide">{item.label}</span>
            </Link>
          )
        })}
        
        <div className="mt-auto pt-4 flex flex-col gap-4 items-center border-t border-card-border w-full">
          <button 
            onClick={() => {
              resetSimulations();
              useIntelligenceStore.getState().clearLogs();
              useIntelligenceStore.getState().setConflictActive(false);
            }}
            className="w-10 h-10 rounded-full flex items-center justify-center text-muted-foreground hover:bg-red-500/10 hover:text-red-500 transition-all mt-4"
            title="Reset All Simulations"
          >
            <span className="font-bold text-xs">RST</span>
          </button>
          
          <button 
            onClick={togglePause}
            className={cn(
              "w-10 h-10 rounded-full flex items-center justify-center transition-all",
              isPaused ? "bg-amber-500/20 text-amber-500 border border-amber-500/50 animate-pulse" : "text-muted-foreground hover:bg-muted hover:text-foreground"
            )}
            title={isPaused ? "Resume Live Updates" : "Pause Live Updates"}
          >
            <span className="font-bold text-xs">{isPaused ? "▶" : "⏸"}</span>
          </button>

          <button 
            onClick={() => {
              toggleTrafficCongestion();
              if (!isTrafficCongested) {
                useIntelligenceStore.getState().addReasoningLog({
                  timestamp: new Date().toISOString(),
                  action: "Staggering public evacuation alerts to prevent Metro General overload.",
                  trigger: "City-wide Traffic Congestion"
                });
              }
            }}
            className={cn(
              "w-10 h-10 rounded-full flex items-center justify-center transition-all",
              isTrafficCongested ? "bg-orange-500/20 text-orange-500 border border-orange-500/50" : "text-muted-foreground hover:bg-muted hover:text-foreground"
            )}
            title="Simulate Evacuation Congestion"
          >
            <Car className="w-4 h-4" />
          </button>

          <button 
            onClick={toggleSimulateFailure}
            className={cn(
              "w-10 h-10 rounded-full flex items-center justify-center transition-all",
              forceSimulateFailure ? "bg-red-500/20 text-red-500 border border-red-500/50 animate-pulse" : "text-muted-foreground hover:bg-muted hover:text-foreground"
            )}
            title="Simulate API Outage"
          >
            {forceSimulateFailure ? <WifiOff className="w-4 h-4" /> : <Activity className="w-4 h-4" />}
          </button>
          
          <button 
            onClick={toggleTheme}
            className="w-12 h-12 rounded-xl flex items-center justify-center text-muted-foreground hover:bg-muted hover:text-foreground transition-all"
            title="Toggle Theme"
          >
            {mounted ? (theme === 'dark' ? <Sun className="w-5 h-5" strokeWidth={1.8} /> : <Moon className="w-5 h-5" strokeWidth={1.8} />) : <div className="w-5 h-5" />}
          </button>
        </div>
      </aside>

      {/* ─── MOBILE DRAWER OVERLAY ─── */}
      {isSidebarOpen && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 lg:hidden" onClick={() => setIsSidebarOpen(false)}>
          <aside 
            className="w-64 h-full bg-card border-r border-card-border flex flex-col p-5 gap-4 shadow-xl animate-in slide-in-from-left duration-200" 
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <div className="w-9 h-9 rounded-xl bg-primary flex items-center justify-center shadow-md shadow-primary/20">
                  <span className="text-primary-foreground font-black text-xs tracking-tight">M</span>
                </div>
                <span className="font-bold text-sm tracking-wide text-foreground">MEDUSA Ops</span>
              </div>
              <button onClick={() => setIsSidebarOpen(false)} className="p-2 rounded-lg hover:bg-muted text-muted-foreground transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>

            <nav className="flex flex-col gap-1 overflow-y-auto pr-1 flex-1">
              {NAV_ITEMS.map((item) => {
                const active = pathname === item.href;
                return (
                  <Link
                    key={item.label}
                    href={item.href}
                    onClick={() => setIsSidebarOpen(false)}
                    className={cn(
                      "w-full flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all",
                      active ? "bg-red-50 dark:bg-red-500/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"
                    )}
                  >
                    <item.icon className="w-5 h-5 shrink-0" strokeWidth={active ? 2.0 : 1.8} />
                    <span className="text-xs font-bold">{item.label}</span>
                  </Link>
                )
              })}
            </nav>

            {/* Mobile Simulation Controls at bottom of drawer */}
            <div className="pt-4 border-t border-card-border flex flex-col gap-2 mt-auto shrink-0">
              <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider px-1">Simulation Control</p>
              
              <button 
                onClick={() => {
                  resetSimulations();
                  useIntelligenceStore.getState().clearLogs();
                  useIntelligenceStore.getState().setConflictActive(false);
                  setIsSidebarOpen(false);
                }}
                className="flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-bold text-red-500 hover:bg-red-500/10 transition-all text-left w-full"
              >
                <span className="font-bold text-xs bg-red-500/10 px-1.5 py-0.5 rounded border border-red-500/20">RST</span>
                <span>Reset Simulations</span>
              </button>
              
              <button 
                onClick={() => {
                  togglePause();
                  setIsSidebarOpen(false);
                }}
                className={cn(
                  "flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-bold transition-all text-left w-full",
                  isPaused ? "bg-amber-500/20 text-amber-500 border border-amber-500/50" : "text-muted-foreground hover:bg-muted"
                )}
              >
                <span className="font-bold text-xs bg-amber-500/10 px-1.5 py-0.5 rounded border border-amber-500/20">{isPaused ? "▶" : "⏸"}</span>
                <span>{isPaused ? "Resume Updates" : "Pause Updates"}</span>
              </button>

              <button 
                onClick={() => {
                  toggleTrafficCongestion();
                  setIsSidebarOpen(false);
                }}
                className={cn(
                  "flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-bold transition-all text-left w-full",
                  isTrafficCongested ? "bg-orange-500/20 text-orange-500 border border-orange-500/50" : "text-muted-foreground hover:bg-muted"
                )}
              >
                <Car className="w-4 h-4 shrink-0" />
                <span>Traffic Congestion</span>
              </button>

              <button 
                onClick={() => {
                  toggleSimulateFailure();
                  setIsSidebarOpen(false);
                }}
                className={cn(
                  "flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-bold transition-all text-left w-full",
                  forceSimulateFailure ? "bg-red-500/20 text-red-500 border border-red-500/50" : "text-muted-foreground hover:bg-muted"
                )}
              >
                {forceSimulateFailure ? <WifiOff className="w-4 h-4 shrink-0" /> : <Activity className="w-4 h-4 shrink-0" />}
                <span>API Outage</span>
              </button>
              
              <button 
                onClick={() => {
                  toggleTheme();
                  setIsSidebarOpen(false);
                }}
                className="flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-bold text-muted-foreground hover:bg-muted transition-all text-left w-full"
              >
                {mounted && (theme === 'dark' ? <Sun className="w-4 h-4 shrink-0" /> : <Moon className="w-4 h-4 shrink-0" />)}
                <span>{theme === 'dark' ? 'Light Mode' : 'Dark Mode'}</span>
              </button>
            </div>
          </aside>
        </div>
      )}

      {/* ─── MAIN AREA ─── */}
      <div className="flex flex-col flex-1 overflow-hidden relative z-10">
        <header className="bg-card border-b border-card-border px-4 lg:px-6 h-14 flex items-center justify-between shrink-0 shadow-sm relative z-20 transition-colors duration-300">
          <div className="flex items-center gap-2">
            <button 
              onClick={() => setIsSidebarOpen(true)} 
              className="p-2 rounded-lg text-muted-foreground hover:bg-muted lg:hidden transition-colors"
              title="Open Navigation"
            >
              <Menu className="w-5 h-5" />
            </button>
            <div>
              <h1 className="text-sm lg:text-base font-semibold text-foreground truncate max-w-[200px] sm:max-w-xs">{title}</h1>
              {subtitle && <p className="text-[10px] lg:text-xs font-medium text-muted-foreground mt-0.5 truncate max-w-[200px] sm:max-w-xs">{subtitle}</p>}
            </div>
          </div>
          <div className="flex items-center gap-2 sm:gap-3">
            {rightContent}
          </div>
        </header>

        {isPaused && (
          <div className="bg-amber-500/10 text-amber-600 dark:text-amber-400 px-4 py-2 flex items-center justify-between shrink-0 shadow-sm border-b border-amber-500/20 animate-in slide-in-from-top-4">
            <div className="flex items-center gap-2 font-bold text-sm">
              <span className="animate-pulse">⏸</span>
              LIVE UPDATES PAUSED
            </div>
            <div className="flex items-center gap-2 text-xs font-semibold">
              <span className="opacity-80">Scenario state is frozen for presentation.</span>
            </div>
          </div>
        )}

        {isDegraded && (
          <div className="bg-red-500/10 text-red-600 dark:text-red-400 px-4 py-2 flex items-center justify-between shrink-0 shadow-sm border-b border-red-500/20 animate-in slide-in-from-top-4">
            <div className="flex items-center gap-2 font-bold text-sm">
              <WifiOff className="w-5 h-5" />
              DEGRADED MODE ACTIVE
            </div>
            <div className="flex items-center gap-2 text-xs font-semibold">
              <span className="opacity-80">Failed Services:</span>
              <span className="bg-red-500 text-white px-2 py-0.5 rounded uppercase tracking-wider">
                {failedServices.join(', ')}
              </span>
              <span className="text-red-500/70 ml-2">Using local cached estimators.</span>
            </div>
          </div>
        )}
        
        {isTrafficCongested && (
          <div className="bg-orange-500/10 text-orange-600 dark:text-orange-400 px-4 py-2 flex items-center justify-between shrink-0 shadow-sm border-b border-orange-500/20 animate-in slide-in-from-top-4">
            <div className="flex items-center gap-2 font-bold text-sm">
              <Car className="w-5 h-5 animate-bounce" />
              EVACUATION CONGESTION DETECTED
            </div>
            <div className="flex items-center gap-2 text-xs font-semibold">
              <span className="opacity-80">Impact:</span>
              <span className="bg-orange-500 text-white px-2 py-0.5 rounded uppercase tracking-wider">
                Heavy Route Delays
              </span>
              <span className="text-orange-500/70 ml-2">Re-evaluating ETAs and dynamic rerouting active.</span>
            </div>
          </div>
        )}

        {conflictActive && (
          <div className="bg-amber-500 text-amber-950 px-4 py-2 flex items-center justify-between shrink-0 shadow-md border-b border-amber-600 animate-in slide-in-from-top-4">
            <div className="flex items-center gap-2 font-bold text-sm">
              <AlertTriangle className="w-5 h-5 animate-pulse" />
              RESOURCE CONFLICT ACTIVE
            </div>
            {reasoningLogs.length > 0 && (
              <div className="flex items-center gap-2 text-xs font-semibold">
                <span className="opacity-80">Latest Intelligence Action:</span>
                <span className="bg-white/20 px-2 py-0.5 rounded text-amber-900 border border-amber-900/10">
                  {reasoningLogs[0].action}
                </span>
              </div>
            )}
          </div>
        )}
        <main className="flex-1 overflow-hidden relative flex flex-col bg-background/50">
          {children}
        </main>
      </div>
    </div>
  );
}
