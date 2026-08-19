'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api, getStoredDriverSession } from '@/lib/api';
import { Compass, ShieldCheck, Lock, Mail, ArrowRight, Sparkles, Zap } from 'lucide-react';

export default function DriverLoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    const session = getStoredDriverSession();
    if (session && session.role === 'DRIVER') {
      router.replace('/driver');
    }
  }, [router]);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setErrorMessage('');
    try {
      await api.loginDriver(email, password);
      router.replace('/driver');
    } catch (err: any) {
      setErrorMessage(err.message || 'Login failed. Please check your partner credentials.');
      setIsLoading(false);
    }
  };

  const setDemoDriver = () => {
    setEmail('marcus.sterling@driver.urbanprime.com');
    setPassword('••••••••••••');
    setErrorMessage('');
  };

  return (
    <div className="min-h-screen bg-[#FCF9F8] flex flex-col justify-between select-none">
      {/* Top Header */}
      <header className="h-[72px] px-8 flex items-center justify-between border-b border-[#DCD9D9] bg-white">
        <div className="flex items-center gap-2.5">
          <div className="w-10 h-10 rounded-xl bg-[#008A5E] flex items-center justify-center text-white shadow-md shadow-emerald-500/20">
            <Compass className="w-5 h-5" />
          </div>
          <div>
            <span className="font-extrabold text-xl tracking-tight text-[#1F1F1F]">
              URBAN<span className="text-[#008A5E]">PRIME</span>
            </span>
            <span className="block text-[10px] tracking-widest uppercase font-semibold text-emerald-600 -mt-1">
              Partner Cockpit
            </span>
          </div>
        </div>

        <Link
          href="/driver/signup"
          className="px-4 py-2 rounded-full bg-[#008A5E] hover:bg-emerald-700 text-white text-xs font-bold transition-all flex items-center gap-1.5 shadow-md shadow-emerald-500/20"
        >
          <Zap className="w-3.5 h-3.5 fill-white" />
          <span>Apply as Driver Partner</span>
        </Link>
      </header>

      {/* Main Login Card */}
      <main className="flex-1 flex items-center justify-center p-6">
        <div className="w-full max-w-md bg-white rounded-[28px] border border-[#DCD9D9] p-8 shadow-2xl space-y-6">
          <div className="text-center space-y-1">
            <div className="w-12 h-12 rounded-2xl bg-emerald-50 text-[#008A5E] flex items-center justify-center mx-auto mb-3 shadow-xs">
              <ShieldCheck className="w-6 h-6" />
            </div>
            <h1 className="text-2xl font-black text-[#1F1F1F] tracking-tight">Partner Cockpit Login</h1>
            <p className="text-xs text-slate-500">Sign in to access real-time high-value ride dispatches</p>
          </div>

          {/* Quick Demo Autofill */}
          <div className="p-3 bg-emerald-50 border border-emerald-200 rounded-xl flex items-center justify-between text-xs">
            <div className="flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-[#008A5E]" />
              <span className="font-bold text-[#008A5E]">Demo: Marcus Sterling (5.0★)</span>
            </div>
            <button
              type="button"
              onClick={setDemoDriver}
              className="text-[11px] font-extrabold text-[#008A5E] hover:underline cursor-pointer"
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
                Partner Work Email
              </label>
              <div className="relative">
                <Mail className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="name@driver.urbanprime.com"
                  className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl pl-10 pr-4 py-3 text-xs font-semibold text-[#1F1F1F] focus:outline-none focus:border-[#008A5E] focus:ring-2 focus:ring-emerald-100 transition-all"
                />
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="block text-xs font-bold uppercase tracking-wider text-slate-600">
                  Password
                </label>
                <a href="#" className="text-[11px] font-bold text-[#008A5E] hover:underline">
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
                  className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl pl-10 pr-4 py-3 text-xs font-semibold text-[#1F1F1F] focus:outline-none focus:border-[#008A5E] focus:ring-2 focus:ring-emerald-100 transition-all"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className="w-full py-4 bg-[#008A5E] hover:bg-emerald-700 text-white font-extrabold text-xs uppercase tracking-wider rounded-xl transition-all shadow-lg shadow-emerald-500/25 active:scale-95 flex items-center justify-center gap-2 cursor-pointer"
            >
              {isLoading ? (
                <span className="inline-block animate-spin">⟳</span>
              ) : (
                <>
                  <span>Sign In to Driver Cockpit</span>
                  <ArrowRight className="w-4 h-4" />
                </>
              )}
            </button>
          </form>

          {/* Bottom Onboarding Link */}
          <div className="pt-2 border-t border-[#DCD9D9] text-center space-y-2">
            <p className="text-xs text-slate-500">Not registered as a Driver Partner yet?</p>
            <Link
              href="/driver/signup"
              className="inline-flex items-center gap-1.5 text-xs font-extrabold text-[#008A5E] hover:underline"
            >
              <span>Apply in 3 minutes & start earning</span>
              <ArrowRight className="w-3.5 h-3.5" />
            </Link>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="h-[60px] px-8 flex items-center justify-between border-t border-[#DCD9D9] bg-white text-xs text-slate-400">
        <div>Urban Prime Mobility OS • Partner Driver Ecosystem</div>
        <div>APISIX Gateway Secured</div>
      </footer>
    </div>
  );
}
