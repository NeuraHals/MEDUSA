export interface CrisisEvent {
  id: string;
  type: string;
  location: string;
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
  timestamp: string;
  hospital: string;
  confidenceScore: number;
  isActive: boolean;
  metadata?: {
    source: string;
    description: string;
    casualties?: number;
    resourcePressure?: string;
  };
  recommendations?: {
    id: string;
    action: string;
    confidence: number;
  }[];
  timeline?: {
    time: string;
    event: string;
  }[];
}

export interface HospitalStatus {
  id: string;
  name: string;
  currentLoad: number;
  maxCapacity: number;
}

export interface AllocationPressure {
  time: string;
  load: number;
}

export interface AllocationPlan {
  id: string;
  crisisId: string;
  status: 'PENDING' | 'ACTIVE' | 'COMPLETED';
}

export interface SimulationRun {
  id: string;
  scenario: string;
  status: 'Running' | 'Completed' | 'Failed';
  successRate: number | null;
}
