import { create } from 'zustand';
import { ReasoningLog } from '@/services/intelligence-engine';
import { Contradiction } from '@/types/intelligence.types';

interface IntelligenceState {
  conflictActive: boolean;
  reasoningLogs: ReasoningLog[];
  contradictions: Contradiction[];
  setConflictActive: (active: boolean) => void;
  addReasoningLog: (log: ReasoningLog) => void;
  clearLogs: () => void;
  setContradictions: (contradictions: Contradiction[]) => void;
  resolveContradiction: (id: string, status: 'verified' | 'dismissed') => void;
}

export const useIntelligenceStore = create<IntelligenceState>((set) => ({
  conflictActive: false,
  reasoningLogs: [],
  contradictions: [],
  setConflictActive: (active) => set({ conflictActive: active }),
  addReasoningLog: (log) => set((state) => ({ 
    reasoningLogs: [log, ...state.reasoningLogs].slice(0, 50) 
  })),
  clearLogs: () => set({ reasoningLogs: [] }),
  setContradictions: (contradictions) => set({ contradictions }),
  resolveContradiction: (id, status) => set((state) => ({
    contradictions: state.contradictions.map(c => c.id === id ? { ...c, status } : c)
  })),
}));
