import { create } from 'zustand';

interface NetworkState {
  isDegraded: boolean;
  failedServices: string[];
  forceSimulateFailure: boolean;
  isTrafficCongested: boolean;
  isPaused: boolean;
  
  setDegraded: (service: string, failed: boolean) => void;
  toggleSimulateFailure: () => void;
  toggleTrafficCongestion: () => void;
  togglePause: () => void;
  resetSimulations: () => void;
}

export const useNetworkStore = create<NetworkState>((set) => ({
  isDegraded: false,
  failedServices: [],
  forceSimulateFailure: false,
  isTrafficCongested: false,
  isPaused: false,
  
  setDegraded: (service, failed) => set((state) => {
    let newFailed = [...state.failedServices];
    if (failed && !newFailed.includes(service)) newFailed.push(service);
    if (!failed) newFailed = newFailed.filter(s => s !== service);
    
    return {
      failedServices: newFailed,
      isDegraded: newFailed.length > 0
    };
  }),
  
  toggleSimulateFailure: () => set((state) => ({ 
    forceSimulateFailure: !state.forceSimulateFailure 
  })),

  toggleTrafficCongestion: () => set((state) => ({
    isTrafficCongested: !state.isTrafficCongested
  })),

  togglePause: () => set((state) => ({
    isPaused: !state.isPaused
  })),

  resetSimulations: () => set({
    isDegraded: false,
    failedServices: [],
    forceSimulateFailure: false,
    isTrafficCongested: false,
    isPaused: false
  }),
}));
