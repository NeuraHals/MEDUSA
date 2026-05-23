import { useQuery, useQueryClient } from '@tanstack/react-query';
import { supabase, isPlaceholderSupabase } from '@/lib/supabase';
import { Ambulance } from '@/types/database.types';
import { useEffect } from 'react';

const MOCK_AMBULANCES: Ambulance[] = [
  { id: '1', unit_number: 'AMB-047', latitude: 40.7306, longitude: -73.9852, status: 'en_route', speed_kmh: 72, destination_id: '1', incident_id: '1' }
];

export function useAmbulances() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['ambulances'],
    queryFn: async () => {
      if (isPlaceholderSupabase) return MOCK_AMBULANCES;
      
      const { data, error } = await supabase.from('ambulances').select('*');
      if (error) throw error;
      return data as Ambulance[];
    }
  });

  useEffect(() => {
    if (isPlaceholderSupabase) return;

    const channelName = `ambulances-changes-${Math.random().toString(36).substring(7)}`;
    const channel = supabase.channel(channelName)
      .on('postgres_changes', { event: '*', schema: 'public', table: 'ambulances' }, (payload) => {
        console.log('Realtime Update (Ambulances):', payload);
        queryClient.invalidateQueries({ queryKey: ['ambulances'] });
      })
      .subscribe();
      
    return () => {
      supabase.removeChannel(channel);
    };
  }, [queryClient]);

  return query;
}
