import { useNetworkStore } from "@/hooks/useNetworkStore";
import { useIntelligenceStore } from "@/hooks/useIntelligenceStore";

// Cache last known good routes for fallback
const routeCache = new Map<string, [number, number][]>();

export async function getRoute(start: { lat: number; lng: number }, end: { lat: number; lng: number }): Promise<[number, number][]> {
  const cacheKey = `${start.lat.toFixed(3)},${start.lng.toFixed(3)}-${end.lat.toFixed(3)},${end.lng.toFixed(3)}`;
  const { forceSimulateFailure } = useNetworkStore.getState();

  try {
    if (forceSimulateFailure) {
      throw new Error("Simulated API Failure");
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 3000); // 3 second strict timeout

    const url = `https://router.project-osrm.org/route/v1/driving/${start.lng},${start.lat};${end.lng},${end.lat}?overview=full&geometries=geojson`;
    const response = await fetch(url, { signal: controller.signal });
    clearTimeout(timeoutId);
    
    if (!response.ok) {
      throw new Error(`OSRM route failed: ${response.status}`);
    }
    
    const data = await response.json();
    const coords = data.routes[0].geometry.coordinates.map(
      ([lng, lat]: [number, number]) => [lat, lng] as [number, number]
    );

    // Save to cache
    routeCache.set(cacheKey, coords);
    
    // Clear degradation flag if it was set
    useNetworkStore.getState().setDegraded('Routing API', false);

    return coords;
  } catch (error) {
    console.error("OSRM failed, using fallback:", error);
    
    // Notify network store of degradation
    useNetworkStore.getState().setDegraded('Routing API', true);
    
    useIntelligenceStore.getState().addReasoningLog({
      timestamp: new Date().toISOString(),
      action: "Routing API unreachable. Activating local heuristic fallback.",
      trigger: "Network Degradation"
    });

    // Attempt cache fallback
    if (routeCache.has(cacheKey)) {
      return routeCache.get(cacheKey)!;
    }

    // Ultimate fallback: Haversine straight line approximation
    return [
      [start.lat, start.lng],
      [end.lat, end.lng]
    ];
  }
}
