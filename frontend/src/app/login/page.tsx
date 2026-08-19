'use client';

import React from 'react';
import Link from 'next/link';
import { Compass, Car, ShieldCheck, ArrowRight, Zap, Star } from 'lucide-react';

export default function PortalGatewayPage() {
  return (
    <div className="min-h-screen bg-[#FCF9F8] flex flex-col justify-between select-none">
      {/* Top Header */}
      <header className="h-[72px] px-8 flex items-center justify-between border-b border-[#DCD9D9] bg-white">
        <Link href="/" className="flex items-center gap-2.5">
          <div className="w-10 h-10 rounded-xl bg-[#276EF1] flex items-center justify-center text-white shadow-md shadow-blue-500/20">
            <Compass className="w-5 h-5" />
          </div>
          <div>
            <span className="font-extrabold text-xl tracking-tight text-[#1F1F1F]">
              URBAN<span className="text-[#276EF1]">PRIME</span>
            </span>
            <span className="block text-[10px] tracking-widest uppercase font-semibold text-slate-400 -mt-1">
              Mobility OS
            </span>
          </div>
        </Link>

        <div className="text-xs text-slate-500 font-semibold">
          High-Availability Distributed System
        </div>
      </header>

      {/* Main 2-Card Portal Selector */}
      <main className="flex-1 flex items-center justify-center p-6">
        <div className="w-full max-w-3xl space-y-8">
          <div className="text-center space-y-2">
            <h1 className="text-3xl font-black text-[#1F1F1F] tracking-tight">
              Select Your Access Portal
            </h1>
            <p className="text-sm text-slate-500 max-w-md mx-auto">
              Rider and Driver Partner portals operate with isolated accounts and separate real-time pipelines.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* 1. Rider Portal Card */}
            <div className="bg-white rounded-[28px] border-2 border-[#DCD9D9] hover:border-[#276EF1] p-8 shadow-xl hover:shadow-2xl transition-all flex flex-col justify-between group">
              <div className="space-y-4">
                <div className="w-14 h-14 rounded-2xl bg-blue-50 text-[#276EF1] flex items-center justify-center shadow-xs group-hover:scale-110 transition-transform">
                  <Car className="w-7 h-7" />
                </div>
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <span className="px-2.5 py-0.5 rounded-full bg-blue-100 text-[#276EF1] text-[10px] font-extrabold uppercase tracking-wider">
                      Rider App
                    </span>
                  </div>
                  <h2 className="text-2xl font-black text-[#1F1F1F]">Rider Portal</h2>
                  <p className="text-xs text-slate-500 mt-1 leading-relaxed">
                    Book executive chauffeur rides, view live Mapbox driving routes, OTP security pins, and seamless payments.
                  </p>
                </div>
              </div>

              <div className="pt-6 mt-6 border-t border-slate-100 space-y-3">
                <Link
                  href="/rider/login"
                  className="w-full py-3.5 bg-[#276EF1] hover:bg-[#1A54C9] text-white font-extrabold text-xs uppercase tracking-wider rounded-xl transition-all shadow-md shadow-blue-500/25 flex items-center justify-center gap-2 cursor-pointer active:scale-95"
                >
                  <span>Sign In as Rider</span>
                  <ArrowRight className="w-4 h-4" />
                </Link>
                <div className="text-center">
                  <Link
                    href="/rider/signup"
                    className="text-xs font-bold text-[#276EF1] hover:underline"
                  >
                    New Rider? Create Account →
                  </Link>
                </div>
              </div>
            </div>

            {/* 2. Driver Partner Portal Card */}
            <div className="bg-white rounded-[28px] border-2 border-[#DCD9D9] hover:border-[#008A5E] p-8 shadow-xl hover:shadow-2xl transition-all flex flex-col justify-between group">
              <div className="space-y-4">
                <div className="w-14 h-14 rounded-2xl bg-emerald-50 text-[#008A5E] flex items-center justify-center shadow-xs group-hover:scale-110 transition-transform">
                  <ShieldCheck className="w-7 h-7" />
                </div>
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <span className="px-2.5 py-0.5 rounded-full bg-emerald-100 text-[#008A5E] text-[10px] font-extrabold uppercase tracking-wider">
                      Partner Cockpit
                    </span>
                  </div>
                  <h2 className="text-2xl font-black text-[#1F1F1F]">Driver Cockpit</h2>
                  <p className="text-xs text-slate-500 mt-1 leading-relaxed">
                    High-urgency 15-second dispatches, real-time GPS telemetry, OTP ride start, and shift earnings tracking.
                  </p>
                </div>
              </div>

              <div className="pt-6 mt-6 border-t border-slate-100 space-y-3">
                <Link
                  href="/driver/login"
                  className="w-full py-3.5 bg-[#008A5E] hover:bg-emerald-700 text-white font-extrabold text-xs uppercase tracking-wider rounded-xl transition-all shadow-md shadow-emerald-500/25 flex items-center justify-center gap-2 cursor-pointer active:scale-95"
                >
                  <span>Sign In as Driver Partner</span>
                  <ArrowRight className="w-4 h-4" />
                </Link>
                <div className="text-center">
                  <Link
                    href="/driver/signup"
                    className="text-xs font-bold text-[#008A5E] hover:underline"
                  >
                    Apply in 3 Minutes & Onboard Fleet →
                  </Link>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="h-[60px] px-8 flex items-center justify-between border-t border-[#DCD9D9] bg-white text-xs text-slate-400">
        <div>Urban Prime Mobility OS • Isolated Rider & Driver Architectures</div>
        <div>APISIX Gateway & Centrifugo Mesh</div>
      </footer>
    </div>
  );
}
