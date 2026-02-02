// Surveillance Dashboard Types

export interface Stats {
  active_users: number;
  open_positions: number;
  open_alerts: number;
  critical_alerts: number;
  total_volume_24h: number;
  last_updated: string;
  system_status: 'operational' | 'warning' | 'halted';
}

export type AlertSeverity = 'critical' | 'high' | 'medium' | 'low';
export type AlertStatus = 'open' | 'resolved' | 'dismissed';

export interface Alert {
  id: string;
  type: string;
  severity: AlertSeverity;
  status: AlertStatus;
  user_id: string;
  market_ticker: string;
  description: string;
  created_at: string;
  resolved_at?: string;
  resolved_by?: string;
  resolution_notes?: string;
}

export interface Market {
  ticker: string;
  title: string;
  is_halted: boolean;
  halt_reason?: string;
  last_price: number;
  volume_24h: number;
  halted_at?: string;
  halted_by?: string;
}

export type UserStatus = 'verified' | 'kyc_pending' | 'suspended' | 'banned';

export interface SurveillanceUser {
  id: string;
  email: string;
  status: UserStatus;
  current_exposure: number;
  position_limit: number;
  alert_count: number;
  last_activity?: string;
}

export interface ActivityLogEntry {
  id: string;
  timestamp: Date;
  message: string;
  type: 'info' | 'success' | 'warning' | 'critical';
}

// WebSocket Message Types
export type WSMessageType =
  | 'initial_state'
  | 'stats_update'
  | 'alert_resolved'
  | 'new_alert'
  | 'market_halted'
  | 'market_resumed'
  | 'global_halt'
  | 'global_resume'
  | 'user_suspended';

export interface WSMessage {
  type: WSMessageType;
  data: unknown;
}

export interface InitialStateData {
  stats: Stats;
  alerts: Alert[];
  markets: Market[];
}
