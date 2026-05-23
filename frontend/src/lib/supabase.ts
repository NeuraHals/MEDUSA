import { createClient } from '@supabase/supabase-js';

// Initialize Supabase client
// Replace these with actual environment variables in production
const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL || 'https://placeholder.supabase.co';
const supabaseKey = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY || 'placeholder-key';

// True when running with mock/demo data (no real Supabase connection)
export const isPlaceholderSupabase = supabaseUrl.includes('placeholder');

export const supabase = createClient(supabaseUrl, supabaseKey);

// Architecture Note:
// The real-time mapping engine listens to the 'ambulances' and 'incidents' tables.
// When GPS coordinates update on the backend, Supabase realtime channels broadcast the new pos.
// 
// Example subscription:
// supabase.channel('public:ambulances')
//   .on('postgres_changes', { event: 'UPDATE', schema: 'public', table: 'ambulances' }, payload => {
//      updateAmbulancePosition(payload.new.id, payload.new.latitude, payload.new.longitude);
//   }).subscribe();
