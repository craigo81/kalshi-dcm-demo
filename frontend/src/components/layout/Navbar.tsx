// Navigation Bar Component

import React from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { Zap, LayoutDashboard, TrendingUp, Briefcase, LogOut, User, Shield } from 'lucide-react';
import { useAuth } from '../../context/AuthContext';

export function Navbar() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, isAuthenticated, isVerified, logout } = useAuth();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const navItems = [
    { path: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { path: '/markets', label: 'Markets', icon: TrendingUp },
    { path: '/portfolio', label: 'Portfolio', icon: Briefcase },
  ];

  if (!isAuthenticated) {
    return (
      <nav className="bg-slate-900/80 backdrop-blur-md border-b border-slate-700 sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4">
          <div className="flex items-center justify-between h-16">
            <Link to="/" className="flex items-center gap-3">
              <div className="bg-gradient-to-br from-primary-500 to-purple-600 p-2 rounded-xl">
                <Zap className="w-5 h-5 text-white" />
              </div>
              <div>
                <span className="text-lg font-bold text-white">Kalshi DCM</span>
                <span className="text-xs text-slate-400 block">CFTC Compliant Demo</span>
              </div>
            </Link>

            <div className="flex items-center gap-3">
              <Link to="/login" className="btn btn-secondary">
                Sign In
              </Link>
              <Link to="/signup" className="btn btn-primary">
                Get Started
              </Link>
            </div>
          </div>
        </div>
      </nav>
    );
  }

  return (
    <nav className="bg-slate-900/80 backdrop-blur-md border-b border-slate-700 sticky top-0 z-40">
      <div className="max-w-7xl mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link to="/dashboard" className="flex items-center gap-3">
            <div className="bg-gradient-to-br from-primary-500 to-purple-600 p-2 rounded-xl">
              <Zap className="w-5 h-5 text-white" />
            </div>
            <div>
              <span className="text-lg font-bold text-white">Kalshi DCM</span>
              <span className="text-xs text-slate-400 block">CFTC Compliant Demo</span>
            </div>
          </Link>

          {/* Navigation Links */}
          <div className="hidden md:flex items-center gap-1">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isActive = location.pathname === item.path;

              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-colors ${
                    isActive
                      ? 'bg-primary-600 text-white'
                      : 'text-slate-400 hover:text-white hover:bg-slate-800'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                  {item.label}
                </Link>
              );
            })}
          </div>

          {/* User Menu */}
          <div className="flex items-center gap-4">
            {/* KYC Status Badge */}
            {!isVerified && (
              <Link
                to="/kyc"
                className="flex items-center gap-2 px-3 py-1.5 bg-amber-500/20 border border-amber-500/30 rounded-lg text-amber-400 text-sm hover:bg-amber-500/30 transition-colors"
              >
                <Shield className="w-4 h-4" />
                Complete KYC
              </Link>
            )}

            {isVerified && (
              <div className="flex items-center gap-2 px-3 py-1.5 bg-emerald-500/20 border border-emerald-500/30 rounded-lg text-emerald-400 text-sm">
                <Shield className="w-4 h-4" />
                Verified
              </div>
            )}

            {/* User dropdown */}
            <div className="flex items-center gap-3">
              <div className="text-right">
                <p className="text-sm font-medium text-white">
                  {user?.first_name} {user?.last_name}
                </p>
                <p className="text-xs text-slate-400">{user?.email}</p>
              </div>
              <div className="w-10 h-10 bg-slate-700 rounded-full flex items-center justify-center">
                <User className="w-5 h-5 text-slate-400" />
              </div>
            </div>

            <button
              onClick={handleLogout}
              className="p-2 hover:bg-slate-800 rounded-lg transition-colors"
              title="Logout"
            >
              <LogOut className="w-5 h-5 text-slate-400" />
            </button>
          </div>
        </div>
      </div>
    </nav>
  );
}
