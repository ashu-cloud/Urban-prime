import Link from 'next/link';
import { Compass, Car, ShieldCheck, ArrowRight, Zap, Radio, Database, Cpu } from 'lucide-react';

export default function Home() {
  return (
    <div className="min-h-screen bg-[#FCF9F8] text-[#1F1F1F] flex flex-col justify-between select-none">
      {/* Top Brand Header */}
      <header className="h-[72px] px-8 bg-white border-b border-[#DCD9D9] flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <div className="w-10 h-10 rounded-xl bg-[#276EF1] text-white flex items-center justify-center shadow-md shadow-blue-500/20">
            <Compass className="w-5 h-5" />
          </div>
          <div>
            <span className="font-extrabold text-xl tracking-tight text-[#1F1F1F]">URBAN<span className="text-[#276EF1]">PRIME</span></span>
            <span className="block text-[10px] tracking-widest uppercase font-bold text-slate-400 -mt-1">Real-Time Mobility</span>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <Link
            href="/login"
            className="px-5 py-2.5 bg-[#276EF1] hover:bg-[#1A54C9] text-white text-xs font-bold rounded-xl transition-all shadow-sm active:scale-95"
          >
            Sign In to Portal
          </Link>
        </div>
      </header>

      {/* Main Hero Section */}
      <main className="max-w-6xl mx-auto px-6 py-12 flex-1 flex flex-col justify-center items-center text-center">
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-[#E7F0FF] text-[#276EF1] text-xs font-extrabold uppercase tracking-wider mb-6 border border-blue-200">
          <Zap className="w-3.5 h-3.5" /> High-Scale Distributed Cab Booking Platform
        </div>

        <h1 className="text-4xl sm:text-5xl lg:text-6xl font-black text-[#1F1F1F] tracking-tight max-w-3xl leading-tight">
          Next-Generation <span className="text-[#276EF1]">Chauffeur & Fleet</span> Dispatch System
        </h1>

        <p className="text-base text-slate-600 max-w-2xl mt-4 leading-relaxed">
          Powered by Go microservices, Apache Kafka event streaming, Redis geospatial matchmaking, APISIX gateway, and Stripe distributed Sagas.
        </p>

        {/* 2-App Portal Selector Cards */}
        <div className="grid md:grid-cols-2 gap-6 w-full max-w-3xl mt-10 text-left">
          {/* Rider App Card */}
          <Link
            href="/rider"
            className="p-8 rounded-[24px] bg-white border border-[#DCD9D9] hover:border-[#276EF1] hover:shadow-xl transition-all group flex flex-col justify-between"
          >
            <div>
              <div className="w-12 h-12 rounded-2xl bg-[#E7F0FF] text-[#276EF1] flex items-center justify-center mb-6 group-hover:scale-110 transition-transform">
                <Car className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-[#1F1F1F] mb-1">Rider Desktop App</h3>
              <p className="text-xs text-slate-500 leading-relaxed mb-6">
                Interactive Mapbox booking interface, dynamic 3D fleet tier selector, and live driver tracking with 3s GPS heading indicator.
              </p>
            </div>

            <div className="flex items-center gap-2 text-xs font-bold text-[#276EF1]">
              <span>Launch Rider Experience</span>
              <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
            </div>
          </Link>

          {/* Driver Partner App Card */}
          <Link
            href="/driver"
            className="p-8 rounded-[24px] bg-white border border-[#DCD9D9] hover:border-[#008A5E] hover:shadow-xl transition-all group flex flex-col justify-between"
          >
            <div>
              <div className="w-12 h-12 rounded-2xl bg-emerald-50 text-[#008A5E] flex items-center justify-center mb-6 group-hover:scale-110 transition-transform">
                <ShieldCheck className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-[#1F1F1F] mb-1">Driver Partner Cockpit</h3>
              <p className="text-xs text-slate-500 leading-relaxed mb-6">
                Real-time metrics overview, active/offline toggle, GPS beacon broadcaster, and 15-second radial dispatch countdown modal.
              </p>
            </div>

            <div className="flex items-center gap-2 text-xs font-bold text-emerald-700">
              <span>Launch Partner Cockpit</span>
              <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
            </div>
          </Link>
        </div>

        {/* Distributed Backend Architecture Bar */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 w-full max-w-3xl mt-8">
          <div className="p-3.5 bg-white rounded-xl border border-[#DCD9D9] text-xs flex items-center gap-2.5">
            <Radio className="w-4 h-4 text-[#276EF1]" />
            <span className="font-bold text-slate-700">Kafka Streaming</span>
          </div>
          <div className="p-3.5 bg-white rounded-xl border border-[#DCD9D9] text-xs flex items-center gap-2.5">
            <Database className="w-4 h-4 text-purple-600" />
            <span className="font-bold text-slate-700">Postgres + Redis Geo</span>
          </div>
          <div className="p-3.5 bg-white rounded-xl border border-[#DCD9D9] text-xs flex items-center gap-2.5">
            <Cpu className="w-4 h-4 text-amber-600" />
            <span className="font-bold text-slate-700">Trip Saga Orchestrator</span>
          </div>
          <div className="p-3.5 bg-white rounded-xl border border-[#DCD9D9] text-xs flex items-center gap-2.5">
            <Zap className="w-4 h-4 text-emerald-600" />
            <span className="font-bold text-slate-700">APISIX API Gateway</span>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="h-14 px-8 bg-white border-t border-[#DCD9D9] flex items-center justify-between text-xs text-slate-400">
        <span>Urban Prime Ecosystem • Desktop First Edition</span>
        <span>Centrifugo WebSocket v5</span>
      </footer>
    </div>
  );
}
