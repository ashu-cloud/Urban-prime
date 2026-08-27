'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { getStoredRiderSession, clearStoredRiderSession } from '@/lib/api';
import { Compass, Car, Clock, HelpCircle, Bell, LogOut } from 'lucide-react';

interface NavbarProps {
  activeTab?: 'ride' | 'activity' | 'help';
  onOpenActivity?: () => void;
  onOpenHelp?: () => void;
}

export default function Navbar({ activeTab = 'ride', onOpenActivity, onOpenHelp }: NavbarProps) {
  const router = useRouter();
  const [mounted, setMounted] = useState(false);
  const [session, setSession] = useState<any>(null);

  useEffect(() => {
    setMounted(true);
    setSession(getStoredRiderSession());
  }, []);

  const handleLogout = () => {
    clearStoredRiderSession();
    router.push('/');
  };

  return (
    <header className="h-[72px] bg-white border-b border-[#DCD9D9] px-8 flex items-center justify-between sticky top-0 z-30 select-none shadow-xs">
      {/* Brand Logo */}
      <div className="flex items-center gap-8">
        <Link href="/rider" className="flex items-center gap-2.5 group">
          <div className="w-10 h-10 rounded-xl bg-[#276EF1] flex items-center justify-center text-white shadow-md shadow-blue-500/20 group-hover:scale-105 transition-transform">
            <Compass className="w-5 h-5" />
          </div>
          <div>
            <span className="font-extrabold text-xl tracking-tight text-[#1F1F1F]">URBAN<span className="text-[#276EF1]">PRIME</span></span>
            <span className="block text-[10px] tracking-widest uppercase font-semibold text-slate-400 -mt-1">Rider Experience</span>
          </div>
        </Link>

        {/* Center Navigation */}
        <nav className="hidden md:flex items-center gap-1 bg-[#FCF9F8] p-1.5 rounded-full border border-[#DCD9D9]/80">
          <Link
            href="/rider"
            className={`flex items-center gap-2 px-4 py-2 rounded-full text-sm font-semibold transition-all ${
              activeTab === 'ride'
                ? 'bg-white text-[#276EF1] shadow-xs'
                : 'text-slate-600 hover:text-slate-900'
            }`}
          >
            <Car className="w-4 h-4" />
            Book Ride
          </Link>
          <button
            onClick={() => {
              if (onOpenActivity) {
                onOpenActivity();
              } else {
                alert('Trip history is available in the Rider app.');
              }
            }}
            className={`flex items-center gap-2 px-4 py-2 rounded-full text-sm font-medium transition-all ${
              activeTab === 'activity'
                ? 'bg-white text-[#276EF1] shadow-xs'
                : 'text-slate-600 hover:text-slate-900'
            }`}
          >
            <Clock className="w-4 h-4" />
            Activity
          </button>
          <button
            onClick={() => {
              if (onOpenHelp) {
                onOpenHelp();
              } else {
                alert('24/7 Priority Concierge available for Urban Prime members.');
              }
            }}
            className={`flex items-center gap-2 px-4 py-2 rounded-full text-sm font-medium transition-all ${
              activeTab === 'help'
                ? 'bg-white text-[#276EF1] shadow-xs'
                : 'text-slate-600 hover:text-slate-900'
            }`}
          >
            <HelpCircle className="w-4 h-4" />
            Help
          </button>
        </nav>
      </div>

      {/* Right Controls */}
      <div className="flex items-center gap-4">
        <button
          title="Notifications"
          className="p-2.5 rounded-full text-slate-500 hover:text-slate-900 hover:bg-slate-100 transition-colors relative"
        >
          <Bell className="w-5 h-5" />
          <span className="w-2 h-2 rounded-full bg-[#276EF1] absolute top-2 right-2 ring-2 ring-white"></span>
        </button>

        {!mounted || !session ? (
          <div className="flex items-center gap-2">
            <Link
              href="/rider/login"
              className="px-4 py-2 text-xs font-bold text-slate-700 hover:text-[#276EF1] transition-colors"
            >
              Sign In
            </Link>
            <Link
              href="/rider/signup"
              className="px-4 py-2 bg-[#276EF1] hover:bg-[#1A54C9] text-white text-xs font-bold rounded-xl transition-all shadow-md shadow-blue-500/20 active:scale-95"
            >
              Sign Up
            </Link>
          </div>
        ) : (
          <div className="flex items-center gap-3 pl-2 border-l border-[#DCD9D9]">
            <div className="w-9 h-9 rounded-full bg-[#E7F0FF] text-[#276EF1] font-bold text-sm flex items-center justify-center border border-[#276EF1]/20">
              {session.name ? session.name[0] : 'U'}
            </div>
            <div className="hidden lg:block text-left">
              <p className="text-xs font-bold text-[#1F1F1F] leading-tight">{session.name || 'Prime Member'}</p>
              <p className="text-[10px] text-emerald-600 font-semibold uppercase tracking-wider">{session.role}</p>
            </div>
            <button
              onClick={handleLogout}
              title="Sign Out"
              className="p-2 text-slate-400 hover:text-red-600 transition-colors rounded-lg hover:bg-red-50 ml-1"
            >
              <LogOut className="w-4 h-4" />
            </button>
          </div>
        )}
      </div>
    </header>
  );
}
