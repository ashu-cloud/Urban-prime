'use client';

import React, { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import { Building2, ArrowRight, Car, Zap, ShieldCheck, Menu, Database, Radio, Cpu, Network, Globe, MapPin, Search, CreditCard } from 'lucide-react';
import MapboxView, { MarkerLocation } from '@/components/map/MapboxView';

// ---------------------------------------------------------------------------
// 1. Real-World NYC Road Corridors
// ---------------------------------------------------------------------------
interface RoadCorridor {
  id: string;
  points: { lat: number; lng: number }[];
}

const NYC_ROADS: RoadCorridor[] = [
  // 1. FDR Drive Northbound (East River shoreline)
  {
    id: 'fdr_nb',
    points: [
      { lat: 40.7100, lng: -73.9780 },
      { lat: 40.7200, lng: -73.9720 },
      { lat: 40.7350, lng: -73.9730 },
      { lat: 40.7500, lng: -73.9670 },
      { lat: 40.7650, lng: -73.9530 },
      { lat: 40.7850, lng: -73.9420 },
    ],
  },
  // 2. FDR Drive Southbound
  {
    id: 'fdr_sb',
    points: [
      { lat: 40.7850, lng: -73.9420 },
      { lat: 40.7650, lng: -73.9530 },
      { lat: 40.7500, lng: -73.9670 },
      { lat: 40.7350, lng: -73.9730 },
      { lat: 40.7200, lng: -73.9720 },
      { lat: 40.7100, lng: -73.9780 },
    ],
  },
  // 3. West Side Highway Northbound
  {
    id: 'wsh_nb',
    points: [
      { lat: 40.7150, lng: -74.0150 },
      { lat: 40.7350, lng: -74.0090 },
      { lat: 40.7550, lng: -74.0030 },
      { lat: 40.7720, lng: -73.9930 },
      { lat: 40.7900, lng: -73.9830 },
    ],
  },
  // 4. West Side Highway Southbound
  {
    id: 'wsh_sb',
    points: [
      { lat: 40.7900, lng: -73.9830 },
      { lat: 40.7720, lng: -73.9930 },
      { lat: 40.7550, lng: -74.0030 },
      { lat: 40.7350, lng: -74.0100 },
      { lat: 40.7150, lng: -74.0150 },
    ],
  },
  // 5. Broadway / 7th Ave Southbound
  {
    id: 'broadway_sb',
    points: [
      { lat: 40.7850, lng: -73.9810 },
      { lat: 40.7680, lng: -73.9820 },
      { lat: 40.7560, lng: -73.9870 },
      { lat: 40.7420, lng: -73.9900 },
      { lat: 40.7280, lng: -73.9970 },
      { lat: 40.7150, lng: -74.0050 },
    ],
  },
  // 6. 5th Avenue Southbound
  {
    id: 'fifth_ave_sb',
    points: [
      { lat: 40.7850, lng: -73.9580 },
      { lat: 40.7680, lng: -73.9700 },
      { lat: 40.7540, lng: -73.9800 },
      { lat: 40.7400, lng: -73.9900 },
      { lat: 40.7310, lng: -73.9960 },
    ],
  },
  // 7. Park Avenue / Madison Ave Northbound
  {
    id: 'park_ave_nb',
    points: [
      { lat: 40.7350, lng: -73.9890 },
      { lat: 40.7480, lng: -73.9790 },
      { lat: 40.7600, lng: -73.9710 },
      { lat: 40.7750, lng: -73.9600 },
      { lat: 40.7900, lng: -73.9480 },
    ],
  },
  // 8. 42nd Street Crosstown (West to East)
  {
    id: 'crosstown_42_eb',
    points: [
      { lat: 40.7600, lng: -74.0010 },
      { lat: 40.7570, lng: -73.9900 },
      { lat: 40.7530, lng: -73.9800 },
      { lat: 40.7490, lng: -73.9690 },
    ],
  },
  // 9. 34th Street Crosstown (East to West)
  {
    id: 'crosstown_34_wb',
    points: [
      { lat: 40.7440, lng: -73.9720 },
      { lat: 40.7480, lng: -73.9850 },
      { lat: 40.7520, lng: -73.9980 },
      { lat: 40.7540, lng: -74.0050 },
    ],
  },
  // 10. Queensboro Bridge (Eastbound to LIC)
  {
    id: 'queensboro_eb',
    points: [
      { lat: 40.7600, lng: -73.9640 },
      { lat: 40.7570, lng: -73.9540 },
      { lat: 40.7530, lng: -73.9450 },
      { lat: 40.7500, lng: -73.9350 },
    ],
  },
  // 11. Queensboro Bridge (Westbound to Manhattan)
  {
    id: 'queensboro_wb',
    points: [
      { lat: 40.7500, lng: -73.9350 },
      { lat: 40.7530, lng: -73.9450 },
      { lat: 40.7570, lng: -73.9540 },
      { lat: 40.7600, lng: -73.9640 },
    ],
  },
  // 12. Williamsburg Bridge (Eastbound to Brooklyn)
  {
    id: 'williamsburg_eb',
    points: [
      { lat: 40.7190, lng: -73.9870 },
      { lat: 40.7140, lng: -73.9730 },
      { lat: 40.7100, lng: -73.9600 },
    ],
  },
  // 13. Williamsburg Bridge (Westbound to Manhattan)
  {
    id: 'williamsburg_wb',
    points: [
      { lat: 40.7100, lng: -73.9600 },
      { lat: 40.7140, lng: -73.9730 },
      { lat: 40.7190, lng: -73.9870 },
    ],
  },
  // 14. Hoboken / Weehawken River Road (Northbound)
  {
    id: 'hoboken_nb',
    points: [
      { lat: 40.7340, lng: -74.0300 },
      { lat: 40.7450, lng: -74.0290 },
      { lat: 40.7550, lng: -74.0260 },
      { lat: 40.7680, lng: -74.0190 },
      { lat: 40.7750, lng: -74.0130 },
    ],
  },
  // 15. Hoboken / Weehawken River Road (Southbound)
  {
    id: 'hoboken_sb',
    points: [
      { lat: 40.7750, lng: -74.0130 },
      { lat: 40.7680, lng: -74.0190 },
      { lat: 40.7550, lng: -74.0260 },
      { lat: 40.7450, lng: -74.0290 },
      { lat: 40.7340, lng: -74.0300 },
    ],
  },
  // 16. Long Island City & Astoria (Northbound)
  {
    id: 'lic_nb',
    points: [
      { lat: 40.7400, lng: -73.9550 },
      { lat: 40.7500, lng: -73.9480 },
      { lat: 40.7600, lng: -73.9400 },
      { lat: 40.7720, lng: -73.9300 },
    ],
  },
  // 17. Long Island City & Astoria (Southbound)
  {
    id: 'lic_sb',
    points: [
      { lat: 40.7720, lng: -73.9300 },
      { lat: 40.7600, lng: -73.9400 },
      { lat: 40.7500, lng: -73.9480 },
      { lat: 40.7400, lng: -73.9550 },
    ],
  },
  // 18. Brooklyn / Bedford Ave (Northbound)
  {
    id: 'brooklyn_nb',
    points: [
      { lat: 40.6980, lng: -73.9850 },
      { lat: 40.7080, lng: -73.9880 },
      { lat: 40.7180, lng: -73.9650 },
      { lat: 40.7280, lng: -73.9530 },
    ],
  },
];

// ---------------------------------------------------------------------------
// 2. High-Resolution Road Trajectories
// ---------------------------------------------------------------------------
const STEPS = 800;

function precomputeSmoothPath(points: { lat: number; lng: number }[]): { lat: number; lng: number; heading: number }[] {
  let totalLength = 0;
  const segLengths: number[] = [];
  for (let i = 0; i < points.length - 1; i++) {
    const dLat = points[i + 1].lat - points[i].lat;
    const dLng = points[i + 1].lng - points[i].lng;
    const len = Math.sqrt(dLat * dLat + dLng * dLng);
    segLengths.push(len);
    totalLength += len;
  }

  const result: { lat: number; lng: number; heading: number }[] = [];
  for (let s = 0; s < STEPS; s++) {
    const targetDist = (s / STEPS) * totalLength;
    let accumulated = 0;

    for (let i = 0; i < segLengths.length; i++) {
      if (accumulated + segLengths[i] >= targetDist || i === segLengths.length - 1) {
        const segFraction = segLengths[i] > 0 ? (targetDist - accumulated) / segLengths[i] : 0;
        const p1 = points[i];
        const p2 = points[i + 1];

        const lat = p1.lat + (p2.lat - p1.lat) * segFraction;
        const lng = p1.lng + (p2.lng - p1.lng) * segFraction;

        const dLat = p2.lat - p1.lat;
        const dLng = (p2.lng - p1.lng) * Math.cos((p1.lat * Math.PI) / 180);
        const heading = (Math.atan2(dLng, dLat) * 180) / Math.PI;

        result.push({
          lat,
          lng,
          heading: (heading + 360) % 360,
        });
        break;
      }
      accumulated += segLengths[i];
    }
  }
  return result;
}

const PRECOMPUTED_PATHS = NYC_ROADS.map((road) => precomputeSmoothPath(road.points));

// 24 Active drivers distributed across all corridors
const DRIVERS_DATA = Array.from({ length: 24 }, (_, i) => ({
  id: `driver-${i + 1}`,
  roadIndex: i % PRECOMPUTED_PATHS.length,
  initialProgress: (Math.floor(i / PRECOMPUTED_PATHS.length) * 0.5 + (i * 0.23)) % 1,
  speed: 0.02 + (i % 4) * 0.005,
}));

export default function Home() {
  const [drivers, setDrivers] = useState<MarkerLocation[]>(() => {
    return DRIVERS_DATA.map((cfg) => {
      const path = PRECOMPUTED_PATHS[cfg.roadIndex] || [];
      const maxIdx = Math.max(0, path.length - 1);
      const idx = Math.min(Math.max(0, Math.floor(cfg.initialProgress * maxIdx)), maxIdx);
      const pt = path[idx] || { lat: 40.7580, lng: -73.9855, heading: 0 };
      return {
        id: cfg.id,
        lat: pt.lat,
        lng: pt.lng,
        heading: pt.heading,
      };
    });
  });

  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const progressesRef = useRef<number[]>(DRIVERS_DATA.map((d) => d.initialProgress));

  // 60fps requestAnimationFrame loop
  useEffect(() => {
    let animId: number;
    let lastTime = performance.now();

    const animate = (time: number) => {
      const delta = Math.min((time - lastTime) / 1000, 0.05);
      lastTime = time;

      const updated: MarkerLocation[] = [];
      for (let i = 0; i < DRIVERS_DATA.length; i++) {
        const cfg = DRIVERS_DATA[i];
        let p = ((progressesRef.current[i] + cfg.speed * delta) % 1 + 1) % 1;
        progressesRef.current[i] = p;

        const path = PRECOMPUTED_PATHS[cfg.roadIndex];
        if (!path || path.length === 0) continue;

        const floatIdx = p * (path.length - 1);
        const i0 = Math.min(Math.max(0, Math.floor(floatIdx)), path.length - 1);
        const i1 = (i0 + 1) % path.length;
        const frac = floatIdx - i0;

        const pt0 = path[i0];
        const pt1 = path[i1] || pt0;

        if (!pt0) continue;

        const lat = pt0.lat + ((pt1.lat - pt0.lat) * frac || 0);
        const lng = pt0.lng + ((pt1.lng - pt0.lng) * frac || 0);
        const heading = pt0.heading;

        updated.push({
          id: cfg.id,
          lat,
          lng,
          heading,
        });
      }

      setDrivers(updated);
      animId = requestAnimationFrame(animate);
    };

    animId = requestAnimationFrame(animate);
    return () => cancelAnimationFrame(animId);
  }, []);

  return (
    <div className="bg-slate-950 text-white min-h-screen relative overflow-x-hidden font-sans antialiased selection:bg-[#007AFF]/30 selection:text-white">
      {/* Subtle Noise Texture Overlay (3% opacity) */}
      <div 
        className="fixed inset-0 pointer-events-none z-[1] opacity-[0.03]"
        style={{
          backgroundImage: `url("data:image/svg+xml,%3Csvg viewBox='0 0 200 200' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noiseFilter'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.8' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noiseFilter)'/%3E%3C/svg%3E")`
        }}
      />

      {/* Global Background Map */}
      <div className="fixed inset-0 z-[0] pointer-events-none">
        <MapboxView
          center={[-73.9851, 40.7484]}
          zoom={13.2}
          interactive={false}
          drivers={drivers}
          className="w-full h-full"
        />
        {/* Subtle dark gradient overlay to ensure type legibility */}
        <div className="absolute inset-0 bg-gradient-to-b from-slate-950/70 via-slate-950/30 to-slate-950/90 pointer-events-none"></div>
      </div>

      {/* Floating Dynamic Liquid Glass Navbar */}
      <header className="fixed top-4 inset-x-0 z-50 max-w-5xl mx-auto px-4 pointer-events-none">
        <nav className="pointer-events-auto w-full h-[52px] bg-slate-950/70 backdrop-blur-xl backdrop-saturate-150 border border-white/[0.12] rounded-full shadow-[0_8px_32px_rgba(0,0,0,0.5),inset_0_1px_0_rgba(255,255,255,0.15)] px-4 md:px-5 flex items-center justify-between transition-all duration-300">
          
          {/* Brand Logo & Name */}
          <Link href="/" className="flex items-center gap-2.5 text-white font-medium text-sm tracking-tight hover:opacity-90 transition-opacity cursor-pointer">
            <div className="w-6 h-6 rounded-full bg-gradient-to-tr from-[#007AFF] to-blue-400 flex items-center justify-center shadow-sm shadow-blue-500/40">
              <Building2 className="w-3.5 h-3.5 text-white" />
            </div>
            <span className="font-semibold text-white tracking-tight">Urban Prime</span>
            
          </Link>
          
          {/* Centered Navigation Links */}
          
          
          {/* Action CTAs */}
          <div className="hidden sm:flex items-center gap-2">
            <Link 
              href="/driver" 
              className="text-xs font-medium text-white/70 hover:text-white px-3 py-1.5 transition-colors cursor-pointer"
            >
              Driver
            </Link>
            <Link 
              href="/rider" 
              className="bg-[#007AFF] hover:bg-[#0069D9] text-white px-4 py-1.5 rounded-full text-xs font-medium transition-all shadow-md shadow-blue-500/25 active:scale-95 flex items-center gap-1.5 cursor-pointer"
            >
              Book a Ride
              <ArrowRight className="w-3 h-3" />
            </Link>
          </div>
          
          {/* Mobile Menu Toggle */}
          <button 
            className="md:hidden text-white/80 hover:text-white p-1.5 cursor-pointer"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            aria-label="Toggle menu"
          >
            <Menu className="w-4 h-4" />
          </button>
        </nav>

        {/* Mobile Dropdown Menu */}
        {mobileMenuOpen && (
          <div className="pointer-events-auto md:hidden mt-2 p-4 rounded-2xl bg-slate-950/90 backdrop-blur-2xl border border-white/10 shadow-2xl flex flex-col gap-3">
            <a 
              href="#architecture" 
              onClick={() => setMobileMenuOpen(false)}
              className="text-sm font-medium text-white/80 hover:text-white px-3 py-2 rounded-lg hover:bg-white/5"
            >
              Architecture & Microservices
            </a>
            <Link 
              href="/rider" 
              onClick={() => setMobileMenuOpen(false)}
              className="text-sm font-medium text-white/80 hover:text-white px-3 py-2 rounded-lg hover:bg-white/5"
            >
              Customer Web App
            </Link>
            <Link 
              href="/driver" 
              onClick={() => setMobileMenuOpen(false)}
              className="text-sm font-medium text-white/80 hover:text-white px-3 py-2 rounded-lg hover:bg-white/5"
            >
              Driver Partner Cockpit
            </Link>
            <div className="pt-2 border-t border-white/10 flex gap-2">
              <Link 
                href="/rider" 
                className="flex-1 bg-[#007AFF] text-white text-center py-2 rounded-lg text-xs font-medium"
              >
                Book a Ride
              </Link>
              <Link 
                href="/driver" 
                className="flex-1 bg-slate-900 text-white text-center py-2 rounded-lg text-xs font-medium border border-white/10"
              >
                Driver Cockpit
              </Link>
            </div>
          </div>
        )}
      </header>

      {/* Main Content (Left-Aligned, max-w-5xl) */}
      <main className="pt-[140px] pb-32 relative z-10 max-w-5xl mx-auto px-6 flex flex-col gap-32">
        
        {/* Left-Aligned Hero Section (No Container Box, Let Map Breathe) */}
        <section className="flex flex-col items-start text-left max-w-3xl">
          {/* Portfolio Architecture Pill */}
         

          <h1 className="text-5xl md:text-6xl lg:text-[72px] font-medium text-white mb-6 leading-[1.05] tracking-tight">
            Real-Time Ride-Hailing <br />
            Microservice Platform
          </h1>
          
          <p className="text-[17px] md:text-[20px] text-white/85 leading-relaxed font-medium mb-10 max-w-2xl">
            A production-grade distributed backend ecosystem built with Go microservices, Apache Kafka event streaming, Redis geospatial indexing, and Saga-orchestrated transactions.
          </p>

          <div className="flex flex-col sm:flex-row gap-4 items-center">
            <Link 
              href="/rider" 
              className="bg-[#007AFF] text-white px-6 py-3 rounded-lg text-sm font-medium hover:bg-[#0069D9] transition-colors flex items-center justify-center gap-2 cursor-pointer active:scale-95 shadow-lg shadow-blue-500/25"
            >
              Book a Ride
              <ArrowRight className="w-4 h-4" />
            </Link>
            <Link 
              href="/driver" 
              className="bg-[#0a0f1e]/80 text-white/90 px-6 py-3 rounded-lg text-sm font-medium border border-white/15 hover:border-white/25 hover:text-white transition-colors flex items-center justify-center gap-2 cursor-pointer active:scale-95"
            >
              Drive with Us
            </Link>
          </div>
        </section>

        {/* Liquid Glass App Portals Section with Background Watermark Icons */}
        <section className="w-full">
          <div className="grid md:grid-cols-2 gap-6 items-start">
            {/* Premium Customer / Passenger Card */}
            <Link 
              href="/rider" 
              className="group relative bg-slate-900/35 backdrop-blur-xl backdrop-saturate-150 p-8 md:p-10 rounded-3xl border border-white/15 shadow-[inset_0_1px_0_rgba(255,255,255,0.15),0_8px_30px_rgba(0,0,0,0.3)] overflow-hidden transition-all hover:border-white/25 hover:bg-slate-900/45 hover:-translate-y-1 cursor-pointer"
            >
              {/* Background Watermark Icon (No Background) */}
              <div className="absolute -top-4 -right-4 p-4 opacity-5 group-hover:opacity-10 transition-opacity rotate-6 pointer-events-none">
                <svg viewBox="0 0 512 512" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-52 h-52 text-white">
                  <circle cx="256" cy="150" r="115" fill="currentColor" />
                  <path d="M52 452c0-105 91-190 204-190s204 85 204 190c0 10-8 18-18 18H70c-10 0-18-8-18-18z" fill="currentColor" />
                </svg>
              </div>
              <div className="relative z-10">
                <div className="w-12 h-12 rounded-xl bg-[#007AFF] flex items-center justify-center mb-6 shadow-md shadow-blue-500/30">
                  <svg viewBox="0 0 512 512" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-6 h-6 text-white">
                    <circle cx="256" cy="150" r="115" fill="currentColor" fillOpacity="0.9" />
                    <path d="M52 452c0-105 91-190 204-190s204 85 204 190c0 10-8 18-18 18H70c-10 0-18-8-18-18z" fill="currentColor" />
                  </svg>
                </div>
                <h3 className="text-2xl font-semibold text-white mb-3 tracking-tight">Customer App</h3>
                <p className="text-sm text-white/70 leading-relaxed font-normal mb-8 pr-4">
                  Interactive Mapbox booking interface, dynamic ride tier selection (Economy, Comfort, Executive), and live GPS driver tracking.
                </p>
                <div className="flex items-center gap-2 text-sm font-semibold text-white group-hover:text-[#007AFF] transition-colors">
                  Open Customer App <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
                </div>
              </div>
            </Link>

            {/* Driver Card */}
            <Link 
              href="/driver" 
              className="group relative bg-slate-900/35 backdrop-blur-xl backdrop-saturate-150 p-8 md:p-10 rounded-3xl border border-white/15 shadow-[inset_0_1px_0_rgba(255,255,255,0.15),0_8px_30px_rgba(0,0,0,0.3)] overflow-hidden transition-all hover:border-white/25 hover:bg-slate-900/45 hover:-translate-y-1 cursor-pointer"
            >
              {/* Background Watermark Icon */}
              <div className="absolute -top-6 -right-6 p-6 opacity-5 group-hover:opacity-10 transition-opacity -rotate-12 pointer-events-none">
                <ShieldCheck className="w-48 h-48 text-white" />
              </div>
              <div className="relative z-10">
                <div className="w-12 h-12 rounded-xl bg-[#34C759] flex items-center justify-center mb-6 shadow-md shadow-emerald-500/30">
                  <ShieldCheck className="w-6 h-6 text-white" />
                </div>
                <h3 className="text-xl font-semibold text-white mb-3 tracking-tight">Driver Partner Cockpit</h3>
                <p className="text-sm text-white/70 leading-relaxed font-normal mb-8 pr-4">
                  Real-time earnings metrics, online/offline availability toggle, GPS beacon broadcasting, and 15-second ride dispatch acceptance.
                </p>
                <div className="flex items-center gap-2 text-sm font-semibold text-white group-hover:text-[#34C759] transition-colors">
                  Launch Driver Cockpit <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
                </div>
              </div>
            </Link>
          </div>
        </section>

        {/* How It Works (Simple Numbered List with Left Border Accent - No Box Cards) */}
        <section className="w-full">
          <div className="mb-12">
            <h2 className="text-2xl md:text-3xl font-medium text-white tracking-tight mb-2">The Trip Lifecycle</h2>
            <p className="text-white/40 text-sm font-normal">How our microservices match riders and drivers in sub-50 milliseconds.</p>
          </div>

          <div className="grid sm:grid-cols-2 md:grid-cols-4 gap-8">
            {[
              { num: '01', title: 'Request Intent', desc: 'Rider inputs pickup & dropoff. APISIX gateway routes to Trip Saga Service.' },
              { num: '02', title: 'Spatial Query', desc: 'Redis Geospatial searches nearest available drivers in real time.' },
              { num: '03', title: 'Event Fan-out', desc: 'Kafka event streams broadcast ride requests directly to Driver Cockpits.' },
              { num: '04', title: 'Saga Settlement', desc: 'Saga orchestrator locks payment authorization and confirms reservation.' },
            ].map((step, i) => (
              <div key={i} className="border-l border-white/10 pl-5 py-2 flex flex-col justify-between">
                <div>
                  <span className="font-mono text-xs text-[#007AFF] font-medium block mb-2">{step.num}</span>
                  <h4 className="text-base font-medium text-white mb-2 tracking-tight">{step.title}</h4>
                  <p className="text-xs text-white/40 leading-relaxed font-normal">{step.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </section>

        {/* Under the Hood / Global Scale */}
        <section className="w-full">
          <div className="grid md:grid-cols-3 gap-6">
            <div className="md:col-span-2 bg-[#0a0f1e] p-8 md:p-10 rounded-2xl border border-white/[0.08] hover:border-white/15 transition-colors flex flex-col justify-between">
              <div>
                <div className="w-10 h-10 rounded-lg bg-slate-900 border border-white/10 flex items-center justify-center mb-6">
                  <Globe className="w-5 h-5 text-white/70" />
                </div>
                <h4 className="text-xl font-medium text-white mb-2 tracking-tight">Global Redis Edge</h4>
                <p className="text-sm text-white/50 leading-relaxed font-normal max-w-md mb-8">
                  Our geospatial index runs at the edge, ensuring ride matchmaking queries resolve in under 10ms regardless of your city.
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                <span className="bg-slate-900 px-3 py-1 rounded text-xs font-mono text-white/60 border border-white/5">NYC Region</span>
                <span className="bg-slate-900 px-3 py-1 rounded text-xs font-mono text-white/60 border border-white/5">LDN Region</span>
                <span className="bg-slate-900 px-3 py-1 rounded text-xs font-mono text-white/60 border border-white/5">TYO Region</span>
              </div>
            </div>

            <div className="bg-[#0a0f1e] p-8 rounded-2xl border border-white/[0.08] hover:border-white/15 transition-colors flex flex-col justify-between">
              <div>
                <div className="w-10 h-10 rounded-lg bg-slate-900 border border-white/10 flex items-center justify-center mb-6">
                  <Cpu className="w-5 h-5 text-white/70" />
                </div>
                <h4 className="text-xl font-medium text-white mb-2 tracking-tight">Saga Pattern</h4>
                <p className="text-sm text-white/50 leading-relaxed font-normal">
                  Distributed transactions ensure financial rollbacks if a driver cancels or a payment hold fails.
                </p>
              </div>
              <div className="text-xs font-mono text-emerald-400 mt-6 flex items-center gap-1.5">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span> Zero split-brain state
              </div>
            </div>
          </div>
        </section>

        {/* Technical Architecture Specs Bar */}
        <section id="architecture" className="w-full">
          <div className="border-t border-b border-white/[0.08] py-10 grid grid-cols-1 md:grid-cols-3 gap-8">
            <div className="flex flex-col">
              <span className="font-mono text-3xl md:text-4xl font-normal text-white tracking-tight mb-1">6 Services</span>
              <span className="text-xs uppercase tracking-widest text-white/40 font-mono">Go Microservices & gRPC</span>
            </div>
            <div className="flex flex-col">
              <span className="font-mono text-3xl md:text-4xl font-normal text-white tracking-tight mb-1">Kafka + Redis</span>
              <span className="text-xs uppercase tracking-widest text-white/40 font-mono">Event Streams & Geospatial</span>
            </div>
            <div className="flex flex-col">
              <span className="font-mono text-3xl md:text-4xl font-normal text-white tracking-tight mb-1">Saga Pattern</span>
              <span className="text-xs uppercase tracking-widest text-white/40 font-mono">Distributed Transactions</span>
            </div>
          </div>
        </section>

        {/* Microservices Topology Grid (Engineered for Technical Interviewers & Recruiters) */}
        <section className="w-full">
          <div className="mb-12">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-500/10 border border-blue-500/20 text-[#007AFF] text-xs font-mono mb-4">
              <Cpu className="w-3.5 h-3.5" /> Service Architecture Topology
            </div>
            <h2 className="text-2xl md:text-3xl font-medium text-white tracking-tight mb-2">Microservice Ecosystem</h2>
            <p className="text-white/50 text-sm font-normal max-w-xl">
              Decoupled, event-driven Go services communicating via low-latency gRPC and asynchronous Kafka event streams.
            </p>
          </div>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {[
              {
                name: 'Trip Service',
                port: ':50051',
                badge: 'Saga Orchestrator',
                badgeColor: 'text-amber-400 border-amber-500/30 bg-amber-500/10',
                desc: 'Orchestrates the distributed trip state machine (PENDING → MATCHING → ACCEPTED → IN_PROGRESS → COMPLETED) and executes compensating transactions on failure.',
                stack: ['Go 1.22', 'PostgreSQL 16', 'Kafka', 'OSRM Engine'],
              },
              {
                name: 'Location Service',
                port: ':50053',
                badge: 'Zero-DB Pipeline',
                badgeColor: 'text-emerald-400 border-emerald-500/30 bg-emerald-500/10',
                desc: 'Ingests continuous GPS driver pings directly into Redis in-memory GEO pipelines (GEOADD) and Kafka streams, bypassing relational DB to eliminate disk I/O bottlenecks.',
                stack: ['Go 1.22', 'Redis Geo', 'Kafka Pub/Sub'],
              },
              {
                name: 'Driver Service',
                port: ':50052',
                badge: 'Atomic Locks',
                badgeColor: 'text-cyan-400 border-cyan-500/30 bg-cyan-500/10',
                desc: 'Executes sub-millisecond Redis GEOSEARCH driver proximity queries with atomic Redis SetNX distributed locks to prevent double-dispatch race conditions.',
                stack: ['Go 1.22', 'Redis 7', 'PostgreSQL'],
              },
              {
                name: 'Payment Service',
                port: ':50054',
                badge: 'Idempotent Ledger',
                badgeColor: 'text-purple-400 border-purple-500/30 bg-purple-500/10',
                desc: 'Handles pre-authorization holds, fare calculation, and instant payment release using idempotency keys to ensure exactly-once financial transactions.',
                stack: ['Go 1.22', 'Stripe API', 'PostgreSQL'],
              },
              {
                name: 'Notification Hub',
                port: ':50055',
                badge: 'WebSocket Push',
                badgeColor: 'text-rose-400 border-rose-500/30 bg-rose-500/10',
                desc: 'Powered by Centrifugo to multiplex 100k+ concurrent client WebSocket channels, streaming real-time driver coordinates and dispatch states without blocking Go routines.',
                stack: ['Centrifugo', 'WebSockets', 'Redis PUB/SUB'],
              },
              {
                name: 'API Gateway',
                port: ':9080',
                badge: 'Edge Gateway',
                badgeColor: 'text-blue-400 border-blue-500/30 bg-blue-500/10',
                desc: 'Apache APISIX reverse proxy performing JWT authentication termination, gRPC-web transcoding, and Redis-backed leaky bucket rate limiting.',
                stack: ['Apache APISIX', 'LuaJIT', 'gRPC-Web'],
              },
            ].map((svc, i) => (
              <div key={i} className="bg-[#0a0f1e] p-6 rounded-2xl border border-white/[0.08] hover:border-white/20 transition-all flex flex-col justify-between group">
                <div>
                  <div className="flex items-center justify-between mb-4">
                    <span className="font-mono text-xs text-white/40">{svc.port}</span>
                    <span className={`text-[11px] font-mono px-2 py-0.5 rounded-full border ${svc.badgeColor}`}>
                      {svc.badge}
                    </span>
                  </div>
                  <h4 className="text-lg font-medium text-white mb-2 tracking-tight group-hover:text-[#007AFF] transition-colors">{svc.name}</h4>
                  <p className="text-xs text-white/50 leading-relaxed font-normal mb-6">{svc.desc}</p>
                </div>
                <div className="flex flex-wrap gap-1.5 pt-4 border-t border-white/5">
                  {svc.stack.map((tech, j) => (
                    <span key={j} className="text-[10px] font-mono bg-slate-900 text-white/60 px-2 py-0.5 rounded border border-white/5">
                      {tech}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </section>

        {/* Distributed Systems Engineering Decisions (Why & How) */}
        <section className="w-full">
          <div className="mb-12">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs font-mono mb-4">
              <Zap className="w-3.5 h-3.5" /> Distributed Engineering Highlights
            </div>
            <h2 className="text-2xl md:text-3xl font-medium text-white tracking-tight mb-2">Key Architectural Solutions</h2>
            <p className="text-white/50 text-sm font-normal max-w-xl">
              Solutions to high-concurrency challenges encountered in large-scale real-time mobility systems.
            </p>
          </div>

          <div className="grid md:grid-cols-2 gap-6">
            <div className="bg-[#0a0f1e] p-8 rounded-2xl border border-white/[0.08] hover:border-white/15 transition-colors">
              <div className="w-9 h-9 rounded-lg bg-slate-900 border border-white/10 flex items-center justify-center mb-5 text-[#007AFF]">
                <Database className="w-4 h-4" />
              </div>
              <h3 className="text-lg font-medium text-white mb-2 tracking-tight">Zero-DB GPS Firehose Ingestion</h3>
              <p className="text-sm text-white/50 leading-relaxed font-normal">
                Continuous driver location beacons (every 3 seconds) bypass PostgreSQL completely. Location Service ingests updates straight to in-memory Redis GEO structures via pipelined commands and streams to Kafka, reducing relational database I/O to zero.
              </p>
            </div>

            <div className="bg-[#0a0f1e] p-8 rounded-2xl border border-white/[0.08] hover:border-white/15 transition-colors">
              <div className="w-9 h-9 rounded-lg bg-slate-900 border border-white/10 flex items-center justify-center mb-5 text-emerald-400">
                <Radio className="w-4 h-4" />
              </div>
              <h3 className="text-lg font-medium text-white mb-2 tracking-tight">Atomic Redis Distributed Locks</h3>
              <p className="text-sm text-white/50 leading-relaxed font-normal">
                During driver match fan-out, atomic Redis <code className="text-white/70 font-mono text-xs bg-slate-900 px-1 py-0.5 rounded">SetNX</code> leases lock the driver record for 15 seconds. This eliminates race conditions where two simultaneous riders attempt to accept the same driver.
              </p>
            </div>

            <div className="bg-[#0a0f1e] p-8 rounded-2xl border border-white/[0.08] hover:border-white/15 transition-colors">
              <div className="w-9 h-9 rounded-lg bg-slate-900 border border-white/10 flex items-center justify-center mb-5 text-amber-400">
                <Cpu className="w-4 h-4" />
              </div>
              <h3 className="text-lg font-medium text-white mb-2 tracking-tight">Saga Distributed Transaction Compensation</h3>
              <p className="text-sm text-white/50 leading-relaxed font-normal">
                Avoids rigid 2-Phase Commit (2PC) locks. If a driver rejects or times out, the Saga Orchestrator dispatches automated compensation events across Kafka to release the payment authorization hold and seamlessly re-queue the rider.
              </p>
            </div>

            <div className="bg-[#0a0f1e] p-8 rounded-2xl border border-white/[0.08] hover:border-white/15 transition-colors">
              <div className="w-9 h-9 rounded-lg bg-slate-900 border border-white/10 flex items-center justify-center mb-5 text-purple-400">
                <Network className="w-4 h-4" />
              </div>
              <h3 className="text-lg font-medium text-white mb-2 tracking-tight">Decoupled Centrifugo WebSocket Hub</h3>
              <p className="text-sm text-white/50 leading-relaxed font-normal">
                Centrifugo acts as a dedicated pub/sub WebSocket proxy. Go backend services remain pure stateless workers, publishing broadcast events to Redis channels while Centrifugo handles thousands of live browser socket connections.
              </p>
            </div>
          </div>
        </section>

        {/* Tech Stack Matrix */}
        <section className="w-full">
          <div className="mb-8">
            <h2 className="text-xl font-medium text-white tracking-tight mb-2">Technology Stack</h2>
            <p className="text-white/40 text-xs font-normal">Tools and protocols powering this distributed implementation.</p>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3">
            {[
              { label: 'Go (Golang)', sub: 'Microservices & gRPC' },
              { label: 'Apache Kafka', sub: 'Event Streaming' },
              { label: 'Redis 7', sub: 'Geo & In-Memory Cache' },
              { label: 'PostgreSQL 16', sub: 'Relational Store' },
              { label: 'Apache APISIX', sub: 'API Gateway' },
              { label: 'Next.js 16', sub: 'App Router & TS' },
            ].map((tech, i) => (
              <div key={i} className="bg-[#0a0f1e] p-4 rounded-xl border border-white/[0.08] flex flex-col justify-center">
                <span className="font-mono text-sm font-medium text-white">{tech.label}</span>
                <span className="text-[11px] text-white/40 font-normal mt-0.5">{tech.sub}</span>
              </div>
            ))}
          </div>
        </section>

        {/* Final CTA */}
        <section className="w-full">
          <div className="bg-[#0d1526] rounded-2xl p-10 md:p-14 text-left border border-[#007AFF]/20 flex flex-col md:flex-row justify-between items-start md:items-center gap-8">
            <div>
              <h2 className="text-2xl md:text-3xl font-medium text-white mb-2 tracking-tight">Test the Live System</h2>
              <p className="text-sm text-white/50 max-w-md font-normal">
                Interact with the Customer map booking interface or open the Driver Cockpit to simulate live dispatch events.
              </p>
            </div>
            <div className="flex flex-col sm:flex-row gap-3">
              <Link 
                href="/rider" 
                className="bg-[#007AFF] text-white px-6 py-3 rounded-lg text-sm font-medium hover:bg-[#0069D9] transition-colors cursor-pointer active:scale-95 whitespace-nowrap text-center"
              >
                Customer App
              </Link>
              <Link 
                href="/driver" 
                className="bg-[#0a0f1e] text-white/80 px-6 py-3 rounded-lg text-sm font-medium border border-white/15 hover:text-white transition-colors cursor-pointer active:scale-95 whitespace-nowrap text-center"
              >
                Driver Cockpit
              </Link>
            </div>
          </div>
        </section>
      </main>

      {/* Clean Linear/Vercel-style Footer */}
      <footer className="bg-slate-950 border-t border-white/[0.08] w-full py-10 relative z-10">
        <div className="max-w-5xl mx-auto px-6 flex flex-col md:flex-row justify-between items-center gap-6">
          <div className="flex flex-col gap-1 items-center md:items-start text-center md:text-left">
            <div className="font-semibold text-white text-sm flex items-center gap-2 tracking-tight">
              <div className="w-5 h-5 rounded bg-slate-900 border border-white/10 flex items-center justify-center">
                <Building2 className="w-3 h-3 text-white" />
              </div>
              Urban Prime
            </div>
            <p className="text-[11px] font-normal text-white/30">© 2026 Urban Prime Inc. All rights reserved.</p>
          </div>
          <div className="flex flex-wrap justify-center gap-6 text-xs font-normal text-white/40">
            <span className="hover:text-white cursor-pointer transition-colors">Privacy Policy</span>
            <span className="hover:text-white cursor-pointer transition-colors">Terms of Use</span>
            <span className="hover:text-white cursor-pointer transition-colors">Security</span>
            <span className="hover:text-white cursor-pointer transition-colors">System Status</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
