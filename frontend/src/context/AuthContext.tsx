// Authentication Context
// Manages user session state and provides auth methods

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { authAPI, User, Wallet, KYCRecord } from '../api/client';

interface AuthState {
  user: User | null;
  wallet: Wallet | null;
  kyc: KYCRecord | null;
  token: string | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  isVerified: boolean;
}

interface AuthContextType extends AuthState {
  login: (email: string, password: string) => Promise<void>;
  signup: (data: SignupData) => Promise<void>;
  logout: () => void;
  refreshProfile: () => Promise<void>;
}

interface SignupData {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  state_code: string;
  date_of_birth: string;
  is_us_resident: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    user: null,
    wallet: null,
    kyc: null,
    token: localStorage.getItem('token'),
    isLoading: true,
    isAuthenticated: false,
    isVerified: false,
  });

  const refreshProfile = useCallback(async () => {
    const token = localStorage.getItem('token');
    if (!token) {
      setState(prev => ({ ...prev, isLoading: false }));
      return;
    }

    try {
      const profile = await authAPI.getProfile();
      setState({
        user: profile.user,
        wallet: profile.wallet,
        kyc: profile.kyc,
        token,
        isLoading: false,
        isAuthenticated: true,
        isVerified: profile.user.status === 'verified',
      });
    } catch {
      localStorage.removeItem('token');
      setState({
        user: null,
        wallet: null,
        kyc: null,
        token: null,
        isLoading: false,
        isAuthenticated: false,
        isVerified: false,
      });
    }
  }, []);

  useEffect(() => {
    refreshProfile();
  }, [refreshProfile]);

  const login = async (email: string, password: string) => {
    const { user, token } = await authAPI.login(email, password);
    localStorage.setItem('token', token);
    setState(prev => ({
      ...prev,
      user,
      token,
      isAuthenticated: true,
      isVerified: user.status === 'verified',
    }));
    await refreshProfile();
  };

  const signup = async (data: SignupData) => {
    const { user, token } = await authAPI.signup(data);
    localStorage.setItem('token', token);
    setState(prev => ({
      ...prev,
      user,
      token,
      isAuthenticated: true,
      isVerified: false,
    }));
    await refreshProfile();
  };

  const logout = () => {
    localStorage.removeItem('token');
    setState({
      user: null,
      wallet: null,
      kyc: null,
      token: null,
      isLoading: false,
      isAuthenticated: false,
      isVerified: false,
    });
  };

  return (
    <AuthContext.Provider value={{ ...state, login, signup, logout, refreshProfile }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
