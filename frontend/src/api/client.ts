import { CrisisEvent, HospitalStatus, SimulationRun, AllocationPressure } from './types';
import { MOCK_ACTIVE_CRISES, MOCK_HOSPITAL_LOAD, MOCK_ALLOCATION_PRESSURE, MOCK_SIMULATIONS } from '@/lib/mock-data';

// Simulate network delay
const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

export const apiClient = {
  getCrises: async (): Promise<CrisisEvent[]> => {
    await delay(800);
    return MOCK_ACTIVE_CRISES.map(c => ({
      id: c.id,
      type: c.type,
      location: c.location,
      severity: c.severity as 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW',
      timestamp: c.time,
      hospital: c.id === 'CR-1029' ? 'Metro Gen' : 'St. Jude',
      confidenceScore: 94,
      isActive: true
    }));
  },
  
  getHospitals: async (): Promise<HospitalStatus[]> => {
    await delay(600);
    return MOCK_HOSPITAL_LOAD.map((h, i) => ({
      id: `h-${i}`,
      name: h.name,
      currentLoad: h.current,
      maxCapacity: h.max
    }));
  },

  getAllocationPressure: async (): Promise<AllocationPressure[]> => {
    await delay(1000);
    return MOCK_ALLOCATION_PRESSURE;
  },

  getSimulations: async (): Promise<SimulationRun[]> => {
    await delay(700);
    return MOCK_SIMULATIONS as SimulationRun[];
  }
};
