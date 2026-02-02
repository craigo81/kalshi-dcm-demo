// API Client for Kalshi DCM Demo
// Handles all communication with the Go backend

import axios, { AxiosError } from 'axios';

const API_BASE = '/api/v1';

// Create axios instance with defaults
const api = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add auth token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ error: string; code: string }>) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// =============================================================================
// TYPES
// =============================================================================

export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  status: string;
  is_us_resident: boolean;
  state_code: string;
  position_limit_usd: number;
  created_at: string;
}

export interface KYCRecord {
  id: string;
  user_id: string;
  status: string;
  document_type: string;
  submitted_at: string;
  reviewed_at?: string;
  rejection_reason?: string;
}

export interface Wallet {
  id: string;
  user_id: string;
  available_usd: number;
  locked_usd: number;
  pending_usd: number;
  total_deposited: number;
  total_withdrawn: number;
}

export interface Transaction {
  id: string;
  type: string;
  status: string;
  amount_usd: number;
  balance_before: number;
  balance_after: number;
  description: string;
  created_at: string;
}

export interface Market {
  ticker: string;
  event_ticker: string;
  series_ticker: string;
  title: string;
  subtitle: string;
  status: string;
  category: string;
  yes_bid: number;
  yes_ask: number;
  no_bid: number;
  no_ask: number;
  last_price: number;
  volume: number;
  volume_24h: number;
  open_interest: number;
  open_time: string;
  close_time: string;
  risk_category: string;
}

export interface Order {
  id: string;
  market_ticker: string;
  side: string;
  type: string;
  status: string;
  quantity: number;
  filled_quantity: number;
  price_cents: number;
  filled_price_cents: number;
  collateral_usd: number;
  created_at: string;
}

export interface Position {
  id: string;
  market_ticker: string;
  side: string;
  quantity: number;
  avg_price_cents: number;
  cost_basis_usd: number;
  current_value_usd: number;
  unrealized_pnl_usd: number;
}

export interface PreTradeCheck {
  passed: boolean;
  errors: string[];
  warnings: string[];
  required_margin: number;
  available_margin: number;
}

export interface PortfolioSummary {
  wallet: {
    available: number;
    locked: number;
    total: number;
  };
  positions: {
    count: number;
    total_value: number;
    unrealized_pnl: number;
  };
  limits: {
    position_limit: number;
    current_exposure: number;
    utilization: number;
  };
}

// =============================================================================
// AUTH API
// =============================================================================

export const authAPI = {
  signup: async (data: {
    email: string;
    password: string;
    first_name: string;
    last_name: string;
    state_code: string;
    date_of_birth: string;
    is_us_resident: boolean;
  }) => {
    const response = await api.post<{ data: { user: User; token: string } }>('/auth/signup', data);
    return response.data.data;
  },

  login: async (email: string, password: string) => {
    const response = await api.post<{ data: { user: User; token: string } }>('/auth/login', { email, password });
    return response.data.data;
  },

  getProfile: async () => {
    const response = await api.get<{ data: { user: User; kyc: KYCRecord | null; wallet: Wallet } }>('/profile');
    return response.data.data;
  },
};

// =============================================================================
// KYC API
// =============================================================================

export const kycAPI = {
  getStatus: async () => {
    const response = await api.get<{ data: KYCRecord | { status: string } }>('/kyc');
    return response.data.data;
  },

  submit: async (data: { document_type: string; document_number: string }) => {
    const response = await api.post<{ data: { kyc_record: KYCRecord } }>('/kyc', data);
    return response.data.data;
  },
};

// =============================================================================
// WALLET API
// =============================================================================

export const walletAPI = {
  get: async () => {
    const response = await api.get<{ data: Wallet }>('/wallet');
    return response.data.data;
  },

  deposit: async (amount_usd: number) => {
    const response = await api.post<{ data: { transaction: Transaction; wallet: Wallet } }>('/wallet/deposit', { amount_usd });
    return response.data.data;
  },

  getTransactions: async (limit = 50) => {
    const response = await api.get<{ data: Transaction[] }>(`/wallet/transactions?limit=${limit}`);
    return response.data.data;
  },
};

// =============================================================================
// MARKETS API
// =============================================================================

export const marketsAPI = {
  list: async (params?: { status?: string; series_ticker?: string; limit?: number; cursor?: string }) => {
    const searchParams = new URLSearchParams();
    if (params?.status) searchParams.set('status', params.status);
    if (params?.series_ticker) searchParams.set('series_ticker', params.series_ticker);
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.cursor) searchParams.set('cursor', params.cursor);

    const response = await api.get<{ data: Market[]; meta: { cursor: string } }>(`/markets?${searchParams}`);
    return response.data;
  },

  get: async (ticker: string) => {
    const response = await api.get<{ data: Market }>(`/markets/${ticker}`);
    return response.data.data;
  },

  getOrderbook: async (ticker: string, depth = 10) => {
    const response = await api.get(`/markets/${ticker}/orderbook?depth=${depth}`);
    return response.data.data;
  },
};

// =============================================================================
// TRADING API
// =============================================================================

export const tradingAPI = {
  preCheck: async (data: { market_ticker: string; side: string; quantity: number; price_cents: number }) => {
    const response = await api.post<{ data: PreTradeCheck }>('/orders/check', data);
    return response.data.data;
  },

  placeOrder: async (data: { market_ticker: string; side: string; type: string; quantity: number; price_cents: number }) => {
    const response = await api.post<{ data: { order: Order; wallet: Wallet } }>('/orders', data);
    return response.data.data;
  },

  getOrders: async (status?: string, limit = 50) => {
    const params = new URLSearchParams();
    if (status) params.set('status', status);
    params.set('limit', limit.toString());

    const response = await api.get<{ data: Order[] }>(`/orders?${params}`);
    return response.data.data;
  },
};

// =============================================================================
// PORTFOLIO API
// =============================================================================

export const portfolioAPI = {
  getPositions: async () => {
    const response = await api.get<{ data: { positions: Position[]; total_value: number; total_pnl: number } }>('/positions');
    return response.data.data;
  },

  getSummary: async () => {
    const response = await api.get<{ data: PortfolioSummary }>('/portfolio');
    return response.data.data;
  },
};

export default api;
