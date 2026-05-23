import { useNetworkStore } from "@/hooks/useNetworkStore";
import { useIntelligenceStore } from "@/hooks/useIntelligenceStore";
import { ContradictionEngine } from "./contradiction-engine";

export type ScenarioType =
  | 'flood'
  | 'multi-vehicle'
  | 'mci'
  | 'evacuation'
  | 'hospital-overload'
  | 'wildfire'
  | 'false-alarm'
  | 'api-outage'
  | 'contradiction';

function log(action: string, trigger: string) {
  useIntelligenceStore.getState().addReasoningLog({
    timestamp: new Date().toISOString(),
    action,
    trigger,
  });
}

function injectIncident(queryClient: any, incident: any) {
  queryClient.setQueryData(['incidents'], (old: any) => {
    if (!old) return [incident];
    return [incident, ...old.filter((i: any) => i.id !== incident.id)];
  });
}

function injectMultipleIncidents(queryClient: any, incidents: any[]) {
  queryClient.setQueryData(['incidents'], (old: any) => {
    const existing = old || [];
    const ids = new Set(incidents.map((i: any) => i.id));
    return [...incidents, ...existing.filter((i: any) => !ids.has(i.id))];
  });
}

export class ScenarioEngine {
  static async execute(scenario: ScenarioType, queryClient: any) {
    // Always reset before a new scenario for deterministic, non-overlapping demos
    useNetworkStore.getState().resetSimulations();
    useIntelligenceStore.getState().clearLogs();
    useIntelligenceStore.getState().setConflictActive(false);
    useIntelligenceStore.getState().setContradictions([]);

    switch (scenario) {

      // ─── FLOOD RESPONSE ────────────────────────────────────────────────────
      case 'flood': {
        log("OBSERVE: Flash flood alert received from emergency services sensor grid.", "Flash Flood Warning — FDR Drive");
        await delay(400);
        log("REASON: Social media reports flooding. Official sensor API shows water level 40cm above critical threshold.", "Contradiction Analysis");
        await delay(400);
        log("DECIDE: Deploying 2 swift-water rescue units. Blocking southern route segments.", "Routing Engine Decision");
        await delay(400);
        log("ACT: AMB-301 & AMB-302 rerouted via 9th Avenue corridor. ETA recalculated: +4 min.", "Dispatch Execution");
        await delay(400);
        log("ADAPT: Hospital intake at Metro General pre-alerted. Standby trauma team activated.", "Hospital Coordination");

        injectMultipleIncidents(queryClient, [
          { id: 'flood-1', external_id: 'FLD-001', severity: 'critical', description: 'Severe Flash Flood — Road Submerged', latitude: 40.7128, longitude: -74.006, location_name: 'FDR Drive @ 23rd St', status: 'active', created_at: new Date().toISOString() },
          { id: 'flood-2', external_id: 'FLD-002', severity: 'high', description: 'Stranded Vehicles in Flood Water', latitude: 40.7050, longitude: -74.012, location_name: 'Battery Park Underpass', status: 'active', created_at: new Date().toISOString() },
        ]);
        useNetworkStore.getState().toggleTrafficCongestion();
        break;
      }

      // ─── MULTI-VEHICLE COLLISION ───────────────────────────────────────────
      case 'multi-vehicle': {
        log("OBSERVE: Traffic sensor reports 12-vehicle collision on I-495. Lane closures confirmed.", "Multi-Vehicle Collision — I-495");
        await delay(400);
        log("REASON: 3 critical injuries detected. Nearest available unit AMB-047 is 6.2km away. Next unit AMB-205 is 8.1km.", "Resource Conflict Analysis");
        await delay(400);
        log("DECIDE: Preempting AMB-047 from low-priority call INC-1022. Assigning to MVC-992.", "Conflict Resolution — Priority Override");
        await delay(400);
        log("ACT: AMB-047 rerouted. AMB-205 dispatched as backup. Metro General trauma bay pre-cleared.", "Dispatch Execution");
        await delay(400);
        log("ADAPT: AMB-047 ETA 8 min. AMB-205 ETA 12 min. Congestion reroute adds +2 min buffer.", "ETA Recalculation");

        injectMultipleIncidents(queryClient, [
          { id: 'mvc-1', external_id: 'MVC-992', severity: 'critical', description: '12-Car Pileup — Critical Injuries', latitude: 40.735, longitude: -73.99, location_name: 'I-495 Eastbound', status: 'active', created_at: new Date().toISOString() },
          { id: 'mvc-2', external_id: 'MVC-993', severity: 'high', description: 'Secondary Collision — Rubbernecking', latitude: 40.738, longitude: -73.988, location_name: 'I-495 Westbound', status: 'active', created_at: new Date().toISOString() },
        ]);
        useIntelligenceStore.getState().setConflictActive(true);
        break;
      }

      // ─── MASS CASUALTY INCIDENT ────────────────────────────────────────────
      case 'mci': {
        log("OBSERVE: Mass Casualty Incident declared. Estimated 40+ casualties at concert venue.", "MCI DECLARED — Sector 7");
        await delay(400);
        log("REASON: System has 5 available ambulances. 8 hospitals in network. Metro General at 88% — HIGH RISK.", "Capacity Analysis");
        await delay(400);
        log("DECIDE: Activating MCI Protocol. Distributing patients: 30% Metro General, 40% St. Jude, 30% City Care.", "Load Balancing Algorithm");
        await delay(400);
        log("ACT: All 5 available units dispatched. Staging area established at venue entrance.", "Mass Dispatch Execution");
        await delay(400);
        log("ADAPT: Diverting minor injuries to urgent care. Reserving critical bays at trauma centers.", "Hospital Intake Optimization");
        await delay(400);
        log("ESCALATE: Requesting mutual aid from adjacent county. ETA 18 minutes.", "Resource Escalation");

        injectMultipleIncidents(queryClient, [
          { id: 'mci-1', external_id: 'MCI-001', severity: 'critical', description: 'Mass Casualty Event — Venue Collapse', latitude: 40.7580, longitude: -73.9855, location_name: 'Madison Square Garden', status: 'active', created_at: new Date().toISOString() },
          { id: 'mci-2', external_id: 'MCI-002', severity: 'high', description: 'Crowd Crush — Secondary Zone', latitude: 40.7570, longitude: -73.9870, location_name: 'MSG Plaza Entrance', status: 'active', created_at: new Date().toISOString() },
          { id: 'mci-3', external_id: 'MCI-003', severity: 'high', description: 'Cardiac Events — Bystanders', latitude: 40.7560, longitude: -73.9840, location_name: 'MSG West Side', status: 'active', created_at: new Date().toISOString() },
        ]);
        useIntelligenceStore.getState().setConflictActive(true);
        useNetworkStore.getState().toggleTrafficCongestion();
        break;
      }

      // ─── EVACUATION CONGESTION ─────────────────────────────────────────────
      case 'evacuation': {
        log("OBSERVE: City-wide evacuation order issued. Traffic sensors detecting severe gridlock.", "Evacuation Congestion Detected");
        await delay(400);
        log("REASON: Average ambulance speed reduced from 65km/h to 18km/h. All primary routes blocked.", "Traffic Impact Assessment");
        await delay(400);
        log("DECIDE: Switching to emergency corridor routing. Requesting police escort for critical units.", "Adaptive Routing Decision");
        await delay(400);
        log("ACT: 4 ambulances redirected via designated emergency lanes. ETAs extended +8 minutes.", "Dynamic Rerouting Active");
        await delay(400);
        log("ADAPT: Staggering public alerts to reduce simultaneous hospital arrivals. Overload risk reduced.", "Hospital Overload Prevention");

        useNetworkStore.getState().toggleTrafficCongestion();
        injectIncident(queryClient, {
          id: 'evac-1', external_id: 'EVAC-001', severity: 'high',
          description: 'Evacuation Emergency — Stranded Patients',
          latitude: 40.730, longitude: -74.000, location_name: 'Holland Tunnel Approach',
          status: 'active', created_at: new Date().toISOString()
        });
        break;
      }

      // ─── HOSPITAL OVERLOAD ─────────────────────────────────────────────────
      case 'hospital-overload': {
        log("OBSERVE: Metro General ER capacity at 97%. ICU at 100%. Overflow imminent.", "Hospital Capacity Alert");
        await delay(400);
        log("REASON: 4 active dispatches routed to Metro General. Combined load will exceed physical capacity.", "Intake Projection Analysis");
        await delay(400);
        log("DECIDE: Diverting all non-critical inbound units to St. Jude Medical. Halting new trauma assignments.", "Hospital Diversion Decision");
        await delay(400);
        log("ACT: AMB-101, AMB-204 destination changed → St. Jude. AMB-307 destination changed → City Care.", "Dispatch Redirection");
        await delay(400);
        log("ADAPT: Notifying receiving hospitals of inbound load. Pre-clearing trauma bays.", "Network-wide Coordination");

        injectIncident(queryClient, {
          id: 'hosp-1', external_id: 'HOV-001', severity: 'critical',
          description: 'Hospital Overload — Diversion Active',
          latitude: 40.7422, longitude: -74.0043, location_name: 'Metro General ER',
          status: 'active', created_at: new Date().toISOString()
        });
        break;
      }

      // ─── WILDFIRE SPREAD ──────────────────────────────────────────────────
      case 'wildfire': {
        log("OBSERVE: Wildfire spread detected across 3km front. Wind speed 45km/h NE direction.", "Rapid Wildfire Spread");
        await delay(400);
        log("REASON: 2 residential zones in projected fire path. Evacuation window: 12 minutes.", "Threat Analysis");
        await delay(400);
        log("DECIDE: Establishing safe staging zones upwind. Routing all units via northern corridors.", "Safe Zone Decision");
        await delay(400);
        log("ACT: AMB-408 & AMB-511 repositioned to staging area. Burn victim protocol activated.", "Fire Response Execution");
        await delay(400);
        log("ADAPT: Monitoring fire progression. Dynamic re-staging if wind direction shifts.", "Continuous Adaptation");

        injectMultipleIncidents(queryClient, [
          { id: 'fire-1', external_id: 'FIRE-100', severity: 'critical', description: 'Uncontained Wildfire — Residential Zone', latitude: 40.76, longitude: -73.95, location_name: 'North Woods Sector', status: 'active', created_at: new Date().toISOString() },
          { id: 'fire-2', external_id: 'FIRE-101', severity: 'high', description: 'Smoke Inhalation Casualties', latitude: 40.762, longitude: -73.948, location_name: 'Downwind Residential', status: 'active', created_at: new Date().toISOString() },
        ]);
        useNetworkStore.getState().toggleTrafficCongestion();
        break;
      }

      // ─── FALSE ALARM RETRACTION ────────────────────────────────────────────
      case 'false-alarm': {
        const allIncidents: any[] = queryClient.getQueryData(['incidents']) || [];
        const targetInc = allIncidents.find((i: any) => i.status === 'active');

        if (targetInc) {
          log(`OBSERVE: Ground unit reports no evidence of emergency at ${targetInc.location_name}.`, "Field Verification");
          await delay(400);
          log(`REASON: Sensor data cross-referenced. ${targetInc.external_id} flagged as likely false positive.`, "Intelligence Validation");
          await delay(400);
          log(`DECIDE: Retracting incident ${targetInc.external_id}. Issuing stand-down order to all assigned units.`, "False Alarm Confirmed");
          await delay(400);
          log("ACT: Dispatched units recalled. Hospital standby cancelled. Units marked available.", "Retraction Execution");
          await delay(400);
          log("ADAPT: Incident logged as FALSE ALARM in audit trail. No data purged — full history preserved.", "Audit Log Updated");

          queryClient.setQueryData(['incidents'], (old: any) => {
            if (!old) return [];
            return old.map((i: any) => i.id === targetInc.id ? { ...i, status: 'retracted' } : i);
          });
        } else {
          log("No active incidents found. Trigger another scenario first, then retract.", "System Notice");
        }
        break;
      }

      // ─── API OUTAGE ───────────────────────────────────────────────────────
      case 'api-outage': {
        log("OBSERVE: External network connectivity lost. OSRM Routing API unreachable.", "Network Blackout Detected");
        await delay(400);
        log("REASON: Last successful API response was 12 seconds ago. Timeout threshold exceeded.", "API Health Monitor");
        await delay(400);
        log("DECIDE: Activating offline heuristic engine. Switching to Haversine distance calculations.", "Fallback Mode Decision");
        await delay(400);
        log("ACT: Routing API → LOCAL FALLBACK. Weather API → SYNTHETIC MODEL. Map tiles → CACHED.", "Degraded Mode Active");
        await delay(400);
        log("ADAPT: All dispatches proceeding on estimated routes. No operations halted. System stable.", "Graceful Degradation Confirmed");

        // Mark services as failed — this triggers DEGRADED MODE banner
        useNetworkStore.getState().setDegraded('Routing API', true);
        useNetworkStore.getState().setDegraded('Weather API', true);

        injectIncident(queryClient, {
          id: 'outage-1', external_id: 'SYS-ERR-01', severity: 'high',
          description: 'Network Blackout — Fallback Mode Active',
          latitude: 40.7580, longitude: -73.9855, location_name: 'System Status',
          status: 'active', created_at: new Date().toISOString()
        });
        break;
      }

      // ─── INTELLIGENCE CONTRADICTION ────────────────────────────────────────
      case 'contradiction': {
        log("OBSERVE: Conflicting data signals detected from multiple intelligence sources.", "Contradiction Engine Scan");
        await delay(400);
        log("REASON: Social media reports active flooding. Official road sensor API reports normal conditions.", "Source Reliability Analysis");
        await delay(400);

        const contradictions = await ContradictionEngine.runScenarios();
        useIntelligenceStore.getState().setContradictions(contradictions);

        const floodC = contradictions.find(c => c.subject.includes('Flood'));
        const hospC = contradictions.find(c => c.subject.includes('Hospital'));

        if (floodC) {
          log(`DECIDE: Cannot act on unverified flood data. Confidence score: ${floodC.confidenceScore}%. Holding dispatch.`, "Contradiction Flagged — Pending Verification");
        }
        if (hospC) {
          log(`DECIDE: Hospital self-report conflicts with telemetry. Trusting sensor data. Diverting inbound units.`, "Hospital Data Conflict Resolved");
        }
        await delay(400);
        log("ACT: Contradiction alerts surfaced to human commander. Awaiting field confirmation.", "Commander Notification");
        await delay(400);
        log("ADAPT: Monitoring both data sources. Will auto-resolve when confidence exceeds 80%.", "Continuous Monitoring Active");
        break;
      }
    }
  }
}

function delay(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}
