import { useQuery, useQueryClient } from '@tanstack/react-query';
import { supabase, isPlaceholderSupabase } from '@/lib/supabase';
import { Dispatch } from '@/types/database.types';
import { useEffect } from 'react';

const MOCK_DISPATCHES: Dispatch[] = [
  { id: '1', ambulance_id: '1', incident_id: '1', hospital_id: '1', status: 'en_route', created_at: new Date().toISOString(), updated_at: new Date().toISOString() }
];

export function useRealtimeDispatch() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['dispatches'],
    queryFn: async () => {
      if (isPlaceholderSupabase) return MOCK_DISPATCHES;
      
      const { data, error } = await supabase.from('dispatches').select('*').neq('status', 'completed').neq('status', 'cancelled');
      if (error) throw error;
      return data as Dispatch[];
    }
  });

  useEffect(() => {
    if (isPlaceholderSupabase) return;

    const channelName = `dispatches-changes-${Math.random().toString(36).substring(7)}`;
    const channel = supabase.channel(channelName)
      .on('postgres_changes', { event: '*', schema: 'public', table: 'dispatches' }, () => {
        queryClient.invalidateQueries({ queryKey: ['dispatches'] });
        queryClient.invalidateQueries({ queryKey: ['ambulances'] });
        queryClient.invalidateQueries({ queryKey: ['incidents'] });
      })
      .subscribe();
      
    return () => {
      supabase.removeChannel(channel);
    };
  }, [queryClient]);

  return query;
}
