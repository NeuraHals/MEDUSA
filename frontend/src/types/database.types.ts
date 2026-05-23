export interface Incident {
  id: string;
  external_id: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  description: string;
  latitude: number;
  longitude: number;
  location_name: string;
  created_at: string;
  status: 'active' | 'resolved' | 'retracted';
}

export interface Hospital {
  id: string;
  name: string;
  latitude: number;
  longitude: number;
  current_load_pct: number;
  max_capacity: number;
  status: 'accepting' | 'divert' | 'overload';
  trauma_level: 'Level 1' | 'Level 2' | 'Level 3' | 'None';
  has_icu: boolean;
}

export interface Ambulance {
  id: string;
  unit_number: string;
  latitude: number;
  longitude: number;
  status: 'available' | 'dispatched' | 'en_route' | 'on_scene' | 'transporting' | 'arrived';
  speed_kmh: number;
  destination_id: string | null;
  incident_id: string | null;
}

export interface Alert {
  id: string;
  title: string;
  location: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  created_at: string;
}

export interface Dispatch {
  id: string;
  ambulance_id: string;
  incident_id: string;
  hospital_id: string | null;
  status: 'dispatched' | 'en_route' | 'on_scene' | 'transporting' | 'arrived' | 'completed' | 'cancelled';
  created_at: string;
  updated_at: string;
}

export interface IncidentEvent {
  id: string;
  incident_id: string;
  event_type: 'created' | 'ambulance_assigned' | 'ambulance_on_scene' | 'transporting' | 'hospital_selected' | 'patient_delivered' | 'resolved' | 'escalated';
  description: string;
  created_at: string;
}
