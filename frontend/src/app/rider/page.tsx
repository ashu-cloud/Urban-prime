'use client';

import React, { useState, useEffect } from 'react';
import Navbar from '@/components/layout/Navbar';
import MapboxView, { MarkerLocation, RouteLegType } from '@/components/map/MapboxView';
import { api, getStoredSession, TripResponse } from '@/lib/api';
import { fetchMapboxDirections } from '@/lib/directions';
import { tripStore, PersistedTripState } from '@/lib/tripStore';
import {
  realtimeBus,
  DriverLocationEvent,
  TripStatusEvent,
  TripLifecycleStage,
  getDistanceInMeters,
} from '@/lib/socket';
import {
  Navigation,
  Car,
  Zap,
  Clock,
  Star,
  Phone,
  MessageSquare,
  CreditCard,
  X,
  Sparkles,
  KeyRound,
  Award,
  Crosshair,
  Route,
} from 'lucide-react';

interface VehicleTier {
  id: 'SEDAN' | 'SUV' | 'PREMIUM' | 'BIKE';
  name: string;
  subtitle: string;
  basePrice: number;
  perKm: number;
  capacity: number;
  icon: string;
  popular?: boolean;
}

const VEHICLE_TIERS: VehicleTier[] = [
  {
    id: 'PREMIUM',
    name: 'Prime Black',
    subtitle: 'Luxury Electric Sedan',
    basePrice: 25.00,
    perKm: 4.50,
    capacity: 4,
    icon: '⚡',
    popular: true,
  },
  {
    id: 'SEDAN',
    name: 'Urban Comfort',
    subtitle: 'Standard 4-Door Hybrid',
    basePrice: 15.00,
    perKm: 3.20,
    capacity: 4,
    icon: '🚘',
  },
  {
    id: 'SUV',
    name: 'Executive SUV',
    subtitle: 'Spacious 6-Passenger SUV',
    basePrice: 35.00,
    perKm: 5.80,
    capacity: 6,
    icon: '🚙',
  },
  {
    id: 'BIKE',
    name: 'Prime Express',
    subtitle: 'Rapid Solo Courier',
    basePrice: 8.00,
    perKm: 2.00,
    capacity: 1,
    icon: '🛵',
  },
];

export default function RiderPage() {
  const session = getStoredSession();
  const [pickupAddress, setPickupAddress] = useState('Empire State Building, NYC');
  const [dropoffAddress, setDropoffAddress] = useState('Grand Central Terminal, NYC');
  const [selectedTier, setSelectedTier] = useState<VehicleTier['id']>('PREMIUM');

  // Interactive Pin Picking Mode: 'PICKUP' | 'DROPOFF' | null
  const [activePinMode, setActivePinMode] = useState<'PICKUP' | 'DROPOFF' | null>(null);

  // Generated Ride PIN / OTP for driver verification
  const [generatedOtp, setGeneratedOtp] = useState('8421');

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

  // Exact Street Driving Distance & Duration from Mapbox Directions API
  const [drivingDistanceKm, setDrivingDistanceKm] = useState<number>(2.4);
  const [drivingDurationText, setDrivingDurationText] = useState<string>('6 min');

  // Fleet of live simulated drivers nearby
  const [nearbyDrivers, setNearbyDrivers] = useState<MarkerLocation[]>([
    { id: 'drv_1', lat: 40.7495, lng: -73.9880, heading: 45, label: 'Tesla Model 3' },
    { id: 'drv_2', lat: 40.7460, lng: -73.9820, heading: 120, label: 'Mercedes EQE' },
    { id: 'drv_3', lat: 40.7510, lng: -73.9810, heading: 270, label: 'BMW i4' },
  ]);

  // Trip Lifecycle State Machine
  const [tripState, setTripState] = useState<TripLifecycleStage>('MATCHING');
  const [isIdle, setIsIdle] = useState(true);
  const [currentTrip, setCurrentTrip] = useState<TripResponse | null>(null);
  const [assignedDriver, setAssignedDriver] = useState<TripStatusEvent | null>(null);
  const [searchSeconds, setSearchSeconds] = useState(0);

  // Rating & Review Modal State
  const [rating, setRating] = useState(5);
  const [selectedTip, setSelectedTip] = useState<number | null>(5);
  const [selectedCompliments, setSelectedCompliments] = useState<string[]>(['Smooth Driving', 'Luxury Vehicle']);
  const [showRatingModal, setShowRatingModal] = useState(false);

  // 1. HYDRATE SINGLE SOURCE OF TRUTH ON MOUNT (SURVIVES REFRESH)
  useEffect(() => {
    const saved = tripStore.get();
    if (saved && saved.status !== 'COMPLETED') {
      setIsIdle(false);
      setTripState(saved.status);
      setPickupAddress(saved.pickupAddress);
      setDropoffAddress(saved.dropoffAddress);
      setPickupCoords({
        lat: saved.pickupLat,
        lng: saved.pickupLng,
        label: `Pickup: ${saved.pickupAddress}`,
        type: 'pickup',
      });
      setDropoffCoords({
        lat: saved.dropoffLat,
        lng: saved.dropoffLng,
        label: `Dropoff: ${saved.dropoffAddress}`,
        type: 'dropoff',
      });
      setGeneratedOtp(saved.otp);
      setCurrentTrip({
        tripId: saved.tripId,
        riderId: saved.riderId,
        status: saved.status,
        fareAmount: saved.fareAmount,
        currency: 'USD',
        pickupLocation: { latitude: saved.pickupLat, longitude: saved.pickupLng, address: saved.pickupAddress },
        dropoffLocation: { latitude: saved.dropoffLat, longitude: saved.dropoffLng, address: saved.dropoffAddress },
        vehicleType: saved.vehicleType,
        createdAt: new Date(saved.createdAt).toISOString(),
      });

      if (saved.driverId) {
        const dLat = saved.driverLat || saved.pickupLat;
        const dLng = saved.driverLng || saved.pickupLng;
        const dObj = {
          tripId: saved.tripId,
          status: saved.status,
          driverId: saved.driverId,
          driverName: saved.driverName || 'Marcus Sterling',
          driverLat: dLat,
          driverLng: dLng,
          driverRating: saved.driverRating || 4.98,
          vehicleModel: saved.vehicleModel || 'Tesla Model S (Obsidian Black)',
          licensePlate: saved.licensePlate || 'NY-7890',
          otp: saved.otp,
        };
        setAssignedDriver(dObj);
        setNearbyDrivers((prev) => [
          {
            id: saved.driverId!,
            lat: dLat,
            lng: dLng,
            heading: saved.driverHeading || 45,
            label: saved.driverName || 'Marcus Sterling (Tesla Model S)',
          },
          ...prev.filter((d) => d.id !== saved.driverId),
        ]);
      }
    }
  }, []);

  // Calculate real road driving distance and duration whenever pickup/dropoff moves
  useEffect(() => {
    let isMounted = true;
    async function loadRoadRoute() {
      const result = await fetchMapboxDirections(
        pickupCoords.lng,
        pickupCoords.lat,
        dropoffCoords.lng,
        dropoffCoords.lat
      );
      if (isMounted && result) {
        setDrivingDistanceKm(result.distanceKm);
        setDrivingDurationText(result.durationFormatted);
      }
    }
    loadRoadRoute();
    return () => {
      isMounted = false;
    };
  }, [pickupCoords.lat, pickupCoords.lng, dropoffCoords.lat, dropoffCoords.lng]);

  // Map Click Handler for Precision Pin Drop
  const handleMapClick = (coords: { lat: number; lng: number }) => {
    if (activePinMode === 'PICKUP') {
      setPickupCoords({
        lat: coords.lat,
        lng: coords.lng,
        label: `Pickup (${coords.lat.toFixed(4)}, ${coords.lng.toFixed(4)})`,
        type: 'pickup',
      });
      setPickupAddress(`Custom Pickup (${coords.lat.toFixed(4)}, ${coords.lng.toFixed(4)})`);
      setActivePinMode(null);
    } else if (activePinMode === 'DROPOFF') {
      setDropoffCoords({
        lat: coords.lat,
        lng: coords.lng,
        label: `Dropoff (${coords.lat.toFixed(4)}, ${coords.lng.toFixed(4)})`,
        type: 'dropoff',
      });
      setDropoffAddress(`Custom Dropoff (${coords.lat.toFixed(4)}, ${coords.lng.toFixed(4)})`);
      setActivePinMode(null);
    }
  };

  // Draggable Marker Handlers
  const handlePickupDrag = (coords: { lat: number; lng: number }) => {
    setPickupCoords({
      lat: coords.lat,
      lng: coords.lng,
      label: `Pickup (${coords.lat.toFixed(4)}, ${coords.lng.toFixed(4)})`,
      type: 'pickup',
    });
    setPickupAddress(`Custom Pickup (${coords.lat.toFixed(4)}, ${coords.lng.toFixed(4)})`);
  };

  const handleDropoffDrag = (coords: { lat: number; lng: number }) => {
    setDropoffCoords({
      lat: coords.lat,
      lng: coords.lng,
      label: `Dropoff (${coords.lat.toFixed(4)}, ${coords.lng.toFixed(4)})`,
      type: 'dropoff',
    });
    setDropoffAddress(`Custom Dropoff (${coords.lat.toFixed(4)}, ${coords.lng.toFixed(4)})`);
  };

  // Subscribe to real-time Driver location telemetry and Trip status events
  useEffect(() => {
    const unsubscribeDriverLoc = realtimeBus.onDriverLocation((loc: DriverLocationEvent) => {
      setNearbyDrivers((prev) => {
        const filtered = prev.filter((d) => d.id !== loc.driverId);
        const updatedDriver = {
          id: loc.driverId,
          lat: loc.latitude,
          lng: loc.longitude,
          heading: loc.heading,
          label: loc.driverName || 'Driver Partner',
        };

        // Update single source of truth store
        const active = tripStore.get();
        if (active && active.driverId === loc.driverId) {
          tripStore.updateStatus(active.status, {
            driverLat: loc.latitude,
            driverLng: loc.longitude,
            driverHeading: loc.heading,
          });
        }

        return [updatedDriver, ...filtered];
      });
    });

    const unsubscribeTrip = realtimeBus.onTripStatus('', (event: TripStatusEvent) => {
      if (
        event.status === 'ACCEPTED_EN_ROUTE_PICKUP' ||
        event.status === 'ARRIVED_AT_PICKUP' ||
        event.status === 'IN_TRANSIT' ||
        event.status === 'ARRIVED_AT_DESTINATION'
      ) {
        setIsIdle(false);
        setTripState(event.status);
        setAssignedDriver((prev) => ({ ...prev, ...event }));

        // Persist to Single Source of Truth
        tripStore.updateStatus(event.status, {
          driverId: event.driverId,
          driverName: event.driverName,
          driverRating: event.driverRating,
          vehicleModel: event.vehicleModel,
          licensePlate: event.licensePlate,
          driverLat: event.driverLat,
          driverLng: event.driverLng,
        });
      } else if (event.status === 'COMPLETED') {
        setIsIdle(false);
        setTripState('COMPLETED');
        setShowRatingModal(true);
        tripStore.updateStatus('COMPLETED');
      }
    });

    return () => {
      unsubscribeDriverLoc();
      unsubscribeTrip();
    };
  }, []);

  // Sonar timer during MATCHING state
  useEffect(() => {
    let timer: any;
    if (!isIdle && tripState === 'MATCHING') {
      timer = setInterval(() => setSearchSeconds((s) => s + 1), 1000);
    } else {
      setSearchSeconds(0);
    }
    return () => clearInterval(timer);
  }, [tripState, isIdle]);

  // Assigned driver resolution for 100% accurate camera follow
  const assignedDriverLocation = assignedDriver
    ? nearbyDrivers.find((d) => d.id === assignedDriver.driverId) || {
        id: assignedDriver.driverId || 'drv_assigned',
        lat: assignedDriver.driverLat || pickupCoords.lat,
        lng: assignedDriver.driverLng || pickupCoords.lng,
        heading: 45,
        label: assignedDriver.driverName || 'Marcus Sterling (Tesla Model S)',
      }
    : null;

  const distanceToRider = assignedDriverLocation
    ? getDistanceInMeters(assignedDriverLocation.lat, assignedDriverLocation.lng, pickupCoords.lat, pickupCoords.lng)
    : 160;

  // Exact Fare Calculation: Base Price + (Per Km Rate * Actual Road Driving Distance)
  const getTierPrice = (tier: VehicleTier) => {
    return Number((tier.basePrice + tier.perKm * drivingDistanceKm).toFixed(2));
  };

  // Request Trip Trigger
  const handleRequestRide = async () => {
    const activeTier = VEHICLE_TIERS.find((t) => t.id === selectedTier) || VEHICLE_TIERS[0];
    const finalFare = getTierPrice(activeTier);
    const otp = Math.floor(1000 + Math.random() * 9000).toString();
    setGeneratedOtp(otp);
    setIsIdle(false);
    setTripState('MATCHING');
    setActivePinMode(null);

    const newTripId = `trip_${Date.now()}`;

    // 1. SAVE PERSISTENT SINGLE SOURCE OF TRUTH
    const initialTripState: PersistedTripState = {
      tripId: newTripId,
      riderId: session?.userId || 'rid_001',
      riderName: session?.name || 'Alexander Vance',
      status: 'MATCHING',
      pickupAddress,
      pickupLat: pickupCoords.lat,
      pickupLng: pickupCoords.lng,
      dropoffAddress,
      dropoffLat: dropoffCoords.lat,
      dropoffLng: dropoffCoords.lng,
      vehicleType: selectedTier,
      fareAmount: finalFare,
      otp: otp,
      createdAt: Date.now(),
    };
    tripStore.save(initialTripState);

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
        fareAmount: finalFare,
      });
      setCurrentTrip(resp);
    } catch {
      setCurrentTrip({
        tripId: newTripId,
        riderId: session?.userId || 'rid_001',
        status: 'MATCHING',
        fareAmount: finalFare,
        currency: 'USD',
        pickupLocation: { latitude: pickupCoords.lat, longitude: pickupCoords.lng, address: pickupAddress },
        dropoffLocation: { latitude: dropoffCoords.lat, longitude: dropoffCoords.lng, address: dropoffAddress },
        vehicleType: selectedTier,
        createdAt: new Date().toISOString(),
      });
    }

    // Broadcast dispatch offer over real-time bus
    realtimeBus.publishDispatchOffer({
      tripId: newTripId,
      riderId: session?.userId || 'rid_001',
      riderName: session?.name || 'Alexander Vance',
      pickupAddress,
      dropoffAddress,
      pickupLat: pickupCoords.lat,
      pickupLng: pickupCoords.lng,
      dropoffLat: dropoffCoords.lat,
      dropoffLng: dropoffCoords.lng,
      fareAmount: finalFare,
      expiresInSeconds: 15,
      otp: otp,
    });
  };

  const handleCancelTrip = () => {
    setIsIdle(true);
    setTripState('MATCHING');
    setCurrentTrip(null);
    setAssignedDriver(null);
    setShowRatingModal(false);
    setActivePinMode(null);
    tripStore.clear();
  };

  const handleSubmitRating = () => {
    alert(`Thank you for rating your chauffeur ${rating}★! Tip: $${selectedTip || 0}`);
    setShowRatingModal(false);
    handleCancelTrip();
  };

  const toggleCompliment = (badge: string) => {
    if (selectedCompliments.includes(badge)) {
      setSelectedCompliments(selectedCompliments.filter((b) => b !== badge));
    } else {
      setSelectedCompliments([...selectedCompliments, badge]);
    }
  };

  const activeTierObj = VEHICLE_TIERS.find((t) => t.id === selectedTier)!;
  const currentFare = getTierPrice(activeTierObj);

  let activeLeg: RouteLegType = 'NONE';
  if (!isIdle) {
    if (tripState === 'ACCEPTED_EN_ROUTE_PICKUP' || tripState === 'ARRIVED_AT_PICKUP') {
      activeLeg = 'TO_PICKUP';
    } else if (tripState === 'IN_TRANSIT' || tripState === 'ARRIVED_AT_DESTINATION') {
      activeLeg = 'TO_DESTINATION';
    }
  }

  // When in an active ride, ONLY show the assigned chauffeur and filter out all other cars
  const renderedDriversList = !isIdle
    ? (assignedDriverLocation ? [assignedDriverLocation] : [])
    : nearbyDrivers;

  return (
    <div className="h-screen w-screen flex flex-col overflow-hidden bg-[#FCF9F8]">
      {/* Top Bar */}
      <Navbar activeTab="ride" />

      {/* Main Map-Centric Workspace */}
      <div className="relative flex-1 w-full h-[calc(100vh-72px)] overflow-hidden">
        {/* Full-Bleed Mapbox View with Real Road Routing and Auto-Vehicle Tracking */}
        <MapboxView
          pickup={pickupCoords}
          dropoff={dropoffCoords}
          drivers={renderedDriversList}
          activeLeg={activeLeg}
          activePinMode={activePinMode}
          onMapClick={handleMapClick}
          onPickupDrag={handlePickupDrag}
          onDropoffDrag={handleDropoffDrag}
          className="absolute inset-0 w-full h-full z-0"
        />

        {/* Floating / Docked Rider Control Panel (430px) */}
        <div className="absolute top-6 left-8 z-20 w-[430px] max-h-[calc(100vh-120px)] flex flex-col bg-white rounded-[24px] border border-[#DCD9D9] shadow-2xl overflow-hidden backdrop-blur-md">
          {/* Panel Header */}
          <div className="p-5 border-b border-[#DCD9D9] bg-[#FCF9F8]/80 flex items-center justify-between">
            <div>
              <h2 className="text-base font-extrabold text-[#1F1F1F] tracking-tight flex items-center gap-2">
                <span>Where to next?</span>
                <span className="px-2 py-0.5 rounded-full bg-[#E7F0FF] text-[#276EF1] text-[10px] uppercase font-bold tracking-wider">
                  Live Dispatch
                </span>
              </h2>
              <p className="text-xs text-slate-500 mt-0.5">Click or drag pins on map to adjust locations</p>
            </div>
            {!isIdle && (
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
            {isIdle && (
              <>
                {/* Location Input Group with PIN ON MAP Triggers */}
                <div className="space-y-2.5 relative">
                  <div className="absolute left-[19px] top-[24px] bottom-[24px] w-0.5 bg-gradient-to-b from-[#276EF1] to-slate-800 z-0"></div>

                  {/* Pickup Input + Pin Button */}
                  <div
                    className={`relative z-10 flex items-center bg-[#FCF9F8] border rounded-xl px-3.5 py-2.5 transition-all ${
                      activePinMode === 'PICKUP'
                        ? 'border-[#276EF1] ring-2 ring-blue-200 bg-blue-50/50'
                        : 'border-[#DCD9D9] focus-within:border-[#276EF1] focus-within:ring-2 focus-within:ring-blue-100'
                    }`}
                  >
                    <div className="w-3 h-3 rounded-full bg-[#276EF1] mr-3 ring-4 ring-blue-100 flex-shrink-0"></div>
                    <input
                      type="text"
                      value={pickupAddress}
                      onChange={(e) => setPickupAddress(e.target.value)}
                      placeholder="Enter pickup address"
                      className="w-full text-xs font-semibold text-[#1F1F1F] bg-transparent focus:outline-none truncate pr-2"
                    />
                    <button
                      type="button"
                      onClick={() => setActivePinMode(activePinMode === 'PICKUP' ? null : 'PICKUP')}
                      title="Set pickup location by pinning on map"
                      className={`px-2.5 py-1 rounded-lg text-[11px] font-bold flex items-center gap-1 transition-all flex-shrink-0 shadow-2xs active:scale-95 ${
                        activePinMode === 'PICKUP'
                          ? 'bg-[#276EF1] text-white shadow-blue-500/25'
                          : 'bg-white text-[#276EF1] border border-[#276EF1]/30 hover:bg-blue-50'
                      }`}
                    >
                      <Crosshair className="w-3 h-3" />
                      <span>{activePinMode === 'PICKUP' ? 'Click Map' : 'Pin Map'}</span>
                    </button>
                  </div>

                  {/* Dropoff Input + Pin Button */}
                  <div
                    className={`relative z-10 flex items-center bg-[#FCF9F8] border rounded-xl px-3.5 py-2.5 transition-all ${
                      activePinMode === 'DROPOFF'
                        ? 'border-slate-900 ring-2 ring-slate-300 bg-slate-100/70'
                        : 'border-[#DCD9D9] focus-within:border-slate-800 focus-within:ring-2 focus-within:ring-slate-200'
                    }`}
                  >
                    <div className="w-3 h-3 rounded-full bg-slate-900 mr-3 ring-4 ring-slate-200 flex-shrink-0"></div>
                    <input
                      type="text"
                      value={dropoffAddress}
                      onChange={(e) => setDropoffAddress(e.target.value)}
                      placeholder="Enter destination"
                      className="w-full text-xs font-semibold text-[#1F1F1F] bg-transparent focus:outline-none truncate pr-2"
                    />
                    <button
                      type="button"
                      onClick={() => setActivePinMode(activePinMode === 'DROPOFF' ? null : 'DROPOFF')}
                      title="Set destination by pinning on map"
                      className={`px-2.5 py-1 rounded-lg text-[11px] font-bold flex items-center gap-1 transition-all flex-shrink-0 shadow-2xs active:scale-95 ${
                        activePinMode === 'DROPOFF'
                          ? 'bg-slate-900 text-white'
                          : 'bg-white text-slate-800 border border-slate-300 hover:bg-slate-100'
                      }`}
                    >
                      <Crosshair className="w-3 h-3" />
                      <span>{activePinMode === 'DROPOFF' ? 'Click Map' : 'Pin Map'}</span>
                    </button>
                  </div>
                </div>

                {/* Real Road Driving Distance & Duration Banner */}
                <div className="flex items-center justify-between px-2 py-2 rounded-xl bg-slate-50 border border-slate-200 text-xs">
                  <div className="flex items-center gap-1.5 font-bold text-slate-700">
                    <Route className="w-3.5 h-3.5 text-[#276EF1]" />
                    <span>Real Driving Route:</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="font-extrabold text-[#276EF1] font-mono">{drivingDistanceKm} km</span>
                    <span className="text-slate-400">•</span>
                    <span className="text-slate-600 font-semibold">{drivingDurationText} drive</span>
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
                      const price = getTierPrice(tier);
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
                                <Clock className="w-2.5 h-2.5" /> {drivingDurationText} arrival
                              </p>
                            </div>
                          </div>

                          <div className="text-right">
                            <span className="text-base font-extrabold text-[#1F1F1F]">
                              ${price.toFixed(2)}
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
                      <span className="font-bold text-[#1F1F1F]">Stripe Pre-Auth Hold</span>
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
                  <span>
                    Request {activeTierObj.name} (${currentFare.toFixed(2)})
                  </span>
                </button>
              </>
            )}

            {/* 2. MATCHING STATE: Radial Sonar & Dispatching */}
            {!isIdle && tripState === 'MATCHING' && (
              <div className="text-center py-8 space-y-6">
                <div className="relative w-28 h-28 mx-auto flex items-center justify-center">
                  <div className="absolute inset-0 rounded-full bg-blue-500/20 animate-ping"></div>
                  <div className="absolute -inset-4 rounded-full bg-blue-500/10 animate-pulse"></div>
                  <div className="w-20 h-20 rounded-full bg-[#276EF1] text-white flex items-center justify-center shadow-xl z-10">
                    <Car className="w-10 h-10 animate-bounce" />
                  </div>
                </div>

                <div>
                  <h3 className="text-lg font-bold text-[#1F1F1F]">Connecting to Nearest Chauffeur...</h3>
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
                    <span className="font-bold text-emerald-600">Held (${currentFare.toFixed(2)})</span>
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

            {/* 3. ACCEPTED / EN ROUTE TO PICKUP / ARRIVED AT PICKUP */}
            {!isIdle && (tripState === 'ACCEPTED_EN_ROUTE_PICKUP' || tripState === 'ARRIVED_AT_PICKUP') && (
              <div className="space-y-4">
                {/* 4-DIGIT RIDE PIN CARD */}
                <div className="p-4 rounded-2xl bg-gradient-to-r from-blue-600 to-[#276EF1] text-white shadow-lg space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-blue-100">
                      <KeyRound className="w-4 h-4" />
                      <span>Your Ride Start PIN</span>
                    </div>
                    <span className="text-[10px] bg-white/20 px-2 py-0.5 rounded-full font-semibold">
                      Required for Chauffeur
                    </span>
                  </div>

                  <div className="flex items-center justify-between pt-1">
                    <span className="font-mono text-3xl font-black tracking-widest bg-white/10 px-4 py-1.5 rounded-xl border border-white/20">
                      {generatedOtp}
                    </span>
                    <p className="text-xs text-blue-100 max-w-[160px] text-right font-medium leading-tight">
                      Share this 4-digit PIN with Marcus once he arrives.
                    </p>
                  </div>
                </div>

                {/* Proximity / Status Banner */}
                <div
                  className={`p-3.5 rounded-xl border flex items-center justify-between text-xs ${
                    tripState === 'ARRIVED_AT_PICKUP'
                      ? 'bg-emerald-50 border-emerald-300 text-emerald-900 font-bold animate-pulse'
                      : 'bg-blue-50 border-blue-200 text-blue-900 font-medium'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <Clock className="w-4 h-4 text-[#276EF1]" />
                    <span>
                      {tripState === 'ARRIVED_AT_PICKUP'
                        ? '⚡ Chauffeur has arrived outside! Share your PIN.'
                        : `Chauffeur is en route (${distanceToRider}m away)`}
                    </span>
                  </div>
                  <span className="font-bold text-[#276EF1]">
                    {tripState === 'ARRIVED_AT_PICKUP' ? 'Here Now' : '2 Mins'}
                  </span>
                </div>

                {/* Driver Identity Card */}
                <div className="p-4 rounded-2xl bg-[#FCF9F8] border border-[#DCD9D9] flex items-center justify-between">
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
              </div>
            )}

            {/* 4. IN TRANSIT TO DESTINATION */}
            {!isIdle && (tripState === 'IN_TRANSIT' || tripState === 'ARRIVED_AT_DESTINATION') && (
              <div className="space-y-4">
                <div className="p-4 rounded-2xl bg-emerald-50 border border-emerald-200 text-emerald-900 flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-9 h-9 rounded-full bg-emerald-600 text-white flex items-center justify-center font-bold">
                      <Navigation className="w-5 h-5" />
                    </div>
                    <div>
                      <h4 className="text-sm font-extrabold">Trip In Progress</h4>
                      <p className="text-xs text-emerald-700">Driving to {dropoffAddress}</p>
                    </div>
                  </div>
                  <span className="text-sm font-black text-emerald-800 font-mono">65 MPH</span>
                </div>

                <div className="p-4 bg-[#FCF9F8] rounded-xl border border-[#DCD9D9] space-y-2 text-xs">
                  <div className="flex justify-between text-slate-600">
                    <span>Estimated Arrival:</span>
                    <span className="font-bold text-[#1F1F1F]">{drivingDurationText}</span>
                  </div>
                  <div className="flex justify-between text-slate-600">
                    <span>Road Distance:</span>
                    <span className="font-bold text-slate-800">{drivingDistanceKm} km</span>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* 5. POST-TRIP LUXURY 5-STAR RATING & REVIEW MODAL */}
        {showRatingModal && (
          <div className="fixed inset-0 z-50 bg-slate-950/70 backdrop-blur-md flex items-center justify-center p-6 animate-in fade-in duration-200">
            <div className="w-full max-w-md bg-white rounded-[28px] border border-[#DCD9D9] p-8 shadow-2xl space-y-6">
              <div className="text-center space-y-1.5">
                <div className="w-16 h-16 mx-auto rounded-full bg-emerald-50 text-emerald-600 flex items-center justify-center mb-2 shadow-inner">
                  <Award className="w-8 h-8" />
                </div>
                <h3 className="text-2xl font-black text-[#1F1F1F] tracking-tight">How was your chauffeur?</h3>
                <p className="text-xs text-slate-500">Rate your experience with Marcus Sterling (Tesla Model S)</p>
              </div>

              {/* 5 Interactive Stars */}
              <div className="flex items-center justify-center gap-2 py-2">
                {[1, 2, 3, 4, 5].map((star) => (
                  <button
                    key={star}
                    type="button"
                    onClick={() => setRating(star)}
                    className="p-1 hover:scale-125 transition-transform"
                  >
                    <Star
                      className={`w-8 h-8 ${
                        star <= rating
                          ? 'fill-amber-400 text-amber-400 drop-shadow-md'
                          : 'text-slate-300 hover:text-amber-200'
                      }`}
                    />
                  </button>
                ))}
              </div>

              {/* Compliment Badges */}
              <div>
                <label className="text-[11px] font-bold uppercase tracking-wider text-slate-400 block mb-2 text-center">
                  Give a Compliment
                </label>
                <div className="flex flex-wrap justify-center gap-2">
                  {[
                    'Smooth Driving',
                    'Luxury Vehicle',
                    'Polite & Professional',
                    'Great Music',
                    'Perfect Route',
                  ].map((badge) => {
                    const isSelected = selectedCompliments.includes(badge);
                    return (
                      <button
                        key={badge}
                        type="button"
                        onClick={() => toggleCompliment(badge)}
                        className={`px-3 py-1.5 rounded-full text-xs font-bold transition-all border ${
                          isSelected
                            ? 'bg-[#E7F0FF] border-[#276EF1] text-[#276EF1]'
                            : 'bg-[#FCF9F8] border-[#DCD9D9] text-slate-600 hover:border-slate-400'
                        }`}
                      >
                        {badge}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Tip Selection */}
              <div>
                <label className="text-[11px] font-bold uppercase tracking-wider text-slate-400 block mb-2 text-center">
                  Add a Tip for Marcus
                </label>
                <div className="grid grid-cols-4 gap-2">
                  {[2, 5, 10, null].map((tipVal, idx) => (
                    <button
                      key={idx}
                      type="button"
                      onClick={() => setSelectedTip(tipVal)}
                      className={`py-2 rounded-xl text-xs font-extrabold border transition-all ${
                        selectedTip === tipVal
                          ? 'bg-[#276EF1] text-white border-[#276EF1] shadow-sm'
                          : 'bg-[#FCF9F8] text-slate-700 border-[#DCD9D9] hover:border-slate-400'
                      }`}
                    >
                      {tipVal !== null ? `$${tipVal}` : 'No Tip'}
                    </button>
                  ))}
                </div>
              </div>

              {/* Submit Review CTA */}
              <button
                onClick={handleSubmitRating}
                className="w-full py-4 bg-[#276EF1] hover:bg-[#1A54C9] text-white font-extrabold text-sm rounded-xl shadow-lg shadow-blue-500/25 transition-all active:scale-95"
              >
                Submit Feedback & Return Home
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
