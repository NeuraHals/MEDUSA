export const MOCK_ACTIVE_CRISES = [
  { id: 'CR-1029', type: 'Mass Casualty', location: 'Downtown Metro', severity: 'CRITICAL', time: '10 mins ago' },
  { id: 'CR-1030', type: 'Infrastructure Failure', location: 'St. Jude Hospital', severity: 'HIGH', time: '25 mins ago' },
];

export const MOCK_HOSPITAL_LOAD = [
  { name: 'St. Jude', current: 85, max: 100 },
  { name: 'Metro Gen', current: 95, max: 100 },
  { name: 'City Care', current: 40, max: 100 },
  { name: 'Mercy Hosp', current: 60, max: 100 },
];

export const MOCK_ALLOCATION_PRESSURE = [
  { time: '00:00', load: 45 },
  { time: '04:00', load: 50 },
  { time: '08:00', load: 85 },
  { time: '12:00', load: 92 },
  { time: '16:00', load: 78 },
  { time: '20:00', load: 60 },
];

export const MOCK_SIMULATIONS = [
  { id: 'SIM-A1', scenario: 'Cascade Failure', status: 'Running', successRate: null },
  { id: 'SIM-A2', scenario: 'Grid Blackout', status: 'Completed', successRate: 98.5 },
  { id: 'SIM-B1', scenario: 'Staffing Collapse', status: 'Failed', successRate: 42.1 },
];
