"use client";
import dynamic from "next/dynamic";
import { ReactNode } from "react";

// Dynamically import the actual map implementation to avoid SSR issues with Leaflet
const MapImplementation = dynamic(() => import("./map-implementation"), {
  ssr: false,
  loading: () => (
    <div className="w-full h-full bg-card flex items-center justify-center border border-card-border">
      <div className="flex flex-col items-center gap-3">
        <span className="w-6 h-6 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
        <span className="text-xs font-bold text-muted-foreground uppercase tracking-widest">Initializing Telemetry...</span>
      </div>
    </div>
  )
});

interface LiveMapProps {
  center?: [number, number];
  zoom?: number;
  className?: string;
  children?: ReactNode; // For absolute positioned overlays
}

export function LiveMap({ center, zoom, className, children }: LiveMapProps) {
  return (
    <div className={`relative w-full h-full overflow-hidden ${className || ""}`}>
      <MapImplementation center={center} zoom={zoom} />
      {children}
    </div>
  );
}
