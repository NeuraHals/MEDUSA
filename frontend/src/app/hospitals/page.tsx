"use client";
import { AppLayout } from "@/components/layout/app-layout";
import { SectionCard, StatusBadge } from "@/components/ui/shared";
import { Search } from "lucide-react";

export default function HospitalsPage() {
  const hospitals = [
    { name: "Metro General", type: "Level 1 Trauma", status: "Over Capacity", load: 98, icu: 95 },
    { name: "St. Jude Medical", type: "Level 2 Trauma", status: "Nominal", load: 65, icu: 70 },
    { name: "City Care Clinic", type: "Urgent Care", status: "Nominal", load: 40, icu: 20 },
    { name: "Mercy Hospital", type: "Level 2 Trauma", status: "High Load", load: 85, icu: 88 },
  ];

  return (
    <AppLayout 
      title="Hospital Monitoring" 
      subtitle="Network capacity and facility readiness"
      rightContent={
        <div className="relative">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <input type="text" placeholder="Search facilities..." className="pl-9 pr-4 py-2 bg-muted/50 border border-card-border rounded-lg text-sm font-medium focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 w-64 text-foreground placeholder:text-muted-foreground transition-colors" />
        </div>
      }
    >
       <div className="p-5 flex-1 overflow-auto">
         <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {hospitals.map((h, i) => (
              <SectionCard key={i} className="hover:shadow-md transition-shadow hover:-translate-y-0.5">
                <div className="flex justify-between items-start mb-6">
                  <div>
                    <h3 className="text-xl font-bold text-foreground tracking-tight">{h.name}</h3>
                    <p className="text-sm font-medium text-muted-foreground mt-1">{h.type}</p>
                  </div>
                  <StatusBadge status={h.status} type={h.load > 90 ? "critical" : h.load > 80 ? "warning" : "success"} />
                </div>
                <div className="space-y-5 bg-muted/30 p-4 rounded-xl border border-card-border transition-colors">
                   <div>
                     <div className="flex justify-between text-xs font-bold text-muted-foreground uppercase tracking-wider mb-2">
                       <span>ER Load</span>
                       <span className={h.load > 90 ? "text-red-500" : "text-foreground"}>{h.load}%</span>
                     </div>
                     <div className="h-2.5 bg-muted rounded-full overflow-hidden">
                       <div className={`h-full rounded-full ${h.load > 90 ? 'bg-red-500' : h.load > 80 ? 'bg-amber-500' : 'bg-emerald-500'}`} style={{width: `${h.load}%`}} />
                     </div>
                   </div>
                   <div>
                     <div className="flex justify-between text-xs font-bold text-muted-foreground uppercase tracking-wider mb-2">
                       <span>ICU Occupancy</span>
                       <span className={h.icu > 90 ? "text-red-500" : "text-foreground"}>{h.icu}%</span>
                     </div>
                     <div className="h-2.5 bg-muted rounded-full overflow-hidden">
                       <div className={`h-full rounded-full ${h.icu > 90 ? 'bg-red-500' : h.icu > 80 ? 'bg-amber-500' : 'bg-emerald-500'}`} style={{width: `${h.icu}%`}} />
                     </div>
                   </div>
                </div>
              </SectionCard>
            ))}
         </div>
       </div>
    </AppLayout>
  );
}
