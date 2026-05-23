"use client";
import { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { motion } from "framer-motion";
import { useTheme } from "@/components/theme-provider";

export function SectionCard({
  children,
  className,
  title,
  noPadding = false,
}: {
  children: ReactNode;
  className?: string;
  title?: string;
  noPadding?: boolean;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
      className={cn(
        "bg-card rounded-2xl shadow-sm border border-card-border flex flex-col transition-colors duration-300",
        noPadding ? "" : "p-6 gap-4",
        className
      )}
    >
      {title && (
        <h2
          className={cn(
            "text-[10px] font-bold text-muted-foreground uppercase tracking-widest shrink-0",
            noPadding ? "px-6 pt-6 pb-0" : ""
          )}
        >
          {title}
        </h2>
      )}
      {children}
    </motion.div>
  );
}

export function StatCard({
  title,
  value,
  subtitle,
  icon: Icon,
  color = "blue",
}: {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: React.ElementType;
  color?: "blue" | "red" | "emerald" | "amber";
}) {
  const colorMap = {
    blue: "text-blue-600 bg-blue-50 border-blue-100 dark:bg-blue-500/10 dark:text-blue-400 dark:border-blue-500/20",
    red: "text-red-600 bg-red-50 border-red-100 dark:bg-red-500/10 dark:text-red-400 dark:border-red-500/20",
    emerald: "text-emerald-600 bg-emerald-50 border-emerald-100 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20",
    amber: "text-amber-600 bg-amber-50 border-amber-100 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/20",
  };
  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
      className="bg-card rounded-2xl p-5 shadow-sm border border-card-border flex items-start justify-between transition-colors duration-300 hover:shadow-md hover:-translate-y-0.5"
    >
      <div>
        <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest mb-1.5">
          {title}
        </p>
        <p className="text-3xl font-bold text-foreground tracking-tight">{value}</p>
        {subtitle && (
          <p className="text-xs font-medium text-muted-foreground mt-1">{subtitle}</p>
        )}
      </div>
      <div className={cn("p-2.5 rounded-xl border shrink-0 transition-colors duration-300", colorMap[color])}>
        <Icon className="w-5 h-5" strokeWidth={2} />
      </div>
    </motion.div>
  );
}

export function StatusBadge({
  status,
  type = "default",
  className,
}: {
  status: string;
  type?: "success" | "warning" | "critical" | "default";
  className?: string;
}) {
  const styles = {
    success: "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20",
    warning: "bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/20",
    critical: "bg-red-50 text-red-700 border-red-200 dark:bg-red-500/10 dark:text-red-400 dark:border-red-500/20",
    default: "bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700",
  };
  return (
    <span
      className={cn(
        "px-2.5 py-1 rounded-full text-[10px] font-bold uppercase tracking-wider border inline-block transition-colors duration-300",
        styles[type],
        className
      )}
    >
      {status}
    </span>
  );
}

export function RealisticMapPolygons({ isDark }: { isDark: boolean }) {
  const waterFill = isDark ? "#040914" : "#cbd5e1";
  const landFill = isDark ? "#0B1221" : "#e2e8f0";
  const roadStroke = isDark ? "#162032" : "#f1f5f9";
  const majorRoadStroke = isDark ? "#1E2A40" : "#ffffff";
  const textFill = isDark ? "#2A3A54" : "#94a3b8";

  return (
    <>
      {/* Water Background */}
      <rect width="100%" height="100%" fill={waterFill} />
      
      {/* Jersey City / Hoboken */}
      <path d="M -50 -50 L 280 -50 L 260 150 L 290 300 L 250 490 L -50 490 Z" fill={landFill} />
      
      {/* Manhattan */}
      <path d="M 330 -50 L 480 -50 L 420 180 L 380 320 L 320 440 L 290 440 L 280 200 Z" fill={landFill} />
      
      {/* Brooklyn / Queens */}
      <path d="M 520 -50 L 950 -50 L 950 490 L 380 490 L 440 280 L 460 150 Z" fill={landFill} />

      {/* Grid of minor roads */}
      <g stroke={roadStroke} strokeWidth="1" opacity="0.6">
        {Array.from({ length: 40 }).map((_, i) => (
          <line key={`v${i}`} x1={i * 25} y1={0} x2={i * 25 + 100} y2={440} />
        ))}
        {Array.from({ length: 20 }).map((_, i) => (
          <line key={`h${i}`} x1={0} y1={i * 25} x2={900} y2={i * 25 - 50} />
        ))}
      </g>

      {/* Major arterial roads / highways */}
      <g stroke={majorRoadStroke} strokeWidth="2.5" fill="none" opacity="0.8">
        {/* NJ Turnpike / I-95 */}
        <path d="M 50 -50 L 100 200 L 80 490" />
        <path d="M 120 -50 L 150 250 L 180 490" />
        
        {/* Manhattan Avenues */}
        <path d="M 360 -50 L 310 440" />
        <path d="M 390 -50 L 340 440" />
        <path d="M 420 -50 L 370 440" />
        <path d="M 450 -50 L 400 350 L 380 440" />

        {/* BQE / Queens Blvd */}
        <path d="M 550 -50 L 520 150 L 480 300 L 450 490" />
        <path d="M 950 50 L 700 150 L 550 200 L 480 300" />
        <path d="M 950 250 L 750 300 L 600 380 L 500 490" />
        <path d="M 950 400 L 800 420 L 650 490" />
      </g>

      {/* Bridges / Tunnels */}
      <g stroke={majorRoadStroke} strokeWidth="3" fill="none" opacity="0.9" strokeDasharray="4 4">
        {/* Lincoln Tunnel approx */}
        <path d="M 270 100 L 345 100" />
        {/* Holland Tunnel approx */}
        <path d="M 285 250 L 350 250" />
        {/* Williamsburg Bridge approx */}
        <path d="M 405 280 L 460 280" />
        {/* Queensboro Bridge approx */}
        <path d="M 440 120 L 500 120" />
      </g>

      {/* Region Labels */}
      <text x="120" y="80" fill={textFill} fontSize="24" fontWeight="900" letterSpacing="0.2em" opacity="0.4">JERSEY CITY</text>
      <text x="350" y="200" fill={textFill} fontSize="20" fontWeight="900" letterSpacing="0.3em" opacity="0.6" transform="rotate(80, 350, 200)">MANHATTAN</text>
      <text x="700" y="100" fill={textFill} fontSize="24" fontWeight="900" letterSpacing="0.2em" opacity="0.4">QUEENS</text>
      <text x="650" y="350" fill={textFill} fontSize="24" fontWeight="900" letterSpacing="0.2em" opacity="0.4">BROOKLYN</text>
      
      <text x="310" y="100" fill={waterFill} stroke={textFill} strokeWidth="0.5" fontSize="12" fontWeight="800" letterSpacing="0.1em" opacity="0.8" transform="rotate(80, 310, 100)">HUDSON RIVER</text>
      <text x="440" y="250" fill={waterFill} stroke={textFill} strokeWidth="0.5" fontSize="12" fontWeight="800" letterSpacing="0.1em" opacity="0.8" transform="rotate(75, 440, 250)">EAST RIVER</text>
    </>
  );
}

// Premium DarkMap with realistic NYC geography, glowing routes, and markers
export function DarkMap({ className }: { className?: string }) {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  
  const textColor = isDark ? "#f8fafc" : "#0f172a";

  return (
    <div className={cn("relative overflow-hidden w-full transition-colors duration-500", className)}>
      <svg
        width="100%"
        height="100%"
        viewBox="0 0 900 440"
        preserveAspectRatio="xMidYMid slice"
        className="absolute inset-0"
      >
        <defs>
          <filter id="glow" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="8" result="blur" />
            <feComposite in="SourceGraphic" in2="blur" operator="over" />
          </filter>
        </defs>

        {/* Photorealistic SVG Base */}
        <RealisticMapPolygons isDark={isDark} />
        
        {/* Animated Active Route */}
        <path
          d="M 160 280 C 250 260, 350 250, 450 240 S 600 220, 660 200"
          fill="none" stroke="#3b82f6" strokeWidth="4" strokeLinecap="round" opacity={isDark ? "0.6" : "0.4"} filter="url(#glow)"
        />
        <path
          d="M 160 280 C 250 260, 350 250, 450 240 S 600 220, 660 200"
          fill="none" stroke={isDark ? "#60a5fa" : "#2563eb"} strokeWidth="2" strokeLinecap="round" strokeDasharray="8 6"
        >
           <animate attributeName="stroke-dashoffset" values="14;0" dur="1s" repeatCount="indefinite" />
        </path>

        {/* Origin: Incident */}
        <circle cx="160" cy="280" r="16" fill="#f97316" opacity="0.2" filter="url(#glow)" />
        <circle cx="160" cy="280" r="6" fill="#f97316" />
        <circle cx="160" cy="280" r="12" fill="none" stroke="#f97316" strokeWidth="1.5">
          <animate attributeName="r" values="6;20;6" dur="2s" repeatCount="indefinite" />
          <animate attributeName="opacity" values="0.8;0;0.8" dur="2s" repeatCount="indefinite" />
        </circle>
        
        {/* Destination: Hospital */}
        <circle cx="660" cy="200" r="20" fill="#ef4444" opacity="0.2" filter="url(#glow)" />
        <circle cx="660" cy="200" r="6" fill="#ef4444" />
        
        {/* Moving Ambulance */}
        <g>
          <animateMotion 
            dur="8s" 
            repeatCount="indefinite" 
            path="M 160 280 C 250 260, 350 250, 450 240 S 600 220, 660 200" 
          />
          <circle cx="0" cy="0" r="14" fill="#3b82f6" opacity="0.25" filter="url(#glow)" />
          <circle cx="0" cy="0" r="4" fill={isDark ? "#60a5fa" : "#2563eb"} />
          <circle cx="0" cy="0" r="10" fill="none" stroke={isDark ? "#60a5fa" : "#2563eb"} strokeWidth="1.5">
            <animate attributeName="r" values="4;16;4" dur="1.5s" repeatCount="indefinite" />
            <animate attributeName="opacity" values="0.8;0;0.8" dur="1.5s" repeatCount="indefinite" />
          </circle>
        </g>

        <text x="160" y="255" textAnchor="middle" fill={textColor} fontSize="11" fontWeight="800" letterSpacing="0.05em" filter={isDark ? "url(#glow)" : ""}>INCIDENT ZONE</text>
        <text x="660" y="175" textAnchor="middle" fill={textColor} fontSize="11" fontWeight="800" letterSpacing="0.05em" filter={isDark ? "url(#glow)" : ""}>METRO GENERAL</text>
      </svg>
    </div>
  );
}
