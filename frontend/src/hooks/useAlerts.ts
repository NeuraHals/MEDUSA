import { useQuery, useQueryClient } from '@tanstack/react-query';
import { supabase, isPlaceholderSupabase } from '@/lib/supabase';
import { Alert } from '@/types/database.types';
import { useEffect } from 'react';

const MOCK_ALERTS: Alert[] = [
  { id: '1', title: 'Mass Casualty Protocol', location: 'Downtown Metro', severity: 'critical', created_at: new Date(Date.now() - 12*60000).toISOString() },
  { id: '2', title: 'ER Capacity Warning', location: 'Metro General Hospital', severity: 'high', created_at: new Date(Date.now() - 24*60000).toISOString() },
];

export function useAlerts() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['alerts'],
    queryFn: async () => {
      if (isPlaceholderSupabase) return MOCK_ALERTS;
      
      const { data, error } = await supabase.from('alerts').select('*').order('created_at', { ascending: false }).limit(10);
      if (error) throw error;
      return data as Alert[];
    }
  });

  useEffect(() => {
    if (isPlaceholderSupabase) return;

    const channelName = `alerts-changes-${Math.random().toString(36).substring(7)}`;
    const channel = supabase.channel(channelName)
      .on('postgres_changes', { event: '*', schema: 'public', table: 'alerts' }, (payload) => {
        console.log('Realtime Update (Alerts):', payload);
        queryClient.invalidateQueries({ queryKey: ['alerts'] });
      })
      .subscribe();
      
    return () => {
      supabase.removeChannel(channel);
    };
  }, [queryClient]);

  return query;
}
