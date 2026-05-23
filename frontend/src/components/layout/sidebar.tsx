import Link from 'next/link';
import { Activity, LayoutDashboard, ShieldAlert, Cpu, Settings, ScrollText, Ambulance } from 'lucide-react';

const routes = [
  { href: '/', label: 'Dashboard', icon: LayoutDashboard },
  { href: '/crisis-feed', label: 'Crisis Feed', icon: ShieldAlert },
  { href: '/ambulances', label: 'Ambulances', icon: Ambulance },
  { href: '/allocation', label: 'Allocation', icon: Activity },
  { href: '/simulations', label: 'Simulations', icon: Cpu },
  { href: '/logs', label: 'System Logs', icon: ScrollText },
  { href: '/settings', label: 'Settings', icon: Settings },
];

export function Sidebar() {
  return (
    <aside className="w-64 border-r border-border bg-card flex flex-col h-full shrink-0">
      <div className="h-16 flex items-center px-6 border-b border-border">
        <div className="flex items-center gap-2 text-primary font-bold text-xl tracking-wider">
          <ShieldAlert className="w-6 h-6" />
          MEDUSA
        </div>
      </div>
      <nav className="flex-1 p-4 space-y-2">
        {routes.map((r) => (
          <Link key={r.href} href={r.href} className="flex items-center gap-3 px-3 py-2 rounded-md hover:bg-border transition-colors text-sm font-medium text-gray-400 hover:text-white">
            <r.icon className="w-4 h-4" />
            {r.label}
          </Link>
        ))}
      </nav>
    </aside>
  );
}
