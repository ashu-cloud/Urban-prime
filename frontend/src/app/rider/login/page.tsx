'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api, getStoredRiderSession } from '@/lib/api';
import { Compass, Car, Lock, Mail, ArrowRight, Sparkles, Navigation } from 'lucide-react';

export default function RiderLoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [pendingRide, setPendingRide] = useState<any>(null);

  useEffect(() => {
    const session = getStoredRiderSession();
    if (session && session.role === 'RIDER') {
      router.replace('/rider');
      return;
    }

    try {
      const rawPending = localStorage.getItem('urban_pending_ride');
      if (rawPending) {
        setPendingRide(JSON.parse(rawPending));
      }
    } catch {
      // Ignore
    }
  }, [router]);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setErrorMessage('');
    try {
      await api.loginRider(email, password);
      router.replace('/rider');
    } catch (err: any) {
      setErrorMessage(err.message || 'Login failed. Please check your credentials.');
      setIsLoading(false);
    }
  };

  const setDemoRider = () => {
    setEmail('alexander.vance@urbanprime.com');
    setPassword('••••••••••••');
    setErrorMessage('');
  };

  return (
    <div className="min-h-screen bg-[#FCF9F8] flex flex-col justify-between select-none">
      {/* Top Header */}
      <header className="h-[72px] px-8 flex items-center justify-between border-b border-[#DCD9D9] bg-white">
        <div className="flex items-center gap-2.5">
          <div className="w-10 h-10 rounded-xl bg-[#276EF1] flex items-center justify-center text-white shadow-md shadow-blue-500/20">
            <Compass className="w-5 h-5" />
          </div>
          <div>
            <span className="font-extrabold text-xl tracking-tight text-[#1F1F1F]">
              URBAN<span className="text-[#276EF1]">PRIME</span>
            </span>
            <span className="block text-[10px] tracking-widest uppercase font-semibold text-slate-400 -mt-1">
              Rider Portal
            </span>
          </div>
        </div>

        <Link
          href="/rider/signup"
          className="px-4 py-2 rounded-full bg-[#276EF1] hover:bg-[#1A54C9] text-white text-xs font-bold transition-all flex items-center gap-1.5 shadow-md shadow-blue-500/20"
        >
          <span>Create Rider Account</span>
          <ArrowRight className="w-3.5 h-3.5" />
        </Link>
      </header>

      {/* Main Login Card */}
      <main className="flex-1 flex items-center justify-center p-6">
        <div className="w-full max-w-md bg-white rounded-[28px] border border-[#DCD9D9] p-8 shadow-2xl space-y-6">
          <div className="text-center space-y-1">
            <div className="w-12 h-12 rounded-2xl bg-blue-50 text-[#276EF1] flex items-center justify-center mx-auto mb-3 shadow-xs">
              <Car className="w-6 h-6" />
            </div>
            <h1 className="text-2xl font-black text-[#1F1F1F] tracking-tight">Rider Sign In</h1>
            <p className="text-xs text-slate-500">
              {pendingRide
                ? 'Sign in to complete your requested ride dispatch'
                : 'Sign in to book executive on-demand chauffeur rides'}
            </p>
          </div>

          {/* Pending Ride Preview Alert */}
          {pendingRide && (
            <div className="p-3 bg-blue-50 border border-blue-200 rounded-2xl flex items-center gap-3">
              <div className="w-8 h-8 rounded-xl bg-[#276EF1] text-white flex items-center justify-center shrink-0">
                <Navigation className="w-4 h-4" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-[11px] font-extrabold text-[#276EF1] uppercase tracking-wide">
                  Preserved Ride Dispatch
                </p>
                <p className="text-xs font-bold text-slate-800 truncate">
                  To: {pendingRide.dropoffAddress}
                </p>
                <p className="text-[10px] text-slate-500 font-semibold">
                  {pendingRide.selectedTier} • {pendingRide.drivingDistanceKm} km ({pendingRide.drivingDurationText})
                </p>
              </div>
            </div>
          )}

          {/* Quick Demo Autofill */}
          <div className="p-3 bg-blue-50 border border-blue-200 rounded-xl flex items-center justify-between text-xs">
            <div className="flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-[#276EF1]" />
              <span className="font-bold text-[#276EF1]">Demo: Alexander Vance</span>
            </div>
            <button
              type="button"
              onClick={setDemoRider}
              className="text-[11px] font-extrabold text-[#276EF1] hover:underline cursor-pointer"
            >
              Autofill
            </button>
          </div>

          {errorMessage && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-xs font-bold text-red-700">
              {errorMessage}
            </div>
          )}

          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                Rider Email
              </label>
              <div className="relative">
                <Mail className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="rider@urbanprime.com"
                  className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl pl-10 pr-4 py-3 text-xs font-semibold text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] focus:ring-2 focus:ring-blue-100 transition-all"
                />
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="block text-xs font-bold uppercase tracking-wider text-slate-600">
                  Password
                </label>
                <a href="#" className="text-[11px] font-bold text-[#276EF1] hover:underline">
                  Forgot?
                </a>
              </div>
              <div className="relative">
                <Lock className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••••••"
                  className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl pl-10 pr-4 py-3 text-xs font-semibold text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] focus:ring-2 focus:ring-blue-100 transition-all"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className="w-full py-4 bg-[#276EF1] hover:bg-[#1A54C9] text-white font-extrabold text-xs uppercase tracking-wider rounded-xl transition-all shadow-lg shadow-blue-500/25 active:scale-95 flex items-center justify-center gap-2 cursor-pointer"
            >
              {isLoading ? (
                <span className="inline-block animate-spin">⟳</span>
              ) : (
                <>
                  <span>{pendingRide ? 'Sign In & Confirm Ride' : 'Sign In as Rider'}</span>
                  <ArrowRight className="w-4 h-4" />
                </>
              )}
            </button>
          </form>

          {/* Bottom Onboarding Link */}
          <div className="pt-2 border-t border-[#DCD9D9] text-center space-y-2">
            <p className="text-xs text-slate-500">New to Urban Prime?</p>
            <Link
              href="/rider/signup"
              className="inline-flex items-center gap-1.5 text-xs font-extrabold text-[#276EF1] hover:underline"
            >
              <span>Create your rider account</span>
              <ArrowRight className="w-3.5 h-3.5" />
            </Link>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="h-[60px] px-8 flex items-center justify-between border-t border-[#DCD9D9] bg-white text-xs text-slate-400">
        <div>Urban Prime Mobility OS • Rider Experience</div>
        <div>APISIX Protected Gateway</div>
      </footer>
    </div>
  );
}
