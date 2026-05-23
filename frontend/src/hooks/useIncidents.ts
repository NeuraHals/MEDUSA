import { useQuery, useQueryClient } from '@tanstack/react-query';
import { supabase, isPlaceholderSupabase } from '@/lib/supabase';
import { Incident } from '@/types/database.types';
import { useEffect } from 'react';

// Fallback mock data if Supabase is not configured
const MOCK_INCIDENTS: Incident[] = [
  { id: '1', external_id: 'INC-1021', severity: 'critical', description: 'Mass Transit Collision', latitude: 40.7422, longitude: -74.0043, location_name: 'Sector 41 North', created_at: new Date(Date.now() - 11*60000).toISOString(), status: 'active' },
  { id: '2', external_id: 'INC-1022', severity: 'high', description: 'Structural Fire', latitude: 40.7284, longitude: -73.9857, location_name: 'Sector 42 North', created_at: new Date(Date.now() - 12*60000).toISOString(), status: 'active' }
];

export function useIncidents() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['incidents'],
    queryFn: async () => {
      // Return mock data if Supabase is a placeholder
      if (isPlaceholderSupabase) {
        return MOCK_INCIDENTS;
      }
      
      const { data, error } = await supabase
        .from('incidents')
        .select('*')
        .eq('status', 'active')
        .order('created_at', { ascending: false });
        
      if (error) {
        console.error("Supabase Error (Incidents):", error);
        throw error;
      }
      return data as Incident[];
    }
  });

  useEffect(() => {
    if (isPlaceholderSupabase) return;

    const channelName = `incidents-changes-${Math.random().toString(36).substring(7)}`;
    const channel = supabase.channel(channelName)
      .on('postgres_changes', { event: '*', schema: 'public', table: 'incidents' }, (payload) => {
        console.log('Realtime Update (Incidents):', payload);
        queryClient.invalidateQueries({ queryKey: ['incidents'] });
      })
      .subscribe();
      
    return () => {
      supabase.removeChannel(channel);
    };
  }, [queryClient]);

  return query;
}
