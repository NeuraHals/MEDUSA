"use client";
import { useState } from "react";
import { AppLayout } from "@/components/layout/app-layout";
import { SectionCard } from "@/components/ui/shared";
import { useIntelligenceStore } from "@/hooks/useIntelligenceStore";

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState('Profile Info');
  const { addReasoningLog } = useIntelligenceStore();

  const handleSave = () => {
    addReasoningLog({
      timestamp: new Date().toISOString(),
      action: `${activeTab} settings saved successfully.`,
      trigger: "Operator Update"
    });
  };

  const TABS = ['Profile Info', 'Notifications', 'Map Preferences', 'Security & Audit'];

  return (
    <AppLayout title="System Settings" subtitle="Global configuration and preferences">
      <div className="p-4 lg:p-5 max-w-5xl mx-auto h-full flex flex-col lg:grid lg:grid-cols-4 gap-6 lg:gap-8 overflow-y-auto lg:overflow-hidden">
        <div className="col-span-1 flex flex-row lg:flex-col gap-2 overflow-x-auto whitespace-nowrap pb-2 lg:pb-0 shrink-0 custom-scrollbar">
          {TABS.map((s) => (
            <button 
              key={s} 
              onClick={() => setActiveTab(s)}
              className={`text-left px-5 py-3.5 rounded-xl text-sm font-bold transition-colors ${activeTab === s ? 'bg-card shadow-sm border border-card-border text-foreground' : 'text-muted-foreground hover:bg-card hover:text-foreground'}`}
            >
              {s}
            </button>
          ))}
        </div>
        
        <div className="col-span-3">
          {activeTab === 'Profile Info' && (
            <SectionCard title="Profile Information" className="space-y-6 lg:space-y-8 animate-in fade-in duration-300">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 lg:gap-6">
                <div>
                  <label className="block text-[10px] font-bold text-muted-foreground uppercase tracking-widest mb-2.5">Full Name</label>
                  <input type="text" defaultValue="Cmdr. Shepard" className="w-full bg-card border border-card-border rounded-xl px-4 py-3 text-sm font-bold text-foreground shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-colors" />
                </div>
                <div>
                  <label className="block text-[10px] font-bold text-muted-foreground uppercase tracking-widest mb-2.5">Clearance Role</label>
                  <input type="text" defaultValue="Global Oversight" disabled className="w-full bg-muted/50 border border-card-border rounded-xl px-4 py-3 text-sm font-bold text-muted-foreground cursor-not-allowed transition-colors" />
                </div>
                <div className="col-span-1 sm:col-span-2">
                  <label className="block text-[10px] font-bold text-muted-foreground uppercase tracking-widest mb-2.5">Email Address</label>
                  <input type="email" defaultValue="c.shepard@medusa-ops.gov" className="w-full bg-card border border-card-border rounded-xl px-4 py-3 text-sm font-bold text-foreground shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-colors" />
                </div>
              </div>
              
              <div className="pt-4 border-t border-card-border flex justify-end transition-colors">
                <button onClick={handleSave} className="bg-foreground text-background px-8 py-3 rounded-xl text-sm font-bold shadow-md hover:opacity-90 transition-all">
                  Save Changes
                </button>
              </div>
            </SectionCard>
          )}

          {activeTab === 'Notifications' && (
            <SectionCard title="Notification Preferences" className="space-y-8 animate-in fade-in duration-300">
              <div className="space-y-4">
                <div className="flex items-center justify-between p-4 border border-card-border rounded-xl">
                  <div>
                    <h4 className="text-sm font-bold text-foreground">Critical Alerts</h4>
                    <p className="text-xs font-medium text-muted-foreground mt-1">Push notifications for high severity events.</p>
                  </div>
                  <input type="checkbox" defaultChecked className="w-4 h-4 accent-blue-500" />
                </div>
                <div className="flex items-center justify-between p-4 border border-card-border rounded-xl">
                  <div>
                    <h4 className="text-sm font-bold text-foreground">System Warnings</h4>
                    <p className="text-xs font-medium text-muted-foreground mt-1">Updates on API degradation or latency.</p>
                  </div>
                  <input type="checkbox" defaultChecked className="w-4 h-4 accent-blue-500" />
                </div>
              </div>
              <div className="pt-4 border-t border-card-border flex justify-end transition-colors">
                <button onClick={handleSave} className="bg-foreground text-background px-8 py-3 rounded-xl text-sm font-bold shadow-md hover:opacity-90 transition-all">Save Changes</button>
              </div>
            </SectionCard>
          )}

          {activeTab === 'Map Preferences' && (
            <SectionCard title="Map Preferences" className="space-y-6 lg:space-y-8 animate-in fade-in duration-300">
              <div className="grid grid-cols-1 gap-4 lg:gap-6">
                <div className="col-span-1">
                  <label className="block text-[10px] font-bold text-muted-foreground uppercase tracking-widest mb-2.5">Default Map Layer</label>
                  <select className="w-full bg-card border border-card-border rounded-xl px-4 py-3 text-sm font-bold text-foreground shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-colors">
                    <option>Dark Mode Vector</option>
                    <option>Satellite Hybrid</option>
                    <option>Traffic Heatmap</option>
                  </select>
                </div>
              </div>
              <div className="pt-4 border-t border-card-border flex justify-end transition-colors">
                <button onClick={handleSave} className="bg-foreground text-background px-8 py-3 rounded-xl text-sm font-bold shadow-md hover:opacity-90 transition-all">Save Changes</button>
              </div>
            </SectionCard>
          )}

          {activeTab === 'Security & Audit' && (
            <SectionCard title="Security & Audit Logs" className="space-y-8 animate-in fade-in duration-300">
              <div className="p-8 text-center text-muted-foreground font-bold border border-dashed border-card-border rounded-xl">
                Audit logs are restricted to Level 4 Clearance.
              </div>
            </SectionCard>
          )}
        </div>
      </div>
    </AppLayout>
  );
}
