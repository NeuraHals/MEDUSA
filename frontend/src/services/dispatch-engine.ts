import { Ambulance, Incident, Dispatch, Hospital } from '@/types/database.types';
import { supabase, isPlaceholderSupabase } from '@/lib/supabase';
import { calculateDistance, findBestHospital } from './routing-engine';

export class DispatchEngine {
  
  static findBestAmbulance(incident: Incident, ambulances: Ambulance[]): Ambulance | null {
    const available = ambulances.filter(a => a.status === 'available');
    if (available.length === 0) return null;

    return available.reduce((best, current) => {
      const distToBest = calculateDistance(incident.latitude, incident.longitude, best.latitude, best.longitude);
      const distToCurrent = calculateDistance(incident.latitude, incident.longitude, current.latitude, current.longitude);
      return distToCurrent < distToBest ? current : best;
    });
  }

  static async autoDispatch(incident: Incident, ambulances: Ambulance[], hospitals: Hospital[]) {
    if (isPlaceholderSupabase) {
       console.log('[Mock Mode] Auto Dispatch skipped.');
       return null;
    }

    const bestAmbulance = this.findBestAmbulance(incident, ambulances);
    if (!bestAmbulance) return null;

    const bestHospital = await findBestHospital(incident, hospitals);

    // 1. Create Dispatch Record
    const { data: dispatch, error: dErr } = await supabase.from('dispatches').insert([{
      ambulance_id: bestAmbulance.id,
      incident_id: incident.id,
      hospital_id: bestHospital?.id || null,
      status: 'dispatched'
    }]).select().single();

    if (dErr) throw dErr;

    // 2. Update Ambulance Status
    await supabase.from('ambulances').update({ 
      status: 'dispatched', 
      incident_id: incident.id,
      destination_id: bestHospital?.id || null 
    }).eq('id', bestAmbulance.id);

    // 3. Log Event
    await supabase.from('incident_events').insert([{
      incident_id: incident.id,
      event_type: 'ambulance_assigned',
      description: `Unit ${bestAmbulance.unit_number} dispatched to INC-${incident.external_id}`
    }]);

    return dispatch;
  }

  static async updateDispatchStatus(dispatch: Dispatch, newStatus: Dispatch['status'], ambulanceId: string) {
    if (isPlaceholderSupabase) return;

    // Update dispatch
    await supabase.from('dispatches').update({ status: newStatus }).eq('id', dispatch.id);

    // Map dispatch status to ambulance status
    const ambStatusMap: Record<string, Ambulance['status']> = {
      'dispatched': 'dispatched',
      'en_route': 'en_route',
      'on_scene': 'on_scene',
      'transporting': 'transporting',
      'arrived': 'arrived',
      'completed': 'available',
      'cancelled': 'available'
    };

    const ambStatus = ambStatusMap[newStatus];
    const ambUpdate: Partial<Ambulance> = { status: ambStatus };
    
    if (ambStatus === 'available') {
      ambUpdate.incident_id = null;
      ambUpdate.destination_id = null;
    }

    await supabase.from('ambulances').update(ambUpdate).eq('id', ambulanceId);
  }

  static async retractIncident(incidentId: string) {
    if (isPlaceholderSupabase) {
      console.log(`[Mock Mode] Retracting Incident ${incidentId}`);
      return;
    }

    // 1. Mark incident as retracted
    await supabase.from('incidents').update({ status: 'retracted' }).eq('id', incidentId);

    // 2. Fetch all active dispatches for this incident
    const { data: activeDispatches } = await supabase.from('dispatches')
      .select('*')
      .eq('incident_id', incidentId)
      .in('status', ['dispatched', 'en_route', 'on_scene']);

    if (activeDispatches && activeDispatches.length > 0) {
      // 3. Cancel dispatches
      const dispatchIds = activeDispatches.map(d => d.id);
      await supabase.from('dispatches').update({ status: 'cancelled' }).in('id', dispatchIds);

      // 4. Free up assigned ambulances
      const ambulanceIds = activeDispatches.map(d => d.ambulance_id);
      await supabase.from('ambulances').update({
        status: 'available',
        incident_id: null,
        destination_id: null
      }).in('id', ambulanceIds);
    }

    // 5. Append audit log
    await supabase.from('incident_events').insert([{
      incident_id: incidentId,
      event_type: 'incident_retracted',
      description: 'Incident marked as FALSE ALARM. All active unit dispatches cancelled and units returned to available status.'
    }]);
  }
}
