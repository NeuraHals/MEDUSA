import { Incident, Ambulance, Dispatch, Hospital, IncidentEvent } from '@/types/database.types';
import { supabase } from '@/lib/supabase';
import { calculateDistance, findBestHospital } from './routing-engine';

export interface ReasoningLog {
  timestamp: string;
  action: string;
  reasoning?: string;
  trigger?: string;
}

export class IntelligenceEngine {
  // Simulates autonomous reasoning step
  static async evaluateResourceState(
    incidents: Incident[], 
    ambulances: Ambulance[], 
    dispatches: Dispatch[],
    hospitals: Hospital[]
  ): Promise<{ conflictActive: boolean; reasoningLogs: ReasoningLog[] }> {
    
    let conflictActive = false;
    const reasoningLogs: ReasoningLog[] = [];
    
    const activeIncidents = incidents.filter(i => i.status === 'active');
    
    // Sort active incidents by severity priority
    const severityWeight: Record<string, number> = { 'critical': 100, 'high': 50, 'medium': 20, 'low': 0 };
    activeIncidents.sort((a, b) => severityWeight[b.severity] - severityWeight[a.severity]);

    for (const incident of activeIncidents) {
      // Check if incident has an active dispatch
      const hasDispatch = dispatches.some(d => d.incident_id === incident.id && ['dispatched', 'en_route', 'on_scene', 'transporting'].includes(d.status));
      
      if (!hasDispatch) {
        // Try to allocate an available ambulance first
        const availableAmbulances = ambulances.filter(a => a.status === 'available');
        
        if (availableAmbulances.length > 0) {
          // Standard allocation
          const bestAmbulance = availableAmbulances.reduce((best, current) => {
            const distToBest = calculateDistance(incident.latitude, incident.longitude, best.latitude, best.longitude);
            const distToCurrent = calculateDistance(incident.latitude, incident.longitude, current.latitude, current.longitude);
            return distToCurrent < distToBest ? current : best;
          });
          
          reasoningLogs.push({
            timestamp: new Date().toISOString(),
            action: `Allocated Unit ${bestAmbulance.unit_number}`,
            reasoning: `Standard allocation logic: nearest available unit selected.`
          });
          
        } else {
          // RESOURCE STARVATION - Try Preemption (Reassignment)
          conflictActive = true;
          
          // Find ambulances dispatched to lower severity incidents
          const preemptableDispatches = dispatches.filter(d => {
            if (!['dispatched', 'en_route'].includes(d.status)) return false;
            const currentInc = incidents.find(i => i.id === d.incident_id);
            if (!currentInc) return false;
            return severityWeight[currentInc.severity] < severityWeight[incident.severity];
          });

          if (preemptableDispatches.length > 0) {
            // Pick the one dispatched to the lowest severity
            preemptableDispatches.sort((a, b) => {
              const incA = incidents.find(i => i.id === a.incident_id);
              const incB = incidents.find(i => i.id === b.incident_id);
              return severityWeight[incA!.severity] - severityWeight[incB!.severity];
            });

            const targetDispatch = preemptableDispatches[0];
            const targetAmbulance = ambulances.find(a => a.id === targetDispatch.ambulance_id);
            const oldIncident = incidents.find(i => i.id === targetDispatch.incident_id);

            reasoningLogs.push({
              timestamp: new Date().toISOString(),
              action: `PREEMPTION EXECUTED: Reassigned ${targetAmbulance?.unit_number}`,
              reasoning: `Incident ${incident.external_id} (${incident.severity.toUpperCase()}) overrode ${oldIncident?.external_id} (${oldIncident?.severity.toUpperCase()}). Unit rerouted.`
            });
            
            // In a live system, we would mutate Supabase here to update the dispatch to 'cancelled' 
            // and create a new dispatch for the critical incident.
          } else {
            // Absolute starvation
            reasoningLogs.push({
              timestamp: new Date().toISOString(),
              action: `ESCALATION TRIGGERED`,
              reasoning: `No preemptable units for Critical Incident ${incident.external_id}. Mutual Aid protocol initiated.`
            });
          }
        }
      }
    }

    return { conflictActive, reasoningLogs };
  }
}
