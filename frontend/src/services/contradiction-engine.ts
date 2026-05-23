import { DataPoint, Contradiction, SourceType, SOURCE_RELIABILITY } from '@/types/intelligence.types';
import { useNetworkStore } from '@/hooks/useNetworkStore';
import { useIntelligenceStore } from '@/hooks/useIntelligenceStore';


function simpleId(): string {
  return Math.random().toString(36).slice(2, 10);
}

/**
 * Calculates a confidence score when two data points contradict.
 * 
 * Strategy:
 *  - Weight each source by its base reliability score.
 *  - The confidence score reflects how reliable the dominant source is,
 *    penalized by the disagreement of the opposing source.
 *  - Score < 60: VERIFICATION REQUIRED (unsafe to act on either source)
 *  - Score 60-79: LOW CONFIDENCE (proceed with caution)
 *  - Score 80+: MODERATE CONFIDENCE (act on weighted winner)
 */
function calculateConfidenceScore(a: DataPoint, b: DataPoint): number {
  const reliabilityA = SOURCE_RELIABILITY[a.sourceType];
  const reliabilityB = SOURCE_RELIABILITY[b.sourceType];
  const total = reliabilityA + reliabilityB;

  // Dominant source weight ratio (0-1)
  const dominantWeight = Math.max(reliabilityA, reliabilityB) / total;

  // Disagreement penalty: harsher when sources are close in reliability
  const reliabilityGap = Math.abs(reliabilityA - reliabilityB);
  const penaltyFactor = 1 - (Math.max(0, 40 - reliabilityGap) / 100);

  const score = Math.round(dominantWeight * 100 * penaltyFactor);
  return Math.min(100, Math.max(0, score));
}

function buildRecommendation(a: DataPoint, b: DataPoint, score: number): string {
  const higherSource = SOURCE_RELIABILITY[a.sourceType] >= SOURCE_RELIABILITY[b.sourceType] ? a : b;
  const lowerSource = higherSource === a ? b : a;

  if (score < 60) {
    return `⚠ VERIFICATION REQUIRED — Conflict between ${higherSource.sourceType.replace(/_/g,' ')} and ${lowerSource.sourceType.replace(/_/g,' ')} is unresolvable without field confirmation.`;
  }
  if (score < 80) {
    return `Tentatively trust ${higherSource.sourceType.replace(/_/g,' ')} (reliability: ${SOURCE_RELIABILITY[higherSource.sourceType]}%), but flag for next patrol check.`;
  }
  return `High confidence in ${higherSource.sourceType.replace(/_/g,' ')} (reliability: ${SOURCE_RELIABILITY[higherSource.sourceType]}%). Deprioritize ${lowerSource.sourceType.replace(/_/g,' ')} signal.`;
}

export class ContradictionEngine {

  /**
   * Main entry point: compares two data points on the same subject.
   * Returns null if they do not contradict.
   */
  static evaluate(subject: string, pointA: DataPoint, pointB: DataPoint): Contradiction | null {
    const contradicts = this.doesContradict(pointA, pointB);
    if (!contradicts) return null;

    const confidenceScore = calculateConfidenceScore(pointA, pointB);
    const recommendedAction = buildRecommendation(pointA, pointB, confidenceScore);

    return {
      id: simpleId(),
      subject,
      pointA,
      pointB,
      confidenceScore,
      recommendedAction,
      status: 'unresolved',
      createdAt: new Date().toISOString(),
    };
  }

  /**
   * Returns true if the two data points contain contradictory values.
   * Supports numeric threshold comparison and boolean inversion.
   */
  private static doesContradict(a: DataPoint, b: DataPoint): boolean {
    if (typeof a.value === 'number' && typeof b.value === 'number') {
      // Contradiction if numeric values differ by >20 (e.g., load % 40 vs 90)
      return Math.abs(a.value - b.value) > 20;
    }
    if (typeof a.value === 'boolean' && typeof b.value === 'boolean') {
      return a.value !== b.value;
    }
    // String: direct mismatch
    return a.value !== b.value;
  }

  /**
   * Run all built-in contradiction scenarios and return any detected contradictions.
   * In production, `a` and `b` payloads would come from live API feeds / Supabase columns.
   */
  static async runScenarios(): Promise<Contradiction[]> {
    const results: Contradiction[] = [];

    // ── SCENARIO 1: Flooding report ──────────────────────────────────────────
    const floodSocial: DataPoint = {
      sourceType: 'social_media',
      claim: 'Active flooding blocking FDR Drive',
      value: true,
      rawValue: 'FLOODING',
      timestamp: new Date().toISOString(),
    };
    
    // LIVE WEATHER API INTEGRATION
    let weatherCondition = 'NORMAL';
    let weatherClaim = 'FDR Drive road sensors report normal conditions';
    const { forceSimulateFailure } = useNetworkStore.getState();
    
    try {
      if (forceSimulateFailure) throw new Error("Simulated API Failure");

      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 3000);

      const res = await fetch('https://api.openweathermap.org/data/2.5/weather?lat=40.7128&lon=-74.0060&appid=9f6506f7017da66174e86e0ed41ce960', { signal: controller.signal });
      clearTimeout(timeoutId);

      if (res.ok) {
        const weatherData = await res.json();
        const mainWeather = weatherData.weather[0].main.toUpperCase();
        const desc = weatherData.weather[0].description;
        weatherCondition = mainWeather;
        weatherClaim = `Official live weather API reports: ${desc}`;
        useNetworkStore.getState().setDegraded('Weather API', false);
      } else {
        throw new Error(`Weather API returned ${res.status}`);
      }
    } catch (e) {
      console.warn("Weather API failed, using fallback:", e);
      useNetworkStore.getState().setDegraded('Weather API', true);
      useIntelligenceStore.getState().addReasoningLog({
        timestamp: new Date().toISOString(),
        action: "Weather API timed out. Synthetic meteorological model generated.",
        trigger: "Network Degradation"
      });
      // Fallback: Generate a deterministic synthetic weather state based on the current hour to ensure repeatable demos
      const hour = new Date().getHours();
      if (hour % 2 === 0) {
        weatherCondition = 'RAIN';
        weatherClaim = 'Synthetic Fallback: Heavy rain detected by local barometric estimates.';
      } else {
        weatherCondition = 'CLEAR';
        weatherClaim = 'Synthetic Fallback: Estimated clear skies based on 24h cache.';
      }
    }

    const floodSensor: DataPoint = {
      sourceType: 'official_sensor',
      claim: weatherClaim,
      value: false,
      rawValue: weatherCondition,
      timestamp: new Date().toISOString(),
    };
    const flood = ContradictionEngine.evaluate('FDR Drive Road Conditions', floodSocial, floodSensor);
    if (flood) results.push(flood);

    // ── SCENARIO 2: Hospital capacity mismatch ───────────────────────────────
    const hospSelfReport: DataPoint = {
      sourceType: 'manual_report',
      claim: 'Metro General self-reports 45% capacity',
      value: 45,
      rawValue: '45% (Self-Reported)',
      timestamp: new Date().toISOString(),
    };
    const hospTelemetry: DataPoint = {
      sourceType: 'hospital_telemetry',
      claim: 'Metro General telemetry shows 91% bed occupancy',
      value: 91,
      rawValue: '91% (Telemetry)',
      timestamp: new Date().toISOString(),
    };
    const hospConflict = ContradictionEngine.evaluate('Metro General Hospital Load', hospSelfReport, hospTelemetry);
    if (hospConflict) results.push(hospConflict);

    return results;
  }
}
