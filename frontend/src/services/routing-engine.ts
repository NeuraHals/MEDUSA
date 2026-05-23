import { Hospital, Incident } from '@/types/database.types';
import { useNetworkStore } from '@/hooks/useNetworkStore';

export interface RouteResult {
  distanceKm: number;
  etaMins: number;
  isStale: boolean;
  source: 'osrm_api' | 'haversine_fallback';
  isRerouted?: boolean;
}

// Fallback: Haversine formula to calculate straight-line distance in km
export function calculateDistance(lat1: number, lon1: number, lat2: number, lon2: number): number {
  const R = 6371; // Earth's radius in km
  const dLat = (lat2 - lat1) * (Math.PI / 180);
  const dLon = (lon2 - lon1) * (Math.PI / 180);
  const a = 
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos(lat1 * (Math.PI / 180)) * Math.cos(lat2 * (Math.PI / 180)) * 
    Math.sin(dLon / 2) * Math.sin(dLon / 2);
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  return R * c;
}

// Simple in-memory cache for routing
const routeCache = new Map<string, RouteResult>();

// Simulates an external OSRM / Google Maps Routing API with retries and caching
export async function getRouteInfo(lat1: number, lon1: number, lat2: number, lon2: number): Promise<RouteResult> {
  const networkStore = useNetworkStore.getState();
  const SERVICE_NAME = 'external_routing_api';
  const cacheKey = `${lat1.toFixed(3)},${lon1.toFixed(3)}-${lat2.toFixed(3)},${lon2.toFixed(3)}-${networkStore.isTrafficCongested}`;

  // Try to return cached response if available
  if (routeCache.has(cacheKey) && !networkStore.isDegraded) {
    return routeCache.get(cacheKey)!;
  }

  let attempt = 0;
  const maxRetries = 2;

  while (attempt <= maxRetries) {
    try {
      if (networkStore.forceSimulateFailure) {
        throw new Error("ERR_CONNECTION_TIMED_OUT");
      }

      await new Promise(resolve => setTimeout(resolve, 300));
      
      if (networkStore.failedServices.includes(SERVICE_NAME)) {
        networkStore.setDegraded(SERVICE_NAME, false);
      }

      const straightLine = calculateDistance(lat1, lon1, lat2, lon2);
      
      // Traffic Congestion Adaptation logic
      const speedKmh = networkStore.isTrafficCongested ? 15 : 40;
      const roadDistance = networkStore.isTrafficCongested ? straightLine * 1.6 : straightLine * 1.3;
      
      const result: RouteResult = {
        distanceKm: roadDistance,
        etaMins: Math.round((roadDistance / speedKmh) * 60),
        isStale: false,
        source: 'osrm_api',
        isRerouted: networkStore.isTrafficCongested
      };

      routeCache.set(cacheKey, result);
      return result;

    } catch (error) {
      attempt++;
      if (attempt <= maxRetries) {
        await new Promise(resolve => setTimeout(resolve, attempt * 500));
      }
    }
  }

  // API Failure -> Trigger Fallback Mode
  networkStore.setDegraded(SERVICE_NAME, true);
  
  if (routeCache.has(cacheKey)) {
    const cached = routeCache.get(cacheKey)!;
    return { ...cached, isStale: true }; 
  }
  
  // Ultimate Fallback: Haversine Estimation
  const straightLine = calculateDistance(lat1, lon1, lat2, lon2);
  const fbSpeed = networkStore.isTrafficCongested ? 15 : 40;
  return {
    distanceKm: straightLine,
    etaMins: Math.round((straightLine / fbSpeed) * 60),
    isStale: true,
    source: 'haversine_fallback',
    isRerouted: networkStore.isTrafficCongested
  };
}

export async function findBestHospital(incident: Incident, hospitals: Hospital[]): Promise<Hospital | null> {
  const availableHospitals = hospitals.filter(h => h.status !== 'divert' && h.status !== 'overload');
  
  if (availableHospitals.length === 0) return null;

  let best: Hospital = availableHospitals[0];
  let bestScore = Infinity;

  for (const current of availableHospitals) {
    const route = await getRouteInfo(incident.latitude, incident.longitude, current.latitude, current.longitude);
    const distToCurrent = route.distanceKm;
    
    const scoreCurrent = distToCurrent + (current.current_load_pct / 10);
    
    let finalScore = scoreCurrent;
    if (incident.severity === 'critical' && current.trauma_level === 'Level 1') {
      finalScore -= 20;
    }

    if (finalScore < bestScore) {
      bestScore = finalScore;
      best = current;
    }
  }

  return best;
}
