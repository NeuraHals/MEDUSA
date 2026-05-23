import { Incident, Ambulance, Hospital } from '@/types/database.types';
import { supabase, isPlaceholderSupabase } from '@/lib/supabase';

export class EscalationEngine {
  
  static async checkEscalations(incidents: Incident[], ambulances: Ambulance[], hospitals: Hospital[]) {
    if (isPlaceholderSupabase) return;

    for (const incident of incidents) {
      if (incident.status !== 'active') continue;

      // Rule 1: Critical incident with no ambulance assigned after a threshold
      // For simulation, we check if there are no dispatches for this incident.
      const { data: dispatches } = await supabase.from('dispatches').select('id').eq('incident_id', incident.id);
      
      if (!dispatches || dispatches.length === 0) {
        if (incident.severity === 'critical') {
          const minsActive = Math.floor((Date.now() - new Date(incident.created_at).getTime()) / 60000);
          if (minsActive > 2) {
            await this.triggerAlert(`Critical Escalation: INC-${incident.external_id} unassigned for >2 mins`, incident.location_name, 'critical');
          }
        }
      }
    }

    for (const hospital of hospitals) {
      // Rule 2: Hospital Load Escalation
      if (hospital.current_load_pct > 90 && hospital.status !== 'divert') {
        await this.triggerAlert(`Hospital Overload: ${hospital.name} exceeds 90% capacity`, hospital.name, 'high');
      }
    }
  }

  static async triggerAlert(title: string, location: string, severity: 'critical' | 'high' | 'medium' | 'low') {
    // Check if duplicate alert exists recently to avoid spam
    const { data: recent } = await supabase.from('alerts')
      .select('id')
      .eq('title', title)
      .gte('created_at', new Date(Date.now() - 5 * 60000).toISOString());

    if (!recent || recent.length === 0) {
      await supabase.from('alerts').insert([{ title, location, severity }]);
    }
  }
}
