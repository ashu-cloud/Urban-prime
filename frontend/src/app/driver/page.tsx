'use client';

import React, { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import MapboxView, { MarkerLocation, RouteLegType } from '@/components/map/MapboxView';
import { getStoredSession } from '@/lib/api';
import { tripStore } from '@/lib/tripStore';
import {
  realtimeBus,
  DispatchOfferEvent,
  TripStatusEvent,
  getDistanceInMeters,
  TripLifecycleStage,
} from '@/lib/socket';
import {
  Compass,
  Power,
  DollarSign,
  TrendingUp,
  Award,
  Navigation,
  MapPin,
  Clock,
  CheckCircle2,
  XCircle,
  Car,
  Bell,
  Star,
  Activity,
  KeyRound,
  ShieldAlert,
  Zap,
  Check,
} from 'lucide-react';

export default function DriverPage() {
  const session = getStoredSession();
  const [isOnline, setIsOnline] = useState(true);
  const [earnings, setEarnings] = useState(284.50);
  const [acceptanceRate] = useState('98.4%');
  const [completedTrips, setCompletedTrips] = useState(12);

  // Driver Current GPS Position
  const [driverPos, setDriverPos] = useState<MarkerLocation>({
    lat: 40.7440,
    lng: -73.9900,
    heading: 45,
    label: 'Marcus Sterling (Tesla Model S)',
  });

  // Active Dispatch Modal State
  const [activeOffer, setActiveOffer] = useState<DispatchOfferEvent | null>(null);
  const [secondsRemaining, setSecondsRemaining] = useState(15);
  const timerRef = useRef<any>(null);

  // Active Accepted Trip State & Lifecycle
  const [activeTrip, setActiveTrip] = useState<DispatchOfferEvent | null>(null);
  const [tripStage, setTripStage] = useState<TripLifecycleStage>('MATCHING');

  // OTP Verification State
  const [enteredOtp, setEnteredOtp] = useState(['', '', '', '']);
  const [otpError, setOtpError] = useState(false);
  const [isOtpVerified, setIsOtpVerified] = useState(false);
  const otpInputRefs = useRef<(HTMLInputElement | null)[]>([]);

  // 1. HYDRATE SINGLE SOURCE OF TRUTH ON MOUNT (SURVIVES REFRESH)
  useEffect(() => {
    const saved = tripStore.get();
    if (saved && saved.status !== 'COMPLETED') {
      const offer: DispatchOfferEvent = {
        tripId: saved.tripId,
        riderId: saved.riderId,
        riderName: saved.riderName,
        pickupAddress: saved.pickupAddress,
        dropoffAddress: saved.dropoffAddress,
        pickupLat: saved.pickupLat,
        pickupLng: saved.pickupLng,
        dropoffLat: saved.dropoffLat,
        dropoffLng: saved.dropoffLng,
        fareAmount: saved.fareAmount,
        expiresInSeconds: 15,
        otp: saved.otp,
      };

      if (saved.status === 'MATCHING') {
        setActiveOffer(offer);
      } else {
        setActiveTrip(offer);
        setTripStage(saved.status);
        if (saved.status === 'IN_TRANSIT' || saved.status === 'ARRIVED_AT_DESTINATION') {
          setIsOtpVerified(true);
        }
        if (saved.driverLat && saved.driverLng) {
          setDriverPos({
            lat: saved.driverLat,
            lng: saved.driverLng,
            heading: saved.driverHeading || 45,
            label: 'Marcus Sterling (Tesla Model S)',
          });
        }
      }
    }
  }, []);

  // Periodically simulate driver GPS movement and publish over realtimeBus
  useEffect(() => {
    if (!isOnline) return;

    const interval = setInterval(() => {
      setDriverPos((prev) => {
        let nextLat = prev.lat;
        let nextLng = prev.lng;
        let heading = prev.heading || 45;

        // If driving to pickup
        if (activeTrip && tripStage === 'ACCEPTED_EN_ROUTE_PICKUP') {
          const step = 0.0003;
          const dLat = activeTrip.pickupLat - prev.lat;
          const dLng = activeTrip.pickupLng - prev.lng;
          nextLat += Math.sign(dLat) * Math.min(Math.abs(dLat), step);
          nextLng += Math.sign(dLng) * Math.min(Math.abs(dLng), step);
          heading = (Math.atan2(dLng, dLat) * 180) / Math.PI;
        } 
        // If in transit to destination
        else if (activeTrip && tripStage === 'IN_TRANSIT') {
          const step = 0.0004;
          const dLat = activeTrip.dropoffLat - prev.lat;
          const dLng = activeTrip.dropoffLng - prev.lng;
          nextLat += Math.sign(dLat) * Math.min(Math.abs(dLat), step);
          nextLng += Math.sign(dLng) * Math.min(Math.abs(dLng), step);
          heading = (Math.atan2(dLng, dLat) * 180) / Math.PI;
        } 
        // Idle wandering
        else {
          nextLat += (Math.random() - 0.5) * 0.0003;
          nextLng += (Math.random() - 0.5) * 0.0003;
        }

        const newPos = {
          lat: nextLat,
          lng: nextLng,
          heading: Math.round(heading),
          label: 'Marcus Sterling (Tesla Model S)',
        };

        // Publish to Centrifugo/Mesh
        realtimeBus.publishDriverLocation({
          driverId: session?.userId || 'drv_901',
          latitude: newPos.lat,
          longitude: newPos.lng,
          heading: newPos.heading,
          isAvailable: isOnline && !activeTrip,
          driverName: session?.name || 'Marcus Sterling',
          vehicleType: 'PREMIUM',
        });

        // Update single source of truth store
        const active = tripStore.get();
        if (active) {
          tripStore.updateStatus(active.status, {
            driverLat: newPos.lat,
            driverLng: newPos.lng,
            driverHeading: newPos.heading,
          });
        }

        return newPos;
      });
    }, 2000);

    return () => clearInterval(interval);
  }, [isOnline, activeTrip, tripStage, session]);

  // Subscribe to Incoming Dispatch Offers
  useEffect(() => {
    const unsubscribe = realtimeBus.onDispatchOffer((offer: DispatchOfferEvent) => {
      if (isOnline && !activeTrip) {
        setActiveOffer(offer);
        setSecondsRemaining(offer.expiresInSeconds || 15);
      }
    });

    return () => unsubscribe();
  }, [isOnline, activeTrip]);

  // 15s Countdown Timer for Dispatch Offer
  useEffect(() => {
    if (activeOffer) {
      timerRef.current = setInterval(() => {
        setSecondsRemaining((prev) => {
          if (prev <= 1) {
            clearInterval(timerRef.current);
            setActiveOffer(null);
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    } else {
      clearInterval(timerRef.current);
    }
    return () => clearInterval(timerRef.current);
  }, [activeOffer]);

  // Distance calculations
  const distanceToPickup = activeTrip
    ? getDistanceInMeters(driverPos.lat, driverPos.lng, activeTrip.pickupLat, activeTrip.pickupLng)
    : null;

  const distanceToDropoff = activeTrip
    ? getDistanceInMeters(driverPos.lat, driverPos.lng, activeTrip.dropoffLat, activeTrip.dropoffLng)
    : null;

  // Auto-detect arrival at pickup within 20m
  useEffect(() => {
    if (activeTrip && tripStage === 'ACCEPTED_EN_ROUTE_PICKUP' && distanceToPickup !== null) {
      if (distanceToPickup <= 20) {
        setTripStage('ARRIVED_AT_PICKUP');
        tripStore.updateStatus('ARRIVED_AT_PICKUP');
        realtimeBus.publishTripStatus({
          tripId: activeTrip.tripId,
          status: 'ARRIVED_AT_PICKUP',
          driverId: session?.userId || 'drv_901',
          driverName: session?.name || 'Marcus Sterling',
          distanceMeters: distanceToPickup,
        });
      }
    }
  }, [activeTrip, tripStage, distanceToPickup, session]);

  // Auto-detect arrival at dropoff within 20m
  useEffect(() => {
    if (activeTrip && tripStage === 'IN_TRANSIT' && distanceToDropoff !== null) {
      if (distanceToDropoff <= 20) {
        setTripStage('ARRIVED_AT_DESTINATION');
        tripStore.updateStatus('ARRIVED_AT_DESTINATION');
        realtimeBus.publishTripStatus({
          tripId: activeTrip.tripId,
          status: 'ARRIVED_AT_DESTINATION',
          driverId: session?.userId || 'drv_901',
          distanceMeters: distanceToDropoff,
        });
      }
    }
  }, [activeTrip, tripStage, distanceToDropoff, session]);

  // 1. Accept Dispatch Action
  const handleAcceptOffer = () => {
    if (!activeOffer) return;
    const accepted = activeOffer;
    setActiveOffer(null);
    setActiveTrip(accepted);
    setTripStage('ACCEPTED_EN_ROUTE_PICKUP');
    setEnteredOtp(['', '', '', '']);
    setOtpError(false);
    setIsOtpVerified(false);

    tripStore.updateStatus('ACCEPTED_EN_ROUTE_PICKUP', {
      driverId: session?.userId || 'drv_901',
      driverName: session?.name || 'Marcus Sterling',
      driverRating: 4.98,
      vehicleModel: 'Tesla Model S (Obsidian Black)',
      licensePlate: 'NY-7890',
      driverLat: driverPos.lat,
      driverLng: driverPos.lng,
    });

    // Notify Rider screen that driver accepted & is heading to pickup
    realtimeBus.publishTripStatus({
      tripId: accepted.tripId,
      status: 'ACCEPTED_EN_ROUTE_PICKUP',
      driverId: session?.userId || 'drv_901',
      driverName: session?.name || 'Marcus Sterling',
      driverRating: 4.98,
      vehicleModel: 'Tesla Model S (Obsidian Black)',
      licensePlate: 'NY-7890',
      driverLat: driverPos.lat,
      driverLng: driverPos.lng,
      etaMinutes: 3,
      distanceMeters: distanceToPickup || 180,
      otp: accepted.otp,
      pickupCoords: {
        lat: accepted.pickupLat,
        lng: accepted.pickupLng,
        address: accepted.pickupAddress,
      },
      dropoffCoords: {
        lat: accepted.dropoffLat,
        lng: accepted.dropoffLng,
        address: accepted.dropoffAddress,
      },
      fareAmount: accepted.fareAmount,
    });
  };

  const handleDeclineOffer = () => {
    setActiveOffer(null);
  };

  // Quick Simulation Shortcut: Jump within 20m of pickup
  const handleSimulateArrivePickup = () => {
    if (!activeTrip) return;
    setDriverPos({
      lat: activeTrip.pickupLat + 0.0001,
      lng: activeTrip.pickupLng + 0.0001,
      heading: 90,
      label: 'Marcus Sterling (Tesla Model S)',
    });
    setTripStage('ARRIVED_AT_PICKUP');
    tripStore.updateStatus('ARRIVED_AT_PICKUP', {
      driverLat: activeTrip.pickupLat + 0.0001,
      driverLng: activeTrip.pickupLng + 0.0001,
    });
    realtimeBus.publishTripStatus({
      tripId: activeTrip.tripId,
      status: 'ARRIVED_AT_PICKUP',
      driverId: session?.userId || 'drv_901',
      driverName: session?.name || 'Marcus Sterling',
      distanceMeters: 12,
    });
  };

  // Quick Simulation Shortcut: Jump within 20m of dropoff
  const handleSimulateArriveDropoff = () => {
    if (!activeTrip) return;
    setDriverPos({
      lat: activeTrip.dropoffLat + 0.0001,
      lng: activeTrip.dropoffLng + 0.0001,
      heading: 180,
      label: 'Marcus Sterling (Tesla Model S)',
    });
    setTripStage('ARRIVED_AT_DESTINATION');
    tripStore.updateStatus('ARRIVED_AT_DESTINATION', {
      driverLat: activeTrip.dropoffLat + 0.0001,
      driverLng: activeTrip.dropoffLng + 0.0001,
    });
    realtimeBus.publishTripStatus({
      tripId: activeTrip.tripId,
      status: 'ARRIVED_AT_DESTINATION',
      driverId: session?.userId || 'drv_901',
      distanceMeters: 14,
    });
  };

  // 2. Handle OTP Input & Verification
  const handleOtpChange = (index: number, value: string) => {
    if (!/^\d*$/.test(value)) return;
    const newOtp = [...enteredOtp];
    newOtp[index] = value.slice(-1);
    setEnteredOtp(newOtp);
    setOtpError(false);

    if (value && index < 3) {
      otpInputRefs.current[index + 1]?.focus();
    }
  };

  const handleOtpKeyDown = (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Backspace' && !enteredOtp[index] && index > 0) {
      otpInputRefs.current[index - 1]?.focus();
    }
  };

  const handleVerifyOtpAndStart = () => {
    if (!activeTrip) return;
    const fullOtp = enteredOtp.join('');
    const expectedOtp = activeTrip.otp || '8421';

    if (fullOtp === expectedOtp) {
      setIsOtpVerified(true);
      setTripStage('IN_TRANSIT');
      tripStore.updateStatus('IN_TRANSIT');

      // Publish In-Transit status over WebSocket mesh
      realtimeBus.publishTripStatus({
        tripId: activeTrip.tripId,
        status: 'IN_TRANSIT',
        driverId: session?.userId || 'drv_901',
        driverName: session?.name || 'Marcus Sterling',
        etaMinutes: 8,
      });
    } else {
      setOtpError(true);
    }
  };

  // 3. Complete Trip Action (within 20m of destination)
  const handleCompleteTrip = () => {
    if (!activeTrip) return;
    realtimeBus.publishTripStatus({
      tripId: activeTrip.tripId,
      status: 'COMPLETED',
      driverId: session?.userId || 'drv_901',
      fareAmount: activeTrip.fareAmount,
    });

    tripStore.updateStatus('COMPLETED');
    setEarnings((prev) => prev + activeTrip.fareAmount);
    setCompletedTrips((prev) => prev + 1);
    setActiveTrip(null);
    setTripStage('MATCHING');
    setIsOtpVerified(false);
  };

  // Active navigation leg for Mapbox
  let activeLeg: RouteLegType = 'NONE';
  if (activeTrip) {
    if (tripStage === 'ACCEPTED_EN_ROUTE_PICKUP' || tripStage === 'ARRIVED_AT_PICKUP') {
      activeLeg = 'TO_PICKUP';
    } else if (tripStage === 'IN_TRANSIT' || tripStage === 'ARRIVED_AT_DESTINATION') {
      activeLeg = 'TO_DESTINATION';
    }
  }

  // SVG Radial Timer calculation
  const radius = 42;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference - (secondsRemaining / 15) * circumference;

  return (
    <div className="h-screen w-screen flex bg-[#FCF9F8] overflow-hidden select-none">
      {/* 1. LEFT SIDEBAR: Partner Navigation (320px) */}
      <aside className="w-[320px] h-full bg-white border-r border-[#DCD9D9] flex flex-col justify-between z-20 shadow-lg">
        <div>
          {/* Logo */}
          <div className="h-[72px] px-6 flex items-center gap-3 border-b border-[#DCD9D9]">
            <div className="w-10 h-10 rounded-xl bg-[#276EF1] text-white flex items-center justify-center shadow-md shadow-blue-500/20">
              <Compass className="w-5 h-5" />
            </div>
            <div>
              <span className="font-extrabold text-lg tracking-tight text-[#1F1F1F]">
                URBAN<span className="text-[#276EF1]">PRIME</span>
              </span>
              <span className="block text-[10px] tracking-widest uppercase font-bold text-emerald-600">
                Partner Cockpit
              </span>
            </div>
          </div>

          {/* Driver Profile Card */}
          <div className="p-6 border-b border-[#DCD9D9] bg-[#FCF9F8]/60">
            <div className="flex items-center gap-3.5 mb-3">
              <div className="w-12 h-12 rounded-2xl bg-[#E7F0FF] text-[#276EF1] font-extrabold text-base flex items-center justify-center border border-[#276EF1]/20">
                {session?.name ? session.name[0] : 'M'}
              </div>
              <div>
                <h3 className="text-sm font-extrabold text-[#1F1F1F]">{session?.name || 'Marcus Sterling'}</h3>
                <p className="text-xs text-slate-500">Tier: Executive Black Fleet</p>
                <div className="flex items-center gap-1 mt-0.5">
                  <Star className="w-3 h-3 fill-amber-500 text-amber-500" />
                  <span className="text-xs font-bold text-slate-700">4.98 Rating</span>
                </div>
              </div>
            </div>

            {/* Online / Offline Toggle */}
            <button
              onClick={() => setIsOnline(!isOnline)}
              disabled={!!activeTrip}
              className={`w-full py-3 rounded-xl font-bold text-xs uppercase tracking-wider flex items-center justify-center gap-2 transition-all shadow-sm active:scale-95 ${
                isOnline
                  ? 'bg-[#008A5E] text-white shadow-emerald-500/25'
                  : 'bg-slate-200 text-slate-600 hover:bg-slate-300'
              }`}
            >
              <Power className="w-4 h-4" />
              <span>{isOnline ? 'Active & Online' : 'Currently Offline'}</span>
            </button>
          </div>

          {/* Nav Links */}
          <nav className="p-4 space-y-1">
            <Link
              href="/driver"
              className="flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-bold bg-[#E7F0FF] text-[#276EF1]"
            >
              <Activity className="w-4 h-4" />
              Cockpit Overview
            </Link>
            <Link
              href="/rider"
              className="flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-semibold text-slate-600 hover:text-slate-900 hover:bg-slate-100 transition-colors"
            >
              <Car className="w-4 h-4" />
              Switch to Rider View
            </Link>
          </nav>
        </div>

        {/* Status Tag */}
        <div className="p-4 border-t border-[#DCD9D9] text-[11px] text-slate-400 text-center">
          Centrifugo GPS Mesh • 20m Proximity Shield
        </div>
      </aside>

      {/* 2. MAIN DASHBOARD CONTENT */}
      <main className="flex-1 h-full flex flex-col overflow-hidden relative">
        {/* Top Header */}
        <header className="h-[72px] bg-white border-b border-[#DCD9D9] px-8 flex items-center justify-between z-10">
          <div>
            <h1 className="text-xl font-extrabold text-[#1F1F1F] tracking-tight">
              Partner Telemetry & Dispatch
            </h1>
            <p className="text-xs text-slate-500">
              Live GPS beacon transmitting every 2 seconds to APISIX
            </p>
          </div>

          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-emerald-50 text-emerald-800 border border-emerald-200 text-xs font-bold">
              <span className="w-2 h-2 rounded-full bg-emerald-500 animate-ping"></span>
              <span>GPS Connected</span>
            </div>
            <button className="p-2 text-slate-400 hover:text-slate-700 transition-colors">
              <Bell className="w-5 h-5" />
            </button>
          </div>
        </header>

        {/* Dashboard Workspace */}
        <div className="flex-1 grid grid-cols-12 gap-6 p-6 overflow-hidden">
          {/* Left Column: Metrics & Live Map (8 cols) */}
          <div className="col-span-8 flex flex-col gap-6 h-full">
            {/* 3-Column Metrics Row */}
            <div className="grid grid-cols-3 gap-4">
              <div className="p-5 bg-white rounded-2xl border border-[#DCD9D9] shadow-xs">
                <div className="flex items-center justify-between text-slate-400 mb-2">
                  <span className="text-xs font-bold uppercase tracking-wider text-slate-500">
                    Today's Earnings
                  </span>
                  <div className="w-8 h-8 rounded-lg bg-blue-50 text-[#276EF1] flex items-center justify-center">
                    <DollarSign className="w-4 h-4" />
                  </div>
                </div>
                <div className="text-2xl font-extrabold text-[#276EF1]">
                  ${earnings.toFixed(2)}
                </div>
                <span className="text-[11px] text-emerald-600 font-semibold mt-1 flex items-center gap-1">
                  <TrendingUp className="w-3 h-3" /> +18.4% vs yesterday
                </span>
              </div>

              <div className="p-5 bg-white rounded-2xl border border-[#DCD9D9] shadow-xs">
                <div className="flex items-center justify-between text-slate-400 mb-2">
                  <span className="text-xs font-bold uppercase tracking-wider text-slate-500">
                    Acceptance Rate
                  </span>
                  <div className="w-8 h-8 rounded-lg bg-emerald-50 text-[#008A5E] flex items-center justify-center">
                    <Award className="w-4 h-4" />
                  </div>
                </div>
                <div className="text-2xl font-extrabold text-[#1F1F1F]">{acceptanceRate}</div>
                <span className="text-[11px] text-slate-400 font-medium mt-1">
                  Tier 1 Chauffeur Target
                </span>
              </div>

              <div className="p-5 bg-white rounded-2xl border border-[#DCD9D9] shadow-xs">
                <div className="flex items-center justify-between text-slate-400 mb-2">
                  <span className="text-xs font-bold uppercase tracking-wider text-slate-500">
                    Completed Trips
                  </span>
                  <div className="w-8 h-8 rounded-lg bg-purple-50 text-purple-600 flex items-center justify-center">
                    <CheckCircle2 className="w-4 h-4" />
                  </div>
                </div>
                <div className="text-2xl font-extrabold text-[#1F1F1F]">
                  {completedTrips} Trips
                </div>
                <span className="text-[11px] text-slate-400 font-medium mt-1">Avg Rating: 5.0★</span>
              </div>
            </div>

            {/* Main Interactive Map View */}
            <div className="flex-1 rounded-[24px] border border-[#DCD9D9] overflow-hidden relative shadow-lg">
              <MapboxView
                center={[driverPos.lng, driverPos.lat]}
                zoom={14.5}
                pickup={
                  activeTrip
                    ? {
                        lat: activeTrip.pickupLat,
                        lng: activeTrip.pickupLng,
                        label: activeTrip.pickupAddress,
                        type: 'pickup',
                      }
                    : null
                }
                dropoff={
                  activeTrip
                    ? {
                        lat: activeTrip.dropoffLat,
                        lng: activeTrip.dropoffLng,
                        label: activeTrip.dropoffAddress,
                        type: 'dropoff',
                      }
                    : null
                }
                drivers={[driverPos]}
                activeLeg={activeLeg}
                className="w-full h-full"
              />

              {/* ACTIVE TRIP FLOATING HUD ON MAP */}
              {activeTrip && (
                <div className="absolute top-4 left-4 right-4 z-20 p-5 bg-white/95 backdrop-blur-md rounded-2xl border-2 border-[#276EF1] shadow-2xl space-y-4">
                  {/* Top Header of HUD */}
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl bg-[#276EF1] text-white flex items-center justify-center font-bold">
                        <Navigation className="w-5 h-5" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <h4 className="text-sm font-extrabold text-[#1F1F1F]">
                            {tripStage === 'ACCEPTED_EN_ROUTE_PICKUP' && 'En Route to Rider Pickup'}
                            {tripStage === 'ARRIVED_AT_PICKUP' && 'Arrived at Pickup (<= 20m)'}
                            {tripStage === 'IN_TRANSIT' && 'Driving to Final Destination'}
                            {tripStage === 'ARRIVED_AT_DESTINATION' && 'Arrived at Destination (<= 20m)'}
                          </h4>
                          <span className="px-2 py-0.5 rounded bg-blue-100 text-[#276EF1] font-bold text-[10px] uppercase">
                            ${activeTrip.fareAmount.toFixed(2)} Guaranteed
                          </span>
                        </div>
                        <p className="text-xs text-slate-600 mt-0.5">
                          {tripStage === 'ACCEPTED_EN_ROUTE_PICKUP' || tripStage === 'ARRIVED_AT_PICKUP'
                            ? `Pickup: ${activeTrip.pickupAddress}`
                            : `Dropoff: ${activeTrip.dropoffAddress}`}
                        </p>
                      </div>
                    </div>

                    {/* Proximity Distance Ticker */}
                    <div className="text-right">
                      <span className="text-[10px] uppercase font-bold text-slate-400 block">
                        Distance Remaining
                      </span>
                      <span className="text-base font-black text-[#276EF1] font-mono">
                        {tripStage === 'ACCEPTED_EN_ROUTE_PICKUP' && `${distanceToPickup || 180}m`}
                        {tripStage === 'ARRIVED_AT_PICKUP' && '0m (Arrived)'}
                        {tripStage === 'IN_TRANSIT' && `${distanceToDropoff || 420}m`}
                        {tripStage === 'ARRIVED_AT_DESTINATION' && '0m (At Dropoff)'}
                      </span>
                    </div>
                  </div>

                  {/* STAGE 1: DRIVING TO PICKUP (>20m) */}
                  {tripStage === 'ACCEPTED_EN_ROUTE_PICKUP' && (
                    <div className="pt-2 border-t border-slate-100 flex items-center justify-between">
                      <div className="flex items-center gap-2 text-xs text-slate-600 font-medium">
                        <ShieldAlert className="w-4 h-4 text-amber-500" />
                        <span>Ride start is locked until you are within 20m of pickup point.</span>
                      </div>
                      <button
                        onClick={handleSimulateArrivePickup}
                        className="px-3.5 py-1.5 bg-amber-50 hover:bg-amber-100 text-amber-900 border border-amber-300 font-bold text-xs rounded-xl flex items-center gap-1.5 transition-all shadow-xs"
                      >
                        <Zap className="w-3.5 h-3.5 fill-amber-500 text-amber-600" />
                        Simulate Arrive at Pickup (≤20m)
                      </button>
                    </div>
                  )}

                  {/* STAGE 2: ARRIVED AT PICKUP (<= 20m) -> 4-DIGIT OTP VERIFICATION */}
                  {tripStage === 'ARRIVED_AT_PICKUP' && (
                    <div className="pt-3 border-t border-slate-100 bg-[#FCF9F8] p-4 rounded-xl space-y-3">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <KeyRound className="w-4 h-4 text-[#276EF1]" />
                          <span className="text-xs font-bold text-[#1F1F1F]">
                            Ask rider for their 4-Digit Security PIN to start trip:
                          </span>
                        </div>
                        {otpError && (
                          <span className="text-[11px] font-bold text-red-600 animate-pulse">
                            Incorrect PIN. Please re-enter.
                          </span>
                        )}
                      </div>

                      {/* 4-Box PIN Input */}
                      <div className="flex items-center gap-3">
                        <div className="flex gap-2">
                          {enteredOtp.map((digit, idx) => (
                            <input
                              key={idx}
                              ref={(el) => {
                                otpInputRefs.current[idx] = el;
                              }}
                              type="text"
                              maxLength={1}
                              value={digit}
                              onChange={(e) => handleOtpChange(idx, e.target.value)}
                              onKeyDown={(e) => handleOtpKeyDown(idx, e)}
                              className={`w-12 h-12 text-center font-mono text-xl font-extrabold rounded-xl border-2 transition-all focus:outline-none ${
                                otpError
                                    ? 'border-red-500 bg-red-50 text-red-700'
                                  : 'border-[#276EF1] bg-white text-[#1F1F1F] focus:ring-4 focus:ring-blue-100'
                              }`}
                            />
                          ))}
                        </div>

                        <button
                          onClick={handleVerifyOtpAndStart}
                          disabled={enteredOtp.some((d) => !d)}
                          className={`flex-1 py-3 px-4 font-bold text-xs rounded-xl flex items-center justify-center gap-2 transition-all shadow-md active:scale-95 ${
                            enteredOtp.every((d) => d)
                              ? 'bg-[#276EF1] hover:bg-[#1A54C9] text-white shadow-blue-500/25 cursor-pointer'
                              : 'bg-slate-200 text-slate-400 cursor-not-allowed'
                          }`}
                        >
                          <Check className="w-4 h-4" />
                          Verify OTP & Start Ride
                        </button>
                      </div>
                    </div>
                  )}

                  {/* STAGE 3: IN TRANSIT TO DESTINATION */}
                  {tripStage === 'IN_TRANSIT' && (
                    <div className="pt-2 border-t border-slate-100 flex items-center justify-between">
                      <div className="flex items-center gap-2 text-xs text-slate-600 font-medium">
                        <CheckCircle2 className="w-4 h-4 text-emerald-600" />
                        <span>OTP Verified • Driving to {activeTrip.dropoffAddress}</span>
                      </div>
                      <button
                        onClick={handleSimulateArriveDropoff}
                        className="px-3.5 py-1.5 bg-blue-50 hover:bg-blue-100 text-[#276EF1] border border-blue-200 font-bold text-xs rounded-xl flex items-center gap-1.5 transition-all shadow-xs"
                      >
                        <Zap className="w-3.5 h-3.5 fill-[#276EF1]" />
                        Simulate Arrive at Dropoff (≤20m)
                      </button>
                    </div>
                  )}

                  {/* STAGE 4: ARRIVED AT DESTINATION (<= 20m) */}
                  {tripStage === 'ARRIVED_AT_DESTINATION' && (
                    <div className="pt-2 border-t border-slate-100 flex items-center justify-between">
                      <div className="flex items-center gap-2 text-xs font-bold text-emerald-700">
                        <CheckCircle2 className="w-4 h-4" />
                        <span>Within 20m of Destination! Ready to complete trip.</span>
                      </div>
                      <button
                        onClick={handleCompleteTrip}
                        className="px-6 py-2.5 bg-[#008A5E] hover:bg-emerald-700 text-white font-extrabold text-xs rounded-xl shadow-lg shadow-emerald-500/25 transition-all active:scale-95 flex items-center gap-2"
                      >
                        <CheckCircle2 className="w-4 h-4" />
                        Complete Trip & Capture ${activeTrip.fareAmount.toFixed(2)}
                      </button>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Right Column: Recent Trips (4 cols) */}
          <div className="col-span-4 flex flex-col gap-6 h-full">
            <div className="flex-1 bg-white rounded-2xl border border-[#DCD9D9] p-5 flex flex-col shadow-xs overflow-hidden">
              <h3 className="text-sm font-extrabold text-[#1F1F1F] mb-4">Recent Shift Log</h3>

              <div className="flex-1 overflow-y-auto space-y-3">
                {[
                  { time: '14:20', dest: 'Grand Central Terminal', fare: '$48.50', miles: '3.2 mi', rating: 5 },
                  { time: '13:45', dest: 'SoHo Broadway', fare: '$32.00', miles: '2.1 mi', rating: 5 },
                  { time: '12:10', dest: 'JFK Airport Terminal 4', fare: '$84.00', miles: '16.4 mi', rating: 5 },
                  { time: '11:05', dest: 'Wall Street District', fare: '$28.50', miles: '1.8 mi', rating: 5 },
                ].map((item, idx) => (
                  <div
                    key={idx}
                    className="p-3.5 rounded-xl bg-[#FCF9F8] border border-[#DCD9D9] text-xs flex justify-between items-center"
                  >
                    <div>
                      <div className="font-bold text-[#1F1F1F] truncate max-w-[160px]">{item.dest}</div>
                      <div className="text-[11px] text-slate-400 mt-0.5">
                        {item.time} • {item.miles}
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="font-extrabold text-[#276EF1]">{item.fare}</div>
                      <div className="text-[10px] text-amber-600 font-bold">★★★★★</div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* 3. HIGH URGENCY 15-SECOND DISPATCH MODAL OVERLAY */}
        {activeOffer && (
          <div className="fixed inset-0 z-50 bg-slate-950/70 backdrop-blur-md flex items-center justify-center p-6 animate-in fade-in duration-200">
            <div className="w-full max-w-lg bg-white rounded-[28px] border-2 border-[#276EF1] p-8 shadow-2xl relative overflow-hidden">
              <div className="flex items-center justify-between mb-6">
                <div>
                  <span className="px-2.5 py-1 rounded-full bg-blue-100 text-[#276EF1] text-xs font-extrabold uppercase tracking-wider">
                    New Dispatch Request
                  </span>
                  <h2 className="text-2xl font-extrabold text-[#1F1F1F] mt-1">Accept Incoming Chauffeur?</h2>
                </div>

                {/* SVG Countdown Ring */}
                <div className="relative w-20 h-20 flex items-center justify-center">
                  <svg className="w-20 h-20 transform -rotate-90">
                    <circle cx="40" cy="40" r={radius} stroke="#E7F0FF" strokeWidth="6" fill="transparent" />
                    <circle
                      cx="40"
                      cy="40"
                      r={radius}
                      stroke="#276EF1"
                      strokeWidth="6"
                      fill="transparent"
                      strokeDasharray={circumference}
                      strokeDashoffset={strokeDashoffset}
                      strokeLinecap="round"
                      className="transition-all duration-1000 ease-linear"
                    />
                  </svg>
                  <div className="absolute inset-0 flex items-center justify-center font-mono font-black text-xl text-[#276EF1]">
                    {secondsRemaining}s
                  </div>
                </div>
              </div>

              {/* Ride Details Card */}
              <div className="p-5 rounded-2xl bg-[#FCF9F8] border border-[#DCD9D9] mb-6 space-y-3.5">
                <div className="flex items-center justify-between border-b border-[#DCD9D9] pb-3">
                  <div>
                    <span className="text-xs text-slate-500">Rider</span>
                    <h4 className="text-base font-bold text-[#1F1F1F]">{activeOffer.riderName}</h4>
                  </div>
                  <div className="text-right">
                    <span className="text-xs text-slate-500">Guaranteed Fare</span>
                    <div className="text-2xl font-black text-[#276EF1]">${activeOffer.fareAmount.toFixed(2)}</div>
                  </div>
                </div>

                {/* Route Visualizer */}
                <div className="space-y-2 text-xs">
                  <div className="flex items-center gap-2 text-slate-800 font-semibold">
                    <MapPin className="w-4 h-4 text-[#276EF1] flex-shrink-0" />
                    <span className="truncate">{activeOffer.pickupAddress}</span>
                  </div>
                  <div className="flex items-center gap-2 text-slate-800 font-semibold">
                    <Navigation className="w-4 h-4 text-slate-900 flex-shrink-0" />
                    <span className="truncate">{activeOffer.dropoffAddress}</span>
                  </div>
                </div>
              </div>

              {/* Action Buttons */}
              <div className="grid grid-cols-2 gap-4">
                <button
                  onClick={handleDeclineOffer}
                  className="py-4 bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold text-sm rounded-xl transition-all flex items-center justify-center gap-2 active:scale-95"
                >
                  <XCircle className="w-4 h-4" />
                  Decline
                </button>
                <button
                  onClick={handleAcceptOffer}
                  className="py-4 bg-[#008A5E] hover:bg-emerald-700 text-white font-extrabold text-sm rounded-xl transition-all shadow-lg shadow-emerald-500/25 flex items-center justify-center gap-2 active:scale-95"
                >
                  <CheckCircle2 className="w-4 h-4" />
                  Accept Dispatch ({secondsRemaining}s)
                </button>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
