"use client";
import { MapContainer, TileLayer, Marker, Popup, Polyline, ZoomControl } from "react-leaflet";
import "leaflet/dist/leaflet.css";
import L from "leaflet";
import { useTheme } from "@/components/theme-provider";
import { useIncidents } from "@/hooks/useIncidents";
import { useHospitals } from "@/hooks/useHospitals";
import { useAmbulances } from "@/hooks/useAmbulances";
import { useRealtimeDispatch } from "@/hooks/useRealtimeDispatch";
import { useNetworkStore } from "@/hooks/useNetworkStore";
import { calculateDistance } from "@/services/routing-engine";
import { getRoute } from "@/lib/osrm";
import { useEffect, useState } from "react";

// Beautiful custom HTML badge icons for the map
const createAmbulanceIcon = (unitNumber: string, status: string) => {
  const color = status === 'available' ? '#10b981' : status === 'en_route' ? '#f59e0b' : '#3b82f6';
  return L.divIcon({
    className: 'custom-div-icon',
    html: `
      <div style="
        display: flex;
        align-items: center;
        gap: 6px;
        background-color: #0f172a;
        color: #f8fafc;
        border: 2px solid ${color};
        padding: 4px 10px;
        border-radius: 9999px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4), 0 0 10px ${color}80;
        font-family: ui-sans-serif, system-ui, sans-serif;
        font-weight: 800;
        font-size: 11px;
        white-space: nowrap;
        transform: translate(-50%, -55%);
        transition: all 0.2s ease-in-out;
      " class="hover:scale-105">
        <span style="font-size: 13px;">🚑</span>
        <span>${unitNumber}</span>
      </div>
    `,
    iconSize: [1, 1],
    iconAnchor: [0, 0]
  });
};

const createIncidentIcon = (externalId: string, severity: string) => {
  const color = severity === 'critical' ? '#ef4444' : '#f97316';
  const pulseClass = severity === 'critical' ? 'critical-pulse' : 'warning-pulse';
  return L.divIcon({
    className: 'custom-div-icon',
    html: `
      <div style="
        display: flex;
        align-items: center;
        gap: 6px;
        background-color: #0f172a;
        color: #f8fafc;
        border: 2px solid ${color};
        padding: 4px 10px;
        border-radius: 9999px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4), 0 0 10px ${color}80;
        font-family: ui-sans-serif, system-ui, sans-serif;
        font-weight: 800;
        font-size: 11px;
        white-space: nowrap;
        transform: translate(-50%, -55%);
      " class="${pulseClass}">
        <span style="font-size: 13px;">🚨</span>
        <span>${externalId}</span>
      </div>
    `,
    iconSize: [1, 1],
    iconAnchor: [0, 0]
  });
};

const createHospitalIcon = (name: string) => {
  const shortName = name.replace(" Hospital", "").replace(" Medical Center", "").replace(" Clinic", "");
  return L.divIcon({
    className: 'custom-div-icon',
    html: `
      <div style="
        display: flex;
        align-items: center;
        gap: 6px;
        background-color: #0f172a;
        color: #f8fafc;
        border: 2px solid #3b82f6;
        padding: 4px 10px;
        border-radius: 9999px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4), 0 0 10px rgba(59, 130, 246, 0.5);
        font-family: ui-sans-serif, system-ui, sans-serif;
        font-weight: 800;
        font-size: 11px;
        white-space: nowrap;
        transform: translate(-50%, -55%);
      ">
        <span style="font-size: 13px;">🏥</span>
        <span>${shortName}</span>
      </div>
    `,
    iconSize: [1, 1],
    iconAnchor: [0, 0]
  });
};

function RoutingPolyline({ start, end, options }: { start: [number, number], end: [number, number], options: any }) {
  const [positions, setPositions] = useState<[number, number][]>([]);

  useEffect(() => {
    let active = true;
    getRoute({ lat: start[0], lng: start[1] }, { lat: end[0], lng: end[1] })
      .then(route => {
        if (active) setPositions(route);
      })
      .catch(() => {
        if (active) setPositions([start, end]);
      });
    return () => { active = false; };
  }, [start[0], start[1], end[0], end[1]]);

  if (positions.length === 0) return null;

  return <Polyline positions={positions} pathOptions={options} />;
}

function MovingAmbulanceMarker({ 
  amb, 
  incidents, 
  hospitals, 
  isDark, 
  isTrafficCongested 
}: { 
  amb: any; 
  incidents: any[]; 
  hospitals: any[]; 
  isDark: boolean; 
  isTrafficCongested: boolean;
}) {
  const [currentPos, setCurrentPos] = useState<[number, number]>([amb.latitude, amb.longitude]);

  // Handle local state movement animation along OSRM route geometry
  useEffect(() => {
    let active = true;
    
    let destLat: number | undefined;
    let destLng: number | undefined;
    
    if (amb.status === 'en_route' || amb.status === 'dispatched') {
      const inc = incidents.find(i => i.id === amb.incident_id);
      destLat = inc?.latitude;
      destLng = inc?.longitude;
    } else if (amb.status === 'transporting') {
      const hosp = hospitals.find(h => h.id === amb.destination_id);
      destLat = hosp?.latitude;
      destLng = hosp?.longitude;
    }

    if (!destLat || !destLng) {
      setCurrentPos([amb.latitude, amb.longitude]);
      return;
    }

    getRoute({ lat: amb.latitude, lng: amb.longitude }, { lat: destLat, lng: destLng })
      .then(route => {
        if (!active || route.length === 0) return;
        
        let index = 0;
        const totalDuration = 12000; // Complete full animation along route in 12s
        const step = Math.max(1, Math.floor(route.length / 60)); // max 60 animation points
        const animRoute = route.filter((_, i) => i % step === 0);
        const intervalTime = Math.max(100, Math.floor(totalDuration / animRoute.length));
        
        const timer = setInterval(() => {
          if (!active) {
            clearInterval(timer);
            return;
          }
          if (index >= animRoute.length) {
            setCurrentPos(animRoute[animRoute.length - 1]);
            clearInterval(timer);
            return;
          }
          setCurrentPos(animRoute[index]);
          index++;
        }, intervalTime);

        return () => {
          clearInterval(timer);
        };
      })
      .catch(() => {
        // Fallback simple linear interpolation
        let pct = 0;
        const timer = setInterval(() => {
          if (!active) {
            clearInterval(timer);
            return;
          }
          if (pct >= 1) {
            setCurrentPos([destLat!, destLng!]);
            clearInterval(timer);
            return;
          }
          const nextLat = amb.latitude + (destLat! - amb.latitude) * pct;
          const nextLng = amb.longitude + (destLng! - amb.longitude) * pct;
          setCurrentPos([nextLat, nextLng]);
          pct += 0.04;
        }, 150);

        return () => {
          clearInterval(timer);
        };
      });

    return () => { active = false; };
  }, [amb.status, amb.incident_id, amb.destination_id, amb.latitude, amb.longitude, incidents, hospitals]);

  let etaStr = "N/A";
  if (amb.status === 'en_route' || amb.status === 'transporting') {
    const dest = amb.status === 'en_route' 
      ? incidents.find(i => i.id === amb.incident_id) 
      : hospitals.find(h => h.id === amb.destination_id);
    
    if (dest) {
      const dist = calculateDistance(currentPos[0], currentPos[1], dest.latitude, dest.longitude);
      const speed = isTrafficCongested ? 15 : amb.speed_kmh || 40;
      const routeDist = isTrafficCongested ? dist * 1.6 : dist * 1.3;
      const mins = Math.round((routeDist / speed) * 60);
      etaStr = mins > 0 ? `${mins} min` : "Under 1 min";
    }
  }

  return (
    <Marker position={currentPos} icon={createAmbulanceIcon(amb.unit_number, amb.status)}>
      <Popup className={isDark ? 'dark-popup' : ''}>
        <div className="font-bold text-lg">{amb.unit_number}</div>
        <div className="text-xs text-muted-foreground mt-1">Speed: {isTrafficCongested ? 15 : amb.speed_kmh} km/h</div>
        <div className="text-xs text-emerald-500 font-bold mt-1">Status: {amb.status.toUpperCase()}</div>
        {(amb.status === 'en_route' || amb.status === 'transporting') && (
          <div className="text-xs font-black mt-2 bg-muted p-1 rounded border border-card-border">
            <span className={isTrafficCongested ? "text-orange-500 animate-pulse" : "text-foreground"}>
              ETA: {etaStr}
            </span>
          </div>
        )}
      </Popup>
    </Marker>
  );
}

export default function MapImplementation({ center = [40.7306, -73.9852], zoom = 13 }: { center?: [number, number], zoom?: number }) {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const { isTrafficCongested } = useNetworkStore();

  const tileUrl = isDark 
    ? 'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png'
    : 'https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png';

  const { data: incidents = [] } = useIncidents();
  const { data: hospitals = [] } = useHospitals();
  const { data: ambulances = [] } = useAmbulances();
  const { data: dispatches = [] } = useRealtimeDispatch();

  const routeColorActive = isTrafficCongested ? '#f97316' : '#10b981';
  const routeColorTransport = isTrafficCongested ? '#f97316' : '#3b82f6';

  return (
    <div className="w-full h-full relative z-0">
      <style>{`
        .leaflet-container { background: ${isDark ? '#060A14' : '#f1f5f9'}; font-family: inherit; }
        .dark-popup .leaflet-popup-content-wrapper { background: #111827; color: #f8fafc; border: 1px solid #1f2937; border-radius: 12px; }
        .dark-popup .leaflet-popup-tip { background: #111827; }
        .custom-div-icon { background: transparent; border: none; }
        
        @keyframes critPulse {
          0% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.8), 0 4px 12px rgba(0, 0, 0, 0.4); }
          70% { box-shadow: 0 0 0 10px rgba(239, 68, 68, 0), 0 4px 12px rgba(0, 0, 0, 0.4); }
          100% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0), 0 4px 12px rgba(0, 0, 0, 0.4); }
        }
        @keyframes warnPulse {
          0% { box-shadow: 0 0 0 0 rgba(249, 115, 22, 0.8), 0 4px 12px rgba(0, 0, 0, 0.4); }
          70% { box-shadow: 0 0 0 10px rgba(249, 115, 22, 0), 0 4px 12px rgba(0, 0, 0, 0.4); }
          100% { box-shadow: 0 0 0 0 rgba(249, 115, 22, 0), 0 4px 12px rgba(0, 0, 0, 0.4); }
        }
        
        .critical-pulse {
          animation: critPulse 2s infinite;
        }
        .warning-pulse {
          animation: warnPulse 2s infinite;
        }
      `}</style>
      <MapContainer center={center} zoom={zoom} zoomControl={false} style={{ height: '100%', width: '100%' }}>
        <TileLayer
          url={tileUrl}
          attribution='&copy; <a href="https://carto.com/">CartoDB</a>'
          maxZoom={19}
        />
        <ZoomControl position="bottomright" />
        
        {/* Draw Routes based on Active Dispatches */}
        {dispatches.map(dispatch => {
          const amb = ambulances.find(a => a.id === dispatch.ambulance_id);
          const inc = incidents.find(i => i.id === dispatch.incident_id);
          const hosp = hospitals.find(h => h.id === dispatch.hospital_id);
          
          if (!amb) return null;

          return (
            <div key={dispatch.id}>
              {inc && (
                <RoutingPolyline 
                  start={[amb.latitude, amb.longitude]}
                  end={[inc.latitude, inc.longitude]}
                  options={{ color: dispatch.status === 'en_route' ? routeColorActive : '#64748b', weight: isTrafficCongested ? 5 : 4, dashArray: dispatch.status === 'en_route' ? (isTrafficCongested ? '10, 15' : undefined) : '5, 10' }} 
                />
              )}
              {inc && hosp && (
                <RoutingPolyline 
                  start={[inc.latitude, inc.longitude]}
                  end={[hosp.latitude, hosp.longitude]}
                  options={{ color: routeColorTransport, weight: isTrafficCongested ? 5 : 4, dashArray: isTrafficCongested ? '10, 15' : '8, 8', opacity: dispatch.status === 'transporting' ? 1 : 0.4 }} 
                />
              )}
            </div>
          );
        })}

        {incidents.filter(inc => inc.status !== 'retracted').map(inc => (
          <Marker key={inc.id} position={[inc.latitude, inc.longitude]} icon={createIncidentIcon(inc.external_id, inc.severity)}>
            <Popup className={isDark ? 'dark-popup' : ''}>
              <div className="font-bold mb-1">{inc.external_id}</div>
              <div className="text-xs text-red-500 uppercase tracking-wider font-bold">Severity: {inc.severity}</div>
            </Popup>
          </Marker>
        ))}

        {hospitals.map(hosp => (
          <Marker key={hosp.id} position={[hosp.latitude, hosp.longitude]} icon={createHospitalIcon(hosp.name)}>
            <Popup className={isDark ? 'dark-popup' : ''}>
              <div className="font-bold">{hosp.name}</div>
              <div className="text-xs text-emerald-500 mt-1">Accepting {hosp.trauma_level}</div>
            </Popup>
          </Marker>
        ))}

        {ambulances.map(amb => (
          <MovingAmbulanceMarker
            key={amb.id}
            amb={amb}
            incidents={incidents}
            hospitals={hospitals}
            isDark={isDark}
            isTrafficCongested={isTrafficCongested}
          />
        ))}
      </MapContainer>
    </div>
  );
}
