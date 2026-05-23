export type SourceType = 'official_sensor' | 'hospital_telemetry' | 'social_media' | 'dispatch_system' | 'manual_report' | 'field_unit';

export const SOURCE_RELIABILITY: Record<SourceType, number> = {
  official_sensor:    95,
  hospital_telemetry: 88,
  dispatch_system:    85,
  field_unit:         75,
  manual_report:      55,
  social_media:       30,
};

export const SOURCE_LABELS: Record<SourceType, string> = {
  official_sensor:    'Official Sensor API',
  hospital_telemetry: 'Hospital Telemetry',
  dispatch_system:    'Dispatch System',
  field_unit:         'Field Unit Report',
  manual_report:      'Manual Report',
  social_media:       'Social Media Feed',
};

export interface DataPoint {
  sourceType: SourceType;
  claim: string;
  value: number | string | boolean; // normalized for comparison
  rawValue: string;                  // human-readable label
  timestamp: string;
}

export interface Contradiction {
  id: string;
  subject: string;        // e.g. "Metro General Hospital Load"
  pointA: DataPoint;
  pointB: DataPoint;
  confidenceScore: number; // 0-100: confidence in the weighted truth
  recommendedAction: string;
  status: 'unresolved' | 'verified' | 'dismissed';
  createdAt: string;
}
