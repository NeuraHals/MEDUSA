"use client";
import { useState } from "react";
import { AppLayout } from "@/components/layout/app-layout";
import { SectionCard, StatusBadge } from "@/components/ui/shared";
import { GitMerge, Check, X, AlertTriangle } from "lucide-react";
import { useIntelligenceStore } from "@/hooks/useIntelligenceStore";
import { useQueryClient } from "@tanstack/react-query";

const INITIAL_BLUEPRINTS = [
  { id: 1021, unit: "Medic 41", destination: "Metro General Hospital", match: 92 },
  { id: 1022, unit: "Rescue 7", destination: "St. Jude Medical", match: 88 },
  { id: 1023, unit: "Medic 12", destination: "City Care Clinic", match: 76 },
  { id: 1024, unit: "AirMed 1", destination: "Mercy Hospital", match: 95 },
];

export default function AllocationPage() {
  const [blueprints, setBlueprints] = useState(INITIAL_BLUEPRINTS);
  const { addReasoningLog } = useIntelligenceStore();
  const queryClient = useQueryClient();

  const handleAccept = (bp: any) => {
    setBlueprints(prev => prev.filter(b => b.id !== bp.id));
    addReasoningLog({
      timestamp: new Date().toISOString(),
      action: `Approved routing blueprint for ${bp.unit} to ${bp.destination}.`,
      trigger: "Operator Approval"
    });

    // Update query cache to make the ambulance move to destination
    const ambulances = queryClient.getQueryData<any[]>(['ambulances']) || [];
    const incidents = queryClient.getQueryData<any[]>(['incidents']) || [];
    const hospitals = queryClient.getQueryData<any[]>(['hospitals']) || [];
    const dispatches = queryClient.getQueryData<any[]>(['dispatches']) || [];

    // Find first available ambulance or any ambulance
    const availableAmb = ambulances.find(a => a.status === 'available') || ambulances[0];
    // Find matching hospital or first
    const targetHosp = hospitals.find(h => h.name.toLowerCase().includes(bp.destination.split(' ')[0].toLowerCase())) || hospitals[0];
    
    // Find or create an incident for the destination point, or use active incident
    let targetInc = incidents.find(i => i.status === 'active');
    if (!targetInc && incidents.length > 0) {
      targetInc = incidents[0];
    } else if (!targetInc) {
      // Create a temporary mock incident at coordinates near center for animation
      targetInc = {
        id: `temp-inc-${bp.id}`,
        external_id: `INC-${bp.id}`,
        severity: 'high',
        description: 'Assigned Dispatch Route',
        latitude: 40.7306 + (Math.random() - 0.5) * 0.02,
        longitude: -73.9852 + (Math.random() - 0.5) * 0.02,
        location_name: bp.destination,
        status: 'active',
        created_at: new Date().toISOString()
      };
      queryClient.setQueryData(['incidents'], [targetInc, ...incidents]);
    }

    if (availableAmb && targetHosp) {
      const updatedAmbulances = ambulances.map(a => {
        if (a.id === availableAmb.id) {
          return {
            ...a,
            unit_number: bp.unit,
            status: 'en_route',
            incident_id: targetInc ? targetInc.id : null,
            destination_id: targetHosp.id,
            speed_kmh: 65
          };
        }
        return a;
      });

      const newDispatch = {
        id: `dispatch-bp-${bp.id}`,
        ambulance_id: availableAmb.id,
        incident_id: targetInc ? targetInc.id : null,
        hospital_id: targetHosp.id,
        status: 'en_route',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };

      queryClient.setQueryData(['ambulances'], updatedAmbulances);
      queryClient.setQueryData(['dispatches'], [...dispatches.filter(d => d.ambulance_id !== availableAmb.id), newDispatch]);
    }
  };

  const handleReject = (bp: any) => {
    setBlueprints(prev => prev.filter(b => b.id !== bp.id));
    addReasoningLog({
      timestamp: new Date().toISOString(),
      action: `Rejected routing blueprint for ${bp.unit}. System will recalculate.`,
      trigger: "Operator Override"
    });
  };

  return (
    <AppLayout title="Resource Allocation" subtitle="Pending transfers and routing blueprints">
      <div className="p-4 lg:p-5 flex flex-col lg:flex-row gap-4 lg:gap-5 h-full min-h-0 overflow-y-auto lg:overflow-hidden">
        <div className="flex-1 flex flex-col gap-4 overflow-visible lg:overflow-auto min-h-0 lg:pr-1">
          {blueprints.length === 0 && (
            <div className="text-center py-20 text-muted-foreground font-bold">
              All routing blueprints resolved.
            </div>
          )}
          {blueprints.map((bp) => (
            <div
              key={bp.id}
              className="bg-card rounded-2xl shadow-sm border border-card-border border-l-4 border-l-blue-500 p-5 flex items-center justify-between hover:shadow-md transition-all cursor-pointer"
            >
              <div className="flex items-center gap-5">
                <div className="w-12 h-12 rounded-xl bg-blue-50 dark:bg-blue-500/10 flex items-center justify-center text-blue-600 dark:text-blue-400 border border-blue-100 dark:border-blue-500/20 shrink-0 transition-colors">
                  <GitMerge className="w-6 h-6" />
                </div>
                <div>
                  <h4 className="text-base font-bold text-foreground">Routing Blueprint #{bp.id}</h4>
                  <p className="text-sm font-medium text-muted-foreground mt-1">
                    Unit {bp.unit} → {bp.destination}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-4 shrink-0">
                <StatusBadge status={`${bp.match}% Match`} type="success" />
                <div className="flex gap-2">
                  <button onClick={() => handleAccept(bp)} className="w-10 h-10 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-100 dark:border-emerald-500/20 flex items-center justify-center hover:bg-emerald-500 hover:text-white dark:hover:bg-emerald-500/30 transition-colors shadow-sm">
                    <Check className="w-5 h-5" />
                  </button>
                  <button onClick={() => handleReject(bp)} className="w-10 h-10 rounded-xl bg-card text-muted-foreground border border-card-border flex items-center justify-center hover:bg-muted hover:text-foreground transition-colors shadow-sm">
                    <X className="w-5 h-5" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>

        <SectionCard className="w-full lg:w-[420px] shrink-0" title="Engine Recommendations">
          <div className="flex flex-col gap-5">
            <div className="bg-amber-50 dark:bg-amber-500/10 rounded-xl p-5 border border-amber-100 dark:border-amber-500/20 flex gap-4 transition-colors">
              <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-bold text-amber-900 dark:text-amber-300">Capacity Warning</p>
                <p className="text-xs font-medium text-amber-700 dark:text-amber-400/80 mt-1.5 leading-relaxed">
                  Optimization model suggests diverting Medic 41 to St. Jude due to projected ER
                  overload at Metro General in 15 minutes.
                </p>
              </div>
            </div>

            <div className="border-t border-card-border pt-5 transition-colors">
              <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest mb-4">Blueprint Metrics</p>
              <div className="space-y-4">
                <div className="flex justify-between">
                  <span className="text-sm font-medium text-muted-foreground">Route Efficiency</span>
                  <span className="text-sm font-bold text-foreground">94%</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm font-medium text-muted-foreground">Load Distribution</span>
                  <span className="text-sm font-bold text-emerald-500">Optimal</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm font-medium text-muted-foreground">Active Blueprints</span>
                  <span className="text-sm font-bold text-foreground">{blueprints.length}</span>
                </div>
              </div>
            </div>
          </div>
        </SectionCard>
      </div>
    </AppLayout>
  );
}
