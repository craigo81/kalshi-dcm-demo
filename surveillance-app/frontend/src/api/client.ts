import type { Stats, Alert, Market, SurveillanceUser } from '../types';

const API_BASE = '/api';

class APIError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message);
    this.name = 'APIError';
  }
}

async function request<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
    ...options,
  });

  if (!response.ok) {
    const error = await response.text();
    throw new APIError(response.status, error || response.statusText);
  }

  return response.json();
}

// Stats
export async function fetchStats(): Promise<Stats> {
  return request<Stats>('/stats');
}

// Alerts
export async function fetchAlerts(status?: string, severity?: string): Promise<Alert[]> {
  const params = new URLSearchParams();
  if (status) params.append('status', status);
  if (severity) params.append('severity', severity);
  const query = params.toString() ? `?${params.toString()}` : '';
  return request<Alert[]>(`/alerts${query}`);
}

export async function resolveAlert(
  id: string,
  resolvedBy: string,
  notes: string
): Promise<void> {
  await request(`/alerts/${id}/resolve`, {
    method: 'POST',
    body: JSON.stringify({ resolved_by: resolvedBy, notes }),
  });
}

// Markets
export async function fetchMarkets(): Promise<Market[]> {
  return request<Market[]>('/markets');
}

export async function haltMarket(
  ticker: string,
  reason: string,
  initiatedBy: string
): Promise<void> {
  await request(`/markets/${ticker}/halt`, {
    method: 'POST',
    body: JSON.stringify({ reason, initiated_by: initiatedBy }),
  });
}

export async function resumeMarket(ticker: string): Promise<void> {
  await request(`/markets/${ticker}/resume`, {
    method: 'POST',
  });
}

// Users
export async function fetchUsers(): Promise<SurveillanceUser[]> {
  return request<SurveillanceUser[]>('/users');
}

export async function suspendUser(id: string): Promise<void> {
  await request(`/users/${id}/suspend`, {
    method: 'POST',
  });
}

// Global Controls
export async function globalHalt(reason: string, initiatedBy: string): Promise<void> {
  await request('/halt', {
    method: 'POST',
    body: JSON.stringify({ reason, initiated_by: initiatedBy }),
  });
}

export async function globalResume(): Promise<void> {
  await request('/resume', {
    method: 'POST',
  });
}

// Health
export async function fetchHealth(): Promise<{ status: string }> {
  return request<{ status: string }>('/health');
}
