"use client";
import { useState } from "react";
import { AppLayout } from "@/components/layout/app-layout";
import { SectionCard, StatusBadge } from "@/components/ui/shared";
import { UserPlus, AlertTriangle, Loader2, Zap } from "lucide-react";
import { useIncidents } from "@/hooks/useIncidents";
import { useAmbulances } from "@/hooks/useAmbulances";
import { useHospitals } from "@/hooks/useHospitals";
import { useRealtimeDispatch } from "@/hooks/useRealtimeDispatch";
import { supabase, isPlaceholderSupabase } from "@/lib/supabase";
import { DispatchEngine } from "@/services/dispatch-engine";
import { IntelligenceEngine } from "@/services/intelligence-engine";
import { useIntelligenceStore } from "@/hooks/useIntelligenceStore";
import { useQueryClient } from "@tanstack/react-query";

export default function IncidentsPage() {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const { data: incidents, isLoading, isError } = useIncidents();
  const { data: ambulances } = useAmbulances();
  const { data: hospitals } = useHospitals();
  const { data: dispatches } = useRealtimeDispatch();
  const { setConflictActive, addReasoningLog, clearLogs } = useIntelligenceStore();
  const queryClient = useQueryClient();

  const handleDeclareIncident = async () => {
    if (isPlaceholderSupabase) {
      console.warn('Supabase not connected. Cannot write to mock DB.');
      return;
    }
    
    await supabase.from('incidents').insert([
      {
        external_id: `INC-10${Math.floor(Math.random() * 90) + 10}`,
        severity: 'high',
        description: 'New Emergency Reported',
        latitude: 40.7128 + (Math.random() * 0.05),
        longitude: -74.0060 + (Math.random() * 0.05),
        location_name: 'Unknown Sector',
        status: 'active'
      }
    ]);
  };

  const handleDispatch = async (incident: any) => {
    // Instant UI feedback for demo purposes
    queryClient.setQueryData(['incidents'], (old: any) => {
      if (!old) return [];
      return old.map((i: any) => i.id === incident.id ? { ...i, status: 'resolved' } : i);
    });

    // Auto advance to the next active incident
    const currentList = queryClient.getQueryData<any[]>(['incidents']) || [];
    const nextActive = currentList.find((i: any) => i.id !== incident.id && i.status === 'active');
    if (nextActive) {
      setSelectedId(nextActive.id);
    } else {
      setSelectedId(null);
    }
    
    addReasoningLog({
      timestamp: new Date().toISOString(),
      action: `Unit manually dispatched to ${incident.external_id}.`,
      trigger: "Manual Operator Override"
    });

    if (isPlaceholderSupabase) return;
    if (!ambulances || !hospitals) return;
    
    try {
      await DispatchEngine.autoDispatch(incident, ambulances, hospitals);
    } catch (e) {
      console.error('Dispatch failed', e);
    }
  };

  const handleSimulateCrisis = async () => {
    clearLogs();
    
    // Create a deterministic conflict scenario payload
    const mockIncidents = [
      { id: 'inc-1', external_id: 'INC-2001', severity: 'critical', description: 'Multi-Vehicle Pileup', latitude: 40.7422, longitude: -74.0043, location_name: 'Lincoln Tunnel', status: 'active', created_at: new Date().toISOString() },
      { id: 'inc-2', external_id: 'INC-2002', severity: 'high', description: 'Subway Fire', latitude: 40.7186, longitude: -73.9552, location_name: 'Williamsburg', status: 'active', created_at: new Date().toISOString() }
    ];

    // Only 1 ambulance exists, and it's currently en route to the High severity subway fire
    const mockAmbulances = [
      { id: 'amb-1', unit_number: 'AMB-047', latitude: 40.7306, longitude: -73.9852, status: 'en_route', speed_kmh: 72, destination_id: null, incident_id: 'inc-2' }
    ];

    const mockDispatches = [
      { id: 'disp-1', ambulance_id: 'amb-1', incident_id: 'inc-2', hospital_id: null, status: 'en_route', created_at: new Date().toISOString(), updated_at: new Date().toISOString() }
    ];

    const result = await IntelligenceEngine.evaluateResourceState(
      mockIncidents as any[], 
      mockAmbulances as any[], 
      mockDispatches as any[], 
      []
    );

    setConflictActive(result.conflictActive);
    result.reasoningLogs.reverse().forEach(log => addReasoningLog(log));
  };

  const handleRetract = async (incident: any) => {
    // Instant UI feedback
    queryClient.setQueryData(['incidents'], (old: any) => {
      if (!old) return [];
      return old.map((i: any) => i.id === incident.id ? { ...i, status: 'retracted' } : i);
    });

    // Auto advance to the next active incident
    const currentList = queryClient.getQueryData<any[]>(['incidents']) || [];
    const nextActive = currentList.find((i: any) => i.id !== incident.id && i.status === 'active');
    if (nextActive) {
      setSelectedId(nextActive.id);
    } else {
      setSelectedId(null);
    }

    try {
      await DispatchEngine.retractIncident(incident.id);
      addReasoningLog({
        timestamp: new Date().toISOString(),
        action: `Incident ${incident.external_id} marked as FALSE ALARM. Dispatches cancelled.`,
        trigger: "Manual Retraction Verification"
      });
    } catch (e) {
      console.error('Retract failed', e);
    }
  };

  return (
    <AppLayout 
      title="Incident Management" 
      subtitle="Active emergencies and critical events"
      rightContent={
        <div className="flex gap-3">
          <button onClick={handleSimulateCrisis} className="bg-amber-500 text-amber-950 px-5 py-2 rounded-lg text-xs font-bold hover:bg-amber-600 transition-colors shadow-sm flex items-center gap-2">
            <Zap className="w-4 h-4" /> Trigger Multi-Crisis
          </button>
          <button onClick={handleDeclareIncident} className="bg-red-500 text-white px-5 py-2 rounded-lg text-xs font-bold hover:bg-red-600 transition-colors shadow-sm shadow-red-500/20">
            Declare Incident
          </button>
        </div>
      }
    >
      <div className="p-4 lg:p-5 flex flex-col lg:flex-row gap-4 lg:gap-5 h-full overflow-y-auto lg:overflow-hidden">
        <SectionCard className="flex-1 overflow-hidden min-h-[300px] lg:min-h-0" noPadding>
          <div className="overflow-x-auto w-full h-full">
            <table className="w-full text-left text-sm whitespace-nowrap min-w-[650px] lg:min-w-0">
              <thead className="bg-muted/50 border-b border-card-border sticky top-0 z-10 transition-colors duration-300">
                <tr>
                  <th className="px-4 lg:px-6 py-3 lg:py-4 font-bold text-muted-foreground text-xs uppercase tracking-wider">ID</th>
                  <th className="px-4 lg:px-6 py-3 lg:py-4 font-bold text-muted-foreground text-xs uppercase tracking-wider">Severity</th>
                  <th className="px-4 lg:px-6 py-3 lg:py-4 font-bold text-muted-foreground text-xs uppercase tracking-wider">Description</th>
                  <th className="px-4 lg:px-6 py-3 lg:py-4 font-bold text-muted-foreground text-xs uppercase tracking-wider">Location</th>
                  <th className="px-4 lg:px-6 py-3 lg:py-4 font-bold text-muted-foreground text-xs uppercase tracking-wider">Time</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-card-border transition-colors duration-300">
                {isLoading && (
                  <tr>
                    <td colSpan={5} className="px-6 py-12 text-center text-muted-foreground">
                      <Loader2 className="w-6 h-6 animate-spin mx-auto mb-2" />
                      Loading live incident data...
                    </td>
                  </tr>
                )}
                {isError && (
                  <tr>
                    <td colSpan={5} className="px-6 py-12 text-center text-red-500 font-bold">
                      Failed to connect to operations datastore. Check connection.
                    </td>
                  </tr>
                )}
                {incidents?.length === 0 && (
                  <tr>
                    <td colSpan={5} className="px-6 py-12 text-center text-muted-foreground font-bold">
                      No active incidents.
                    </td>
                  </tr>
                )}
                {incidents?.map(inc => {
                  const minsAgo = Math.floor((Date.now() - new Date(inc.created_at).getTime()) / 60000);
                  const isRetracted = inc.status === 'retracted';
                  return (
                    <tr key={inc.id} onClick={() => setSelectedId(inc.id)} className={`hover:bg-muted/50 cursor-pointer transition-all group ${isRetracted ? 'opacity-50 grayscale' : ''} ${selectedId === inc.id ? 'bg-muted/80' : ''}`}>
                      <td className={`px-4 lg:px-6 py-3 lg:py-4 font-bold text-foreground ${isRetracted ? 'line-through' : ''}`}>{inc.external_id}</td>
                      <td className="px-4 lg:px-6 py-3 lg:py-4">
                        <StatusBadge 
                          status={isRetracted ? 'Retracted' : inc.severity.charAt(0).toUpperCase() + inc.severity.slice(1)} 
                          type={isRetracted ? 'default' : (inc.severity === 'critical' ? 'critical' : inc.severity === 'high' ? 'warning' : 'default')} 
                        />
                      </td>
                      <td className="px-4 lg:px-6 py-3 lg:py-4 font-semibold text-foreground/80">{inc.description}</td>
                      <td className="px-4 lg:px-6 py-3 lg:py-4 text-muted-foreground font-medium">{inc.location_name}</td>
                      <td className="px-4 lg:px-6 py-3 lg:py-4 text-muted-foreground font-medium">{minsAgo} mins ago</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </SectionCard>
        
        {(() => {
          const displayIncident = incidents?.find(i => i.id === selectedId) || incidents?.find(i => i.status === 'active') || incidents?.[0];
          if (!displayIncident) return null;
          
          return (
          <SectionCard className="w-full lg:w-[420px] shrink-0 overflow-auto min-h-[300px] lg:min-h-0" title="Incident Detail View">
            <div className="flex flex-col gap-6 lg:gap-8">
               <div>
                  <StatusBadge 
                    status={displayIncident.status === 'retracted' ? 'RETRACTED' : displayIncident.severity.charAt(0).toUpperCase() + displayIncident.severity.slice(1)} 
                    type={displayIncident.status === 'retracted' ? 'default' : (displayIncident.severity === 'critical' ? 'critical' : 'warning')} 
                    className="mb-3 inline-block" 
                  />
                  <h3 className={`text-2xl lg:text-3xl font-bold text-foreground tracking-tight transition-colors duration-300 ${displayIncident.status === 'retracted' ? 'line-through text-muted-foreground' : ''}`}>{displayIncident.external_id}</h3>
                  <p className="text-sm lg:text-base font-semibold text-foreground/90 mt-2 transition-colors duration-300">{displayIncident.description}</p>
                  <p className="text-xs lg:text-sm font-medium text-muted-foreground mt-1 transition-colors duration-300">{displayIncident.location_name} • {displayIncident.status.toUpperCase()}</p>
               </div>
               
               {displayIncident.status === 'retracted' && (
                 <div className="p-4 lg:p-5 bg-muted rounded-xl border border-card-border flex gap-4 transition-colors duration-300">
                    <AlertTriangle className="w-6 h-6 text-muted-foreground shrink-0" />
                    <div>
                       <p className="text-sm font-bold text-foreground">False Alarm Confirmed</p>
                       <p className="text-xs font-medium text-muted-foreground mt-1">This incident has been retracted. All dispatched units have been automatically cancelled and freed.</p>
                    </div>
                 </div>
               )}

               {displayIncident.severity === 'critical' && displayIncident.status !== 'retracted' && (
                 <div className="p-4 lg:p-5 bg-red-50 dark:bg-red-500/10 rounded-xl border border-red-100 dark:border-red-500/20 flex gap-4 transition-colors duration-300">
                    <AlertTriangle className="w-6 h-6 text-red-500 dark:text-red-400 shrink-0" />
                    <div>
                       <p className="text-sm font-bold text-red-900 dark:text-red-300">Multiple Casualties Reported</p>
                       <p className="text-xs font-medium text-red-700 dark:text-red-400/80 mt-1">Priority trauma routing required based on signal intelligence.</p>
                    </div>
                 </div>
               )}
               
               {displayIncident.status !== 'retracted' && displayIncident.status !== 'resolved' && (
                 <div className="space-y-3">
                    <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest transition-colors duration-300">Required Orchestration</p>
                    <button onClick={() => handleDispatch(displayIncident)} className="w-full flex items-center justify-center gap-2 bg-card border-2 border-card-border text-foreground px-4 py-3 rounded-xl text-sm font-bold hover:border-blue-500 hover:text-blue-500 dark:hover:text-blue-400 transition-colors shadow-sm">
                      <UserPlus className="w-4 h-4" /> Dispatch Available Unit
                    </button>
                    <button onClick={() => handleRetract(displayIncident)} className="w-full flex items-center justify-center gap-2 bg-card border-2 border-card-border text-foreground px-4 py-3 rounded-xl text-sm font-bold hover:border-red-500 hover:text-red-500 dark:hover:text-red-400 transition-colors shadow-sm mt-2">
                      <AlertTriangle className="w-4 h-4" /> Retract as False Alarm
                    </button>
                 </div>
               )}
            </div>
          </SectionCard>
          );
        })()}
      </div>
    </AppLayout>
  );
}
