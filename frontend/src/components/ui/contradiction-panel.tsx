"use client";
import { Contradiction } from "@/types/intelligence.types";
import { SOURCE_LABELS, SOURCE_RELIABILITY } from "@/types/intelligence.types";
import { useIntelligenceStore } from "@/hooks/useIntelligenceStore";
import { ShieldAlert, CheckCircle2, XCircle, AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";

function ConfidenceBar({ score }: { score: number }) {
  const color = score < 60 ? "bg-red-500" : score < 80 ? "bg-amber-500" : "bg-emerald-500";
  const label = score < 60 ? "VERIFICATION REQUIRED" : score < 80 ? "LOW CONFIDENCE" : "MODERATE";

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex justify-between items-center">
        <span className={cn(
          "text-[10px] font-black uppercase tracking-widest",
          score < 60 ? "text-red-500 animate-pulse" : score < 80 ? "text-amber-500" : "text-emerald-600"
        )}>
          {label}
        </span>
        <span className="text-sm font-black text-foreground">{score}%</span>
      </div>
      <div className="h-2 bg-muted rounded-full overflow-hidden">
        <div 
          className={cn("h-full rounded-full transition-all duration-700", color)} 
          style={{ width: `${score}%` }} 
        />
      </div>
    </div>
  );
}

function SourceBadge({ sourceType }: { sourceType: string }) {
  const reliability = SOURCE_RELIABILITY[sourceType as keyof typeof SOURCE_RELIABILITY] || 50;
  const color = reliability >= 80 ? "border-emerald-500/30 text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-500/10"
    : reliability >= 60 ? "border-amber-500/30 text-amber-700 dark:text-amber-400 bg-amber-50 dark:bg-amber-500/10"
    : "border-red-500/30 text-red-700 dark:text-red-400 bg-red-50 dark:bg-red-500/10";

  return (
    <span className={cn("inline-flex items-center gap-1 px-2 py-0.5 rounded-full border text-[10px] font-bold uppercase tracking-wider", color)}>
      {SOURCE_LABELS[sourceType as keyof typeof SOURCE_LABELS] || sourceType}
      <span className="opacity-70">· {reliability}%</span>
    </span>
  );
}

function ContradictionCard({ c }: { c: Contradiction }) {
  const { resolveContradiction } = useIntelligenceStore();
  const isUnresolved = c.status === 'unresolved';

  return (
    <div className={cn(
      "rounded-xl border p-5 flex flex-col gap-4 transition-all",
      isUnresolved 
        ? c.confidenceScore < 60 ? "border-red-500/40 bg-red-50/50 dark:bg-red-500/5" : "border-amber-500/30 bg-amber-50/50 dark:bg-amber-500/5"
        : "border-card-border bg-muted/30 opacity-60"
    )}>
      {/* Header */}
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-2">
          <ShieldAlert className={cn("w-5 h-5 mt-0.5 shrink-0", c.confidenceScore < 60 ? "text-red-500" : "text-amber-500")} />
          <div>
            <p className="text-sm font-black text-foreground tracking-tight">{c.subject}</p>
            <p className="text-[10px] font-semibold text-muted-foreground mt-0.5 uppercase tracking-widest">
              {new Date(c.createdAt).toLocaleTimeString()}
            </p>
          </div>
        </div>
        {!isUnresolved && (
          <span className={cn(
            "text-[10px] font-black px-3 py-1 rounded-full border uppercase tracking-wider",
            c.status === 'verified' ? "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20"
              : "bg-muted text-muted-foreground border-card-border"
          )}>
            {c.status}
          </span>
        )}
      </div>

      {/* Confidence Score */}
      <ConfidenceBar score={c.confidenceScore} />

      {/* Contradicting Sources */}
      <div className="grid grid-cols-1 gap-3">
        {[c.pointA, c.pointB].map((point, i) => (
          <div key={i} className="flex flex-col gap-1.5 p-3 rounded-lg bg-background/60 border border-card-border">
            <SourceBadge sourceType={point.sourceType} />
            <p className="text-xs font-semibold text-foreground/90 mt-1">{point.claim}</p>
            <p className="text-sm font-black text-foreground">{point.rawValue}</p>
          </div>
        ))}
      </div>

      {/* Recommendation */}
      <div className={cn(
        "p-3 rounded-lg text-xs font-semibold border",
        c.confidenceScore < 60 
          ? "bg-red-100/80 dark:bg-red-500/10 border-red-200 dark:border-red-500/20 text-red-800 dark:text-red-300"
          : "bg-amber-100/80 dark:bg-amber-500/10 border-amber-200 dark:border-amber-500/20 text-amber-800 dark:text-amber-300"
      )}>
        {c.recommendedAction}
      </div>

      {/* Actions */}
      {isUnresolved && (
        <div className="flex gap-2">
          <button
            onClick={() => resolveContradiction(c.id, 'verified')}
            className="flex-1 flex items-center justify-center gap-1.5 py-2 rounded-lg border border-emerald-500/30 text-emerald-700 dark:text-emerald-400 text-xs font-bold hover:bg-emerald-50 dark:hover:bg-emerald-500/10 transition-colors"
          >
            <CheckCircle2 className="w-3.5 h-3.5" /> Verified OK
          </button>
          <button
            onClick={() => resolveContradiction(c.id, 'dismissed')}
            className="flex-1 flex items-center justify-center gap-1.5 py-2 rounded-lg border border-card-border text-muted-foreground text-xs font-bold hover:bg-muted transition-colors"
          >
            <XCircle className="w-3.5 h-3.5" /> Dismiss
          </button>
        </div>
      )}
    </div>
  );
}

export function ContradictionPanel() {
  const { contradictions } = useIntelligenceStore();
  const active = contradictions.filter(c => c.status === 'unresolved');
  const resolved = contradictions.filter(c => c.status !== 'unresolved');
  const criticalCount = active.filter(c => c.confidenceScore < 60).length;

  if (contradictions.length === 0) return null;

  return (
    <div className="flex flex-col gap-3">
      {/* Panel Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 text-red-500" />
          <span className="text-xs font-black text-foreground uppercase tracking-widest">Intelligence Contradictions</span>
          {criticalCount > 0 && (
            <span className="bg-red-500 text-white text-[10px] font-black px-2 py-0.5 rounded-full animate-pulse">
              {criticalCount} CRITICAL
            </span>
          )}
        </div>
        {active.length > 0 && (
          <span className="text-[10px] font-bold text-muted-foreground">{active.length} unresolved</span>
        )}
      </div>

      {/* Active */}
      {active.map(c => <ContradictionCard key={c.id} c={c} />)}

      {/* Resolved */}
      {resolved.length > 0 && (
        <details className="mt-1">
          <summary className="text-[10px] font-bold text-muted-foreground cursor-pointer select-none uppercase tracking-widest hover:text-foreground transition-colors">
            {resolved.length} resolved
          </summary>
          <div className="flex flex-col gap-3 mt-3">
            {resolved.map(c => <ContradictionCard key={c.id} c={c} />)}
          </div>
        </details>
      )}
    </div>
  );
}
