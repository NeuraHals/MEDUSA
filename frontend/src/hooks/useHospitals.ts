import { useQuery, useQueryClient } from '@tanstack/react-query';
import { supabase, isPlaceholderSupabase } from '@/lib/supabase';
import { Hospital } from '@/types/database.types';
import { useEffect } from 'react';

const MOCK_HOSPITALS: Hospital[] = [
  { id: '1', name: 'Metro General', latitude: 40.7186, longitude: -73.9552, current_load_pct: 82, max_capacity: 500, status: 'accepting', trauma_level: 'Level 1', has_icu: true },
  { id: '2', name: 'St. Jude Medical', latitude: 40.7306, longitude: -73.9352, current_load_pct: 65, max_capacity: 350, status: 'accepting', trauma_level: 'Level 2', has_icu: true },
];

export function useHospitals() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['hospitals'],
    queryFn: async () => {
      if (isPlaceholderSupabase) return MOCK_HOSPITALS;
      
      const { data, error } = await supabase.from('hospitals').select('*');
      if (error) throw error;
      return data as Hospital[];
    }
  });

  useEffect(() => {
    if (isPlaceholderSupabase) return;

    const channelName = `hospitals-changes-${Math.random().toString(36).substring(7)}`;
    const channel = supabase.channel(channelName)
      .on('postgres_changes', { event: '*', schema: 'public', table: 'hospitals' }, (payload) => {
        console.log('Realtime Update (Hospitals):', payload);
        queryClient.invalidateQueries({ queryKey: ['hospitals'] });
      })
      .subscribe();
      
    return () => {
      supabase.removeChannel(channel);
    };
  }, [queryClient]);

  return query;
}
