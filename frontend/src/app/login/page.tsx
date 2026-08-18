'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { Compass, Car, ShieldCheck, ArrowRight, CheckCircle2, Lock, Mail } from 'lucide-react';

export default function LoginPage() {
  const router = useRouter();
  const [role, setRole] = useState<'RIDER' | 'DRIVER'>('RIDER');
  const [email, setEmail] = useState('alexander.vance@urbanprime.com');
  const [password, setPassword] = useState('••••••••••••');
  const [isLoading, setIsLoading] = useState(false);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    try {
      await api.login(email, role);
      if (role === 'RIDER') {
        router.push('/rider');
      } else {
        router.push('/driver');
      }
    } catch (err) {
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  const setDemoUser = (selectedRole: 'RIDER' | 'DRIVER') => {
    setRole(selectedRole);
    if (selectedRole === 'RIDER') {
      setEmail('alexander.vance@urbanprime.com');
    } else {
      setEmail('marcus.sterling@driver.urbanprime.com');
    }
  };

  return (
    <div className="min-h-screen bg-[#FCF9F8] flex flex-col justify-between select-none">
      {/* Top Simple Header */}
      <header className="h-[72px] px-8 flex items-center justify-between border-b border-[#DCD9D9] bg-white">
        <div className="flex items-center gap-2.5">
          <div className="w-10 h-10 rounded-xl bg-[#276EF1] flex items-center justify-center text-white shadow-md shadow-blue-500/20">
            <Compass className="w-5 h-5" />
          </div>
          <div>
            <span className="font-extrabold text-xl tracking-tight text-[#1F1F1F]">URBAN<span className="text-[#276EF1]">PRIME</span></span>
            <span className="block text-[10px] tracking-widest uppercase font-semibold text-slate-400 -mt-1">Ecosystem Access</span>
          </div>
        </div>

        <div className="text-xs font-semibold text-slate-500">
          Enterprise Distributed System
        </div>
      </header>

      {/* Main Login Card */}
      <main className="flex-1 flex items-center justify-center p-6">
        <div className="w-full max-w-md bg-white rounded-[24px] border border-[#DCD9D9] p-8 shadow-xl">
          <div className="mb-6 text-center">
            <h1 className="text-2xl font-bold text-[#1F1F1F] tracking-tight">Welcome to Urban Prime</h1>
            <p className="text-sm text-slate-500 mt-1">Select your portal to access real-time dispatch</p>
          </div>

          {/* Role Toggle */}
          <div className="grid grid-cols-2 gap-2 p-1 bg-[#FCF9F8] rounded-2xl border border-[#DCD9D9] mb-6">
            <button
              type="button"
              onClick={() => setDemoUser('RIDER')}
              className={`flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-bold transition-all ${
                role === 'RIDER'
                  ? 'bg-white text-[#276EF1] shadow-sm border border-blue-100'
                  : 'text-slate-500 hover:text-slate-800'
              }`}
            >
              <Car className="w-4 h-4" />
              Rider Portal
            </button>
            <button
              type="button"
              onClick={() => setDemoUser('DRIVER')}
              className={`flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-bold transition-all ${
                role === 'DRIVER'
                  ? 'bg-white text-[#276EF1] shadow-sm border border-blue-100'
                  : 'text-slate-500 hover:text-slate-800'
              }`}
            >
              <ShieldCheck className="w-4 h-4" />
              Driver Partner
            </button>
          </div>

          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                Work Email
              </label>
              <div className="relative">
                <Mail className="w-4 h-4 text-slate-400 absolute left-3.5 top-3.5" />
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl pl-10 pr-4 py-2.5 text-sm font-medium text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] focus:ring-2 focus:ring-blue-100 transition-all"
                  placeholder="name@urbanprime.com"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                Password
              </label>
              <div className="relative">
                <Lock className="w-4 h-4 text-slate-400 absolute left-3.5 top-3.5" />
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl pl-10 pr-4 py-2.5 text-sm font-medium text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] focus:ring-2 focus:ring-blue-100 transition-all"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className="w-full mt-2 py-3.5 bg-[#276EF1] hover:bg-[#1A54C9] text-white font-bold text-sm rounded-xl transition-all shadow-md shadow-blue-500/25 active:scale-95 flex items-center justify-center gap-2"
            >
              {isLoading ? (
                <span>Authenticating with Gateway...</span>
              ) : (
                <>
                  <span>Sign In as {role === 'RIDER' ? 'Rider' : 'Driver Partner'}</span>
                  <ArrowRight className="w-4 h-4" />
                </>
              )}
            </button>
          </form>

          {/* Quick Demo Pre-fill */}
          <div className="mt-6 pt-6 border-t border-slate-100">
            <p className="text-[11px] uppercase font-bold tracking-wider text-slate-400 text-center mb-3">
              1-Click Instant Demo Credentials
            </p>
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => {
                  setDemoUser('RIDER');
                  api.login('alexander.vance@urbanprime.com', 'RIDER').then(() => router.push('/rider'));
                }}
                className="p-2.5 rounded-xl border border-blue-100 bg-blue-50/50 hover:bg-blue-50 text-left transition-colors group"
              >
                <div className="flex items-center gap-1.5 text-xs font-bold text-[#276EF1]">
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  Rider Demo
                </div>
                <div className="text-[11px] text-slate-500 truncate">Alexander Vance</div>
              </button>

              <button
                type="button"
                onClick={() => {
                  setDemoUser('DRIVER');
                  api.login('marcus.sterling@driver.urbanprime.com', 'DRIVER').then(() => router.push('/driver'));
                }}
                className="p-2.5 rounded-xl border border-emerald-100 bg-emerald-50/50 hover:bg-emerald-50 text-left transition-colors group"
              >
                <div className="flex items-center gap-1.5 text-xs font-bold text-emerald-700">
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  Driver Demo
                </div>
                <div className="text-[11px] text-slate-500 truncate">Marcus Sterling</div>
              </button>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="py-4 text-center text-xs text-slate-400 border-t border-[#DCD9D9] bg-white">
        Urban Prime Orchestration Engine • APISIX Gateway • Kafka Real-Time Stream
      </footer>
    </div>
  );
}
