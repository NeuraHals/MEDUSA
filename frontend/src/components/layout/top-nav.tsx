import { Bell, Search, UserCircle } from 'lucide-react';

export function TopNav() {
  return (
    <header className="h-16 border-b border-border bg-card flex items-center justify-between px-6 shrink-0">
      <div className="flex items-center gap-4 flex-1">
        <div className="relative w-96">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
          <input 
            type="text" 
            placeholder="Search hospitals, crises, or resource tags..." 
            className="w-full bg-background border border-border rounded-md pl-10 pr-4 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary text-foreground placeholder:text-gray-500"
          />
        </div>
      </div>
      <div className="flex items-center gap-4">
        <button className="relative p-2 text-gray-400 hover:text-white transition-colors">
          <Bell className="w-5 h-5" />
          <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-destructive rounded-full"></span>
        </button>
        <div className="flex items-center gap-3 pl-4 border-l border-border">
          <UserCircle className="w-8 h-8 text-gray-400" />
          <div className="text-sm">
            <p className="font-medium">Cmdr. Shepard</p>
            <p className="text-xs text-gray-500">Global Oversight</p>
          </div>
        </div>
      </div>
    </header>
  );
}
