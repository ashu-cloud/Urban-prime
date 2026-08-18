'use client';

import React, { useState, useEffect } from 'react';
import Navbar from '@/components/layout/Navbar';
import MapboxView, { MarkerLocation } from '@/components/map/MapboxView';
import { api, getStoredSession, TripResponse } from '@/lib/api';
import { realtimeBus, DriverLocationEvent, TripStatusEvent } from '@/lib/socket';
import {
  MapPin,
  Navigation,
  Car,
  Zap,
  Shield,
  Clock,
  Star,
  Phone,
  MessageSquare,
  CreditCard,
  CheckCircle,
  Loader2,
  X,
  Sparkles,
} from 'lucide-react';

interface VehicleTier {
  id: 'SEDAN' | 'SUV' | 'PREMIUM' | 'BIKE';
  name: string;
  subtitle: string;
  eta: string;
  price: number;
  capacity: number;
  icon: string;
  popular?: boolean;
}

const VEHICLE_TIERS: VehicleTier[] = [
  {
    id: 'PREMIUM',
    name: 'Prime Black',
    subtitle: 'Luxury Electric Sedan',
    eta: '2 min',
    price: 48.50,
    capacity: 4,
    icon: '⚡',
    popular: true,
  },
  {
    id: 'SEDAN',
    name: 'Urban Comfort',
    subtitle: 'Standard 4-Door Hybrid',
    eta: '4 min',
    price: 32.00,
    capacity: 4,
    icon: '🚘',
  },
  {
    id: 'SUV',
    name: 'Executive SUV',
    subtitle: 'Spacious 6-Passenger SUV',
    eta: '6 min',
    price: 64.00,
    capacity: 6,
    icon: '🚙',
  },
  {
    id: 'BIKE',
    name: 'Prime Express',
    subtitle: 'Rapid Solo Courier',
    eta: '2 min',
    price: 18.00,
    capacity: 1,
    icon: '🛵',
  },
];

export default function RiderPage() {
  const session = getStoredSession();
  const [pickupAddress, setPickupAddress] = useState('Empire State Building, NYC');
  const [dropoffAddress, setDropoffAddress] = useState('Grand Central Terminal, NYC');
  const [selectedTier, setSelectedTier] = useState<VehicleTier['id']>('PREMIUM');
  
  // Coordinates
  const [pickupCoords, setPickupCoords] = useState<MarkerLocation>({
    lat: 40.7484,
    lng: -73.9857,
    label: 'Pickup: Empire State',
    type: 'pickup',
  });

  const [dropoffCoords, setDropoffCoords] = useState<MarkerLocation>({
    lat: 40.7527,
    lng: -73.9772,
    label: 'Dropoff: Grand Central',
    type: 'dropoff',
  });

  // Fleet of live simulated drivers nearby
  const [nearbyDrivers, setNearbyDrivers] = useState<MarkerLocation[]>([
    { id: 'drv_1', lat: 40.7495, lng: -73.9880, heading: 45, label: 'Tesla Model 3' },
    { id: 'drv_2', lat: 40.7460, lng: -73.9820, heading: 120, label: 'Mercedes EQE' },
    { id: 'drv_3', lat: 40.7510, lng: -73.9810, heading: 270, label: 'BMW i4' },
  ]);

  // Trip Lifecycle State
  const [tripState, setTripState] = useState<'IDLE' | 'MATCHING' | 'ACCEPTED' | 'IN_TRANSIT' | 'COMPLETED'>('IDLE');
  const [currentTrip, setCurrentTrip] = useState<TripResponse | null>(null);
  const [assignedDriver, setAssignedDriver] = useState<TripStatusEvent | null>(null);
  const [searchSeconds, setSearchSeconds] = useState(0);

  // Subscribe to real-time Driver location telemetry and Trip status events
  useEffect(() => {
    // 1. Listen for Driver location broadcasts
    const unsubscribeDriverLoc = realtimeBus.onDriverLocation((loc: DriverLocationEvent) => {
      setNearbyDrivers((prev) => {
        const filtered = prev.filter((d) => d.id !== loc.driverId);
        if (!loc.isAvailable && tripState === 'IDLE') return filtered;
        return [
          ...filtered,
          {
            id: loc.driverId,
            lat: loc.latitude,
            lng: loc.longitude,
            heading: loc.heading,
            label: loc.driverName || 'Driver Partner',
          },
        ];
      });
    });

    // 2. Listen for Trip updates from Saga / Driver acceptances
    const unsubscribeTrip = realtimeBus.onTripStatus(currentTrip?.tripId || '', (event: TripStatusEvent) => {
      if (event.status === 'ACCEPTED' || event.status === 'DRIVER_ARRIVING') {
        setTripState('ACCEPTED');
        setAssignedDriver(event);
      } else if (event.status === 'IN_TRANSIT') {
        setTripState('IN_TRANSIT');
      } else if (event.status === 'COMPLETED') {
        setTripState('COMPLETED');
      }
    });

    return () => {
      unsubscribeDriverLoc();
      unsubscribeTrip();
    };
  }, [currentTrip, tripState]);

  // Sonar timer during MATCHING state
  useEffect(() => {
    let timer: any;
    if (tripState === 'MATCHING') {
      timer = setInterval(() => setSearchSeconds((s) => s + 1), 1000);
    } else {
      setSearchSeconds(0);
    }
    return () => clearInterval(timer);
  }, [tripState]);

  // Request Trip Trigger
  const handleRequestRide = async () => {
    const activeTier = VEHICLE_TIERS.find((t) => t.id === selectedTier) || VEHICLE_TIERS[0];
    setTripState('MATCHING');

    try {
      const resp = await api.createTrip({
        riderId: session?.userId || 'rid_001',
        pickupAddress,
        pickupLat: pickupCoords.lat,
        pickupLng: pickupCoords.lng,
        dropoffAddress,
        dropoffLat: dropoffCoords.lat,
        dropoffLng: dropoffCoords.lng,
        vehicleType: selectedTier,
        fareAmount: activeTier.price,
      });
      setCurrentTrip(resp);

      // Broadcast dispatch offer over real-time bus so the Driver Partner screen gets notified
      realtimeBus.publishDispatchOffer({
        tripId: resp.tripId,
        riderId: session?.userId || 'rid_001',
        riderName: session?.name || 'Alexander Vance',
        pickupAddress,
        dropoffAddress,
        pickupLat: pickupCoords.lat,
        pickupLng: pickupCoords.lng,
        dropoffLat: dropoffCoords.lat,
        dropoffLng: dropoffCoords.lng,
        fareAmount: activeTier.price,
        expiresInSeconds: 15,
      });
    } catch (err) {
      console.error(err);
    }
  };

  const handleCancelTrip = () => {
    setTripState('IDLE');
    setCurrentTrip(null);
    setAssignedDriver(null);
  };

  const activeTierObj = VEHICLE_TIERS.find((t) => t.id === selectedTier)!;

  return (
    <div className="h-screen w-screen flex flex-col overflow-hidden bg-[#FCF9F8]">
      {/* Top Bar */}
      <Navbar activeTab="ride" />

      {/* Main Map-Centric Workspace */}
      <div className="relative flex-1 w-full h-[calc(100vh-72px)] overflow-hidden">
        {/* Full-Bleed Mapbox View */}
        <MapboxView
          pickup={pickupCoords}
          dropoff={dropoffCoords}
          drivers={nearbyDrivers}
          className="absolute inset-0 w-full h-full z-0"
        />

        {/* Floating / Docked Rider Control Panel (400px) */}
        <div className="absolute top-6 left-8 z-20 w-[420px] max-h-[calc(100vh-120px)] flex flex-col bg-white rounded-[20px] border border-[#DCD9D9] shadow-2xl overflow-hidden backdrop-blur-md">
          {/* Panel Header */}
          <div className="p-5 border-b border-[#DCD9D9] bg-[#FCF9F8]/80 flex items-center justify-between">
            <div>
              <h2 className="text-base font-extrabold text-[#1F1F1F] tracking-tight flex items-center gap-2">
                <span>Where to next?</span>
                <span className="px-2 py-0.5 rounded-full bg-[#E7F0FF] text-[#276EF1] text-[10px] uppercase font-bold tracking-wider">
                  Live Dispatch
                </span>
              </h2>
              <p className="text-xs text-slate-500 mt-0.5">Book certified chauffeur or executive fleet</p>
            </div>
            {tripState !== 'IDLE' && (
              <button
                onClick={handleCancelTrip}
                className="p-1.5 rounded-lg text-slate-400 hover:text-red-500 hover:bg-red-50 transition-colors"
                title="Cancel Request"
              >
                <X className="w-5 h-5" />
              </button>
            )}
          </div>

          <div className="p-5 overflow-y-auto space-y-5">
            {/* 1. IDLE STATE: Location Inputs & Fleet Selection */}
            {tripState === 'IDLE' && (
              <>
                {/* Location Input Group */}
                <div className="space-y-2.5 relative">
                  {/* Connecting Route Line */}
                  <div className="absolute left-[19px] top-[24px] bottom-[24px] w-0.5 bg-gradient-to-b from-[#276EF1] to-slate-800 z-0"></div>

                  {/* Pickup */}
                  <div className="relative z-10 flex items-center bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl px-3.5 py-2.5 focus-within:border-[#276EF1] focus-within:ring-2 focus-within:ring-blue-100 transition-all">
                    <div className="w-3 h-3 rounded-full bg-[#276EF1] mr-3 ring-4 ring-blue-100"></div>
                    <input
                      type="text"
                      value={pickupAddress}
                      onChange={(e) => setPickupAddress(e.target.value)}
                      placeholder="Pickup point"
                      className="w-full text-xs font-semibold text-[#1F1F1F] bg-transparent focus:outline-none truncate"
                    />
                  </div>

                  {/* Dropoff */}
                  <div className="relative z-10 flex items-center bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl px-3.5 py-2.5 focus-within:border-[#276EF1] focus-within:ring-2 focus-within:ring-blue-100 transition-all">
                    <div className="w-3 h-3 rounded-full bg-slate-900 mr-3 ring-4 ring-slate-200"></div>
                    <input
                      type="text"
                      value={dropoffAddress}
                      onChange={(e) => setDropoffAddress(e.target.value)}
                      placeholder="Destination"
                      className="w-full text-xs font-semibold text-[#1F1F1F] bg-transparent focus:outline-none truncate"
                    />
                  </div>
                </div>

                {/* Fleet Selection Section */}
                <div>
                  <div className="flex items-center justify-between mb-2.5">
                    <label className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                      Select Vehicle Tier
                    </label>
                    <span className="text-xs text-[#276EF1] font-semibold flex items-center gap-1">
                      <Sparkles className="w-3 h-3" /> Premium Fleet
                    </span>
                  </div>

                  <div className="space-y-2">
                    {VEHICLE_TIERS.map((tier) => {
                      const isSelected = selectedTier === tier.id;
                      return (
                        <div
                          key={tier.id}
                          onClick={() => setSelectedTier(tier.id)}
                          className={`p-3.5 rounded-2xl cursor-pointer transition-all border flex items-center justify-between group ${
                            isSelected
                              ? 'bg-[#E7F0FF] border-[#276EF1] shadow-xs'
                              : 'bg-white border-[#DCD9D9] hover:border-slate-400'
                          }`}
                        >
                          <div className="flex items-center gap-3.5">
                            <div
                              className={`w-12 h-12 rounded-xl flex items-center justify-center text-xl shadow-xs transition-colors ${
                                isSelected ? 'bg-white text-[#276EF1]' : 'bg-[#FCF9F8] text-slate-700'
                              }`}
                            >
                              {tier.icon}
                            </div>
                            <div>
                              <div className="flex items-center gap-2">
                                <h4 className="text-sm font-bold text-[#1F1F1F]">{tier.name}</h4>
                                {tier.popular && (
                                  <span className="px-1.5 py-0.5 rounded bg-emerald-100 text-emerald-800 text-[9px] font-extrabold uppercase">
                                    Top Pick
                                  </span>
                                )}
                              </div>
                              <p className="text-[11px] text-slate-500">{tier.subtitle}</p>
                              <p className="text-[10px] font-semibold text-[#276EF1] mt-0.5 flex items-center gap-1">
                                <Clock className="w-2.5 h-2.5" /> {tier.eta} arrival
                              </p>
                            </div>
                          </div>

                          <div className="text-right">
                            <span className="text-base font-extrabold text-[#1F1F1F]">
                              ${tier.price.toFixed(2)}
                            </span>
                            <span className="block text-[10px] text-slate-400 font-medium">Est. Total</span>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>

                {/* Payment Breakdown Card */}
                <div className="p-3.5 rounded-xl bg-[#FCF9F8] border border-[#DCD9D9] flex items-center justify-between text-xs">
                  <div className="flex items-center gap-2.5">
                    <div className="w-7 h-7 rounded-lg bg-slate-900 text-white flex items-center justify-center text-xs font-bold">
                      <CreditCard className="w-3.5 h-3.5" />
                    </div>
                    <div>
                      <span className="font-bold text-[#1F1F1F]">Stripe Auth Hold</span>
                      <span className="block text-[10px] text-slate-400">•••• 4242 • Apple Pay</span>
                    </div>
                  </div>
                  <span className="text-xs font-bold text-emerald-600">Pre-authorized</span>
                </div>

                {/* Request CTA Button */}
                <button
                  onClick={handleRequestRide}
                  className="w-full py-4 bg-[#276EF1] hover:bg-[#1A54C9] text-white font-extrabold text-sm rounded-xl transition-all shadow-lg shadow-blue-500/25 active:scale-95 flex items-center justify-center gap-2"
                >
                  <Zap className="w-4 h-4 fill-white" />
                  <span>Request {activeTierObj.name} (${activeTierObj.price.toFixed(2)})</span>
                </button>
              </>
            )}

            {/* 2. MATCHING STATE: Radial Sonar & Dispatching */}
            {tripState === 'MATCHING' && (
              <div className="text-center py-8 space-y-6">
                <div className="relative w-28 h-28 mx-auto flex items-center justify-center">
                  <div className="absolute inset-0 rounded-full bg-blue-500/20 animate-ping"></div>
                  <div className="absolute -inset-4 rounded-full bg-blue-500/10 animate-pulse"></div>
                  <div className="w-20 h-20 rounded-full bg-[#276EF1] text-white flex items-center justify-center shadow-xl z-10">
                    <Car className="w-10 h-10 animate-bounce" />
                  </div>
                </div>

                <div>
                  <h3 className="text-lg font-bold text-[#1F1F1F]">Connecting to Nearest Driver...</h3>
                  <p className="text-xs text-slate-500 mt-1">
                    Searching 5km radius via Redis GEOSEARCH & APISIX
                  </p>
                  <div className="inline-flex items-center gap-2 mt-3 px-3 py-1 rounded-full bg-blue-50 text-[#276EF1] text-xs font-bold font-mono">
                    <Clock className="w-3.5 h-3.5" />
                    <span>00:{searchSeconds < 10 ? `0${searchSeconds}` : searchSeconds}</span>
                  </div>
                </div>

                <div className="p-3 bg-slate-50 rounded-xl border border-slate-200 text-left text-xs space-y-1">
                  <div className="flex justify-between text-slate-500">
                    <span>Trip Saga ID:</span>
                    <span className="font-mono text-slate-800">{currentTrip?.tripId || 'trip_pending'}</span>
                  </div>
                  <div className="flex justify-between text-slate-500">
                    <span>Payment Hold:</span>
                    <span className="font-bold text-emerald-600">Held (${activeTierObj.price.toFixed(2)})</span>
                  </div>
                </div>

                <button
                  onClick={handleCancelTrip}
                  className="w-full py-3 bg-slate-100 hover:bg-red-50 hover:text-red-600 text-slate-700 font-bold text-xs rounded-xl transition-colors border border-slate-200"
                >
                  Cancel Ride Request
                </button>
              </div>
            )}

            {/* 3. ACCEPTED / DRIVER ARRIVING STATE */}
            {(tripState === 'ACCEPTED' || tripState === 'IN_TRANSIT') && (
              <div className="space-y-5">
                {/* Driver Identity Card */}
                <div className="p-4 rounded-2xl bg-[#E7F0FF] border-2 border-[#276EF1] flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-full bg-[#276EF1] text-white font-bold flex items-center justify-center text-base shadow-sm">
                      {assignedDriver?.driverName ? assignedDriver.driverName[0] : 'M'}
                    </div>
                    <div>
                      <h4 className="text-sm font-extrabold text-[#1F1F1F]">
                        {assignedDriver?.driverName || 'Marcus Sterling'}
                      </h4>
                      <p className="text-xs text-slate-600 font-medium">
                        {assignedDriver?.vehicleModel || 'Tesla Model S (Obsidian Black)'}
                      </p>
                      <div className="flex items-center gap-2 mt-1">
                        <span className="flex items-center text-[11px] font-bold text-amber-600">
                          <Star className="w-3 h-3 fill-amber-500 mr-1" />
                          {assignedDriver?.driverRating || 4.98}
                        </span>
                        <span className="text-[10px] px-1.5 py-0.5 rounded bg-white font-mono font-bold text-slate-800 border border-slate-200">
                          {assignedDriver?.licensePlate || 'NY-7890'}
                        </span>
                      </div>
                    </div>
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <button
                      onClick={() => alert('Connecting securely to chauffeur...')}
                      className="p-2.5 rounded-full bg-white text-[#276EF1] hover:bg-blue-50 transition-colors shadow-xs"
                      title="Call Driver"
                    >
                      <Phone className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => alert('Direct messaging chauffeur...')}
                      className="p-2.5 rounded-full bg-white text-slate-700 hover:bg-slate-50 transition-colors shadow-xs"
                      title="Message Driver"
                    >
                      <MessageSquare className="w-4 h-4" />
                    </button>
                  </div>
                </div>

                {/* Live ETA Card */}
                <div className="p-4 rounded-xl bg-[#FCF9F8] border border-[#DCD9D9] flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-emerald-100 text-emerald-800 flex items-center justify-center font-bold">
                      <Clock className="w-4 h-4" />
                    </div>
                    <div>
                      <span className="text-xs font-bold text-[#1F1F1F]">
                        {tripState === 'ACCEPTED' ? 'Chauffeur En Route' : 'Trip In Progress'}
                      </span>
                      <span className="block text-[11px] text-slate-500">Live GPS Heading Synced</span>
                    </div>
                  </div>
                  <span className="text-base font-extrabold text-[#276EF1]">3 Mins</span>
                </div>

                {/* Safety & Pin Info */}
                <div className="p-3 bg-slate-50 rounded-xl border border-slate-200 flex items-center justify-between text-xs">
                  <div className="flex items-center gap-2 text-slate-600">
                    <Shield className="w-4 h-4 text-emerald-600" />
                    <span>Ride PIN Verification:</span>
                  </div>
                  <span className="font-mono font-extrabold text-sm text-slate-900 tracking-wider bg-white px-2.5 py-1 rounded border">
                    8421
                  </span>
                </div>
              </div>
            )}

            {/* 4. COMPLETED STATE */}
            {tripState === 'COMPLETED' && (
              <div className="text-center py-6 space-y-4">
                <div className="w-14 h-14 mx-auto rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center shadow-md">
                  <CheckCircle className="w-8 h-8" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-[#1F1F1F]">You have arrived!</h3>
                  <p className="text-xs text-slate-500 mt-0.5">Payment automatically captured via Stripe</p>
                </div>
                <div className="p-4 bg-[#FCF9F8] rounded-xl border border-[#DCD9D9] text-left text-xs space-y-1.5">
                  <div className="flex justify-between font-bold text-[#1F1F1F]">
                    <span>Total Fare Charged:</span>
                    <span>${activeTierObj.price.toFixed(2)}</span>
                  </div>
                  <div className="flex justify-between text-slate-500">
                    <span>Payment Intent:</span>
                    <span className="font-mono text-emerald-600 font-bold">CAPTURED (pi_live_...)</span>
                  </div>
                </div>
                <button
                  onClick={handleCancelTrip}
                  className="w-full py-3.5 bg-[#276EF1] text-white font-bold text-sm rounded-xl transition-all shadow-md"
                >
                  Book Another Ride
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
