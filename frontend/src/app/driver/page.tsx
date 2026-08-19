'use client';

import React, { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import MapboxView, { MarkerLocation, RouteLegType } from '@/components/map/MapboxView';
import { getStoredDriverSession, clearStoredDriverSession } from '@/lib/api';
import { tripStore } from '@/lib/tripStore';
import { fetchMapboxDirections, calculateRoadHeading } from '@/lib/directions';
import {
  realtimeBus,
  DispatchOfferEvent,
  TripStatusEvent,
  getDistanceInMeters,
  TripLifecycleStage,
} from '@/lib/socket';
import { calculatePlatformFee } from '@/lib/fare';
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
  LogOut,
  Sparkles,
} from 'lucide-react';

interface ShiftTripLog {
  id: string;
  time: string;
  dest: string;
  fare: string;
  grossFare?: string;
  platformFee?: string;
  tip?: string;
  miles: string;
  rating: number;
}

export default function DriverPage() {
  const router = useRouter();
  const [session, setSession] = useState<any>(null);
  const [isAuthChecking, setIsAuthChecking] = useState(true);
  const [isOnline, setIsOnline] = useState(true);
  const [earnings, setEarnings] = useState<number>(0.0);
  const [acceptanceRate, setAcceptanceRate] = useState<string>('100%');
  const [completedTrips, setCompletedTrips] = useState<number>(0);
  const [shiftLog, setShiftLog] = useState<ShiftTripLog[]>([]);
  const [tipToast, setTipToast] = useState<{ amount: number; riderName: string } | null>(null);

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

  // Road coordinates for exact street navigation
  const [activeRoadCoords, setActiveRoadCoords] = useState<[number, number][]>([]);
  const roadIndexRef = useRef<number>(0);

  // OTP Verification State
  const [enteredOtp, setEnteredOtp] = useState(['', '', '', '']);
  const [otpError, setOtpError] = useState(false);
  const [isOtpVerified, setIsOtpVerified] = useState(false);
  const otpInputRefs = useRef<(HTMLInputElement | null)[]>([]);

  // 0. AUTH GUARD: Verify active Driver session & hydrate driver-specific shift stats
  useEffect(() => {
    const currentSession = getStoredDriverSession();
    if (!currentSession || currentSession.role !== 'DRIVER') {
      router.replace('/driver/login');
      return;
    }
    setSession(currentSession);
    setIsAuthChecking(false);

    // Hydrate driver-specific shift metrics & logs (starts at $0.00 / 0 trips for fresh sessions)
    try {
      const rawStats = localStorage.getItem(`urban_driver_stats_${currentSession.userId}`);
      let loadedEarnings = 0.0;
      let loadedCompletedTrips = 0;
      let loadedAcceptanceRate = '100%';
      let loadedShiftLog: ShiftTripLog[] = [];

      if (rawStats) {
        const parsed = JSON.parse(rawStats);
        loadedEarnings = parsed.earnings || 0.0;
        loadedCompletedTrips = parsed.completedTrips || 0;
        loadedAcceptanceRate = parsed.acceptanceRate || '100%';
        loadedShiftLog = parsed.shiftLog || [];
      }

      // Sync real ratings from all rider activity history in localStorage
      const allRiderTrips: any[] = [];
      for (let i = 0; i < localStorage.length; i++) {
        const k = localStorage.key(i);
        if (k && k.startsWith('urban_rider_history_')) {
          try {
            const arr = JSON.parse(localStorage.getItem(k) || '[]');
            if (Array.isArray(arr)) {
              allRiderTrips.push(...arr);
            }
          } catch {}
        }
      }

      if (allRiderTrips.length > 0 && loadedShiftLog.length > 0) {
        loadedShiftLog = loadedShiftLog.map((logItem, idx) => {
          const match = allRiderTrips.find(
            (rt) =>
              rt.tripId === logItem.id ||
              (rt.tripId && logItem.id && (rt.tripId.includes(logItem.id) || logItem.id.includes(rt.tripId)))
          );
          if (match && match.rating !== undefined) {
            return {
              ...logItem,
              rating: match.rating,
              tip: match.tip > 0 ? `+$${match.tip.toFixed(2)} Tip (100% Driver)` : logItem.tip,
            };
          }
          if (allRiderTrips[idx] && allRiderTrips[idx].rating !== undefined) {
            return {
              ...logItem,
              rating: allRiderTrips[idx].rating,
              tip: allRiderTrips[idx].tip > 0 ? `+$${allRiderTrips[idx].tip.toFixed(2)} Tip (100% Driver)` : logItem.tip,
            };
          }
          return logItem;
        });

        localStorage.setItem(
          `urban_driver_stats_${currentSession.userId}`,
          JSON.stringify({
            earnings: loadedEarnings,
            completedTrips: loadedCompletedTrips,
            acceptanceRate: loadedAcceptanceRate,
            shiftLog: loadedShiftLog,
          })
        );
      }

      setEarnings(loadedEarnings);
      setCompletedTrips(loadedCompletedTrips);
      setAcceptanceRate(loadedAcceptanceRate);
      setShiftLog(loadedShiftLog);
    } catch {
      setEarnings(0.0);
      setCompletedTrips(0);
      setAcceptanceRate('100%');
      setShiftLog([]);
    }
  }, [router]);

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
        platformFee: saved.platformFee,
        driverNetFare: saved.driverNetFare,
        feePercentage: saved.feePercentage,
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

  // Fetch real road polyline whenever trip stage changes to navigation
  useEffect(() => {
    let isMounted = true;
    async function loadLegDirections() {
      if (tripStage === 'ACCEPTED_EN_ROUTE_PICKUP' && activeTrip) {
        const result = await fetchMapboxDirections(
          driverPos.lng,
          driverPos.lat,
          activeTrip.pickupLng,
          activeTrip.pickupLat
        );
        if (isMounted && result) {
          setActiveRoadCoords(result.coordinates);
          roadIndexRef.current = 0;
        }
      } else if (tripStage === 'IN_TRANSIT' && activeTrip) {
        const result = await fetchMapboxDirections(
          activeTrip.pickupLng,
          activeTrip.pickupLat,
          activeTrip.dropoffLng,
          activeTrip.dropoffLat
        );
        if (isMounted && result) {
          setActiveRoadCoords(result.coordinates);
          roadIndexRef.current = 0;
        }
      }
    }

    if (activeTrip) {
      loadLegDirections();
    }

    return () => {
      isMounted = false;
    };
  }, [tripStage, activeTrip?.tripId]);

  // GLIDE PRECISELY ALONG ROAD NETWORK POLYLINE
  useEffect(() => {
    if (!isOnline) return;

    const interval = setInterval(() => {
      setDriverPos((prev) => {
        let newPos = { ...prev };

        if (activeRoadCoords.length > 0 && (tripStage === 'ACCEPTED_EN_ROUTE_PICKUP' || tripStage === 'IN_TRANSIT')) {
          const currentIndex = roadIndexRef.current;
          if (currentIndex < activeRoadCoords.length - 1) {
            const nextIndex = Math.min(currentIndex + 1, activeRoadCoords.length - 1);
            roadIndexRef.current = nextIndex;

            const currCoord = activeRoadCoords[currentIndex];
            const nextCoord = activeRoadCoords[nextIndex];
            const roadHeading = calculateRoadHeading(
              currCoord[1],
              currCoord[0],
              nextCoord[1],
              nextCoord[0]
            );

            newPos = {
              lat: nextCoord[1],
              lng: nextCoord[0],
              heading: roadHeading,
              label: session?.name ? `${session.name} (${session.vehicleModel || 'Tesla Model S'})` : 'Marcus Sterling',
            };
          }
        } else {
          // Idle drift around Midtown Manhattan
          const deltaLat = (Math.random() - 0.5) * 0.0003;
          const deltaLng = (Math.random() - 0.5) * 0.0003;
          const prevHeading = prev.heading ?? 45;
          newPos = {
            ...prev,
            lat: Number((prev.lat + deltaLat).toFixed(6)),
            lng: Number((prev.lng + deltaLng).toFixed(6)),
            heading: (prevHeading + (Math.random() * 20 - 10) + 360) % 360,
          };
        }

        // Publish live GPS ping to Centrifugo WebSocket
        realtimeBus.publishDriverLocation({
          driverId: session?.userId || 'drv_901',
          driverName: session?.name || 'Marcus Sterling',
          vehicleType: session?.vehicleType || 'PREMIUM',
          latitude: newPos.lat,
          longitude: newPos.lng,
          heading: newPos.heading ?? 45,
          isAvailable: !activeTrip,
        });

        // Sync with single source of truth
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
    }, 1500);

    return () => clearInterval(interval);
  }, [isOnline, activeTrip, tripStage, activeRoadCoords, session]);

  // Real average rating calculation across all rated shift trips
  const ratedTrips = shiftLog.filter((t) => t.rating !== undefined);
  const calculatedDriverRating =
    ratedTrips.length > 0
      ? (ratedTrips.reduce((acc, t) => acc + (t.rating || 5), 0) / ratedTrips.length).toFixed(2)
      : session?.rating
      ? session.rating.toFixed(2)
      : '5.00';

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

  // Listen for real-time tip payments & feedback from rider
  useEffect(() => {
    const unsubscribe = realtimeBus.onTripStatus('', (event: TripStatusEvent) => {
      if (event.status === 'COMPLETED') {
        // Real-time update for star rating & feedback
        if (event.rating !== undefined || (event.tipAmount && event.tipAmount > 0)) {
          setShiftLog((prev) => {
            const updated = prev.map((item, idx) => {
              const isMatch =
                item.id === event.tripId ||
                (event.tripId && item.id && (item.id.includes(event.tripId) || event.tripId.includes(item.id))) ||
                idx === 0;
              if (isMatch) {
                return {
                  ...item,
                  ...(event.rating !== undefined ? { rating: event.rating } : {}),
                  ...(event.tipAmount && event.tipAmount > 0
                    ? { tip: `+$${event.tipAmount.toFixed(2)} Tip (100% Driver)` }
                    : {}),
                };
              }
              return item;
            });

            if (session?.userId) {
              try {
                const rawStats = localStorage.getItem(`urban_driver_stats_${session.userId}`);
                const cur = rawStats ? JSON.parse(rawStats) : {};
                localStorage.setItem(
                  `urban_driver_stats_${session.userId}`,
                  JSON.stringify({ ...cur, shiftLog: updated })
                );
              } catch {}
            }
            return updated;
          });
        }

        if (event.tipAmount && event.tipAmount > 0) {
          setEarnings((prev) => {
            const updated = Number((prev + event.tipAmount!).toFixed(2));
            if (session?.userId) {
              try {
                const rawStats = localStorage.getItem(`urban_driver_stats_${session.userId}`);
                const cur = rawStats ? JSON.parse(rawStats) : {};
                localStorage.setItem(
                  `urban_driver_stats_${session.userId}`,
                  JSON.stringify({ ...cur, earnings: updated })
                );
              } catch {}
            }
            return updated;
          });

          setTipToast({ amount: event.tipAmount, riderName: event.driverName || 'Alexander Vance' });
          setTimeout(() => setTipToast(null), 6000);
        }
      }
    });

    return () => unsubscribe();
  }, [session?.userId]);

  // Sync when window storage changes (e.g. rider rated in another tab)
  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key && (e.key.startsWith('urban_rider_history_') || e.key.startsWith('urban_driver_stats_'))) {
        try {
          const rawStats = localStorage.getItem(`urban_driver_stats_${session?.userId}`);
          if (rawStats) {
            const parsed = JSON.parse(rawStats);
            if (parsed.shiftLog) setShiftLog(parsed.shiftLog);
            if (parsed.earnings) setEarnings(parsed.earnings);
            if (parsed.completedTrips) setCompletedTrips(parsed.completedTrips);
          }
        } catch {}
      }
    };
    window.addEventListener('storage', handleStorage);
    return () => window.removeEventListener('storage', handleStorage);
  }, [session?.userId]);

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
    const feeBreakdown = calculatePlatformFee(accepted.fareAmount, accepted.tipAmount || 0);

    setActiveOffer(null);
    setActiveTrip(accepted);
    setTripStage('ACCEPTED_EN_ROUTE_PICKUP');
    setEnteredOtp(['', '', '', '']);
    setOtpError(false);
    setIsOtpVerified(false);

    const driverDisplayName = session?.name || 'Chauffeur Partner';

    // Broadcast status to Rider App
    realtimeBus.publishTripStatus({
      tripId: accepted.tripId,
      status: 'ACCEPTED_EN_ROUTE_PICKUP',
      driverId: session?.userId || 'drv_901',
      driverName: driverDisplayName,
      driverRating: session?.rating || 5.0,
      vehicleModel: session?.vehicleModel || 'Tesla Model S (Obsidian Black)',
      licensePlate: session?.vehiclePlate || 'NY-7890',
      driverLat: driverPos.lat,
      driverLng: driverPos.lng,
      etaMinutes: 4,
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
      platformFee: feeBreakdown.platformFee,
      driverNetFare: feeBreakdown.driverNetFare,
    });

    // Update Persistent Single Source of Truth
    tripStore.updateStatus('ACCEPTED_EN_ROUTE_PICKUP', {
      driverId: session?.userId || 'drv_901',
      driverName: driverDisplayName,
      driverRating: session?.rating || 5.0,
      vehicleModel: session?.vehicleModel || 'Tesla Model S (Obsidian Black)',
      licensePlate: session?.vehiclePlate || 'NY-7890',
      driverLat: driverPos.lat,
      driverLng: driverPos.lng,
      pickupAddress: accepted.pickupAddress,
      pickupLat: accepted.pickupLat,
      pickupLng: accepted.pickupLng,
      dropoffAddress: accepted.dropoffAddress,
      dropoffLat: accepted.dropoffLat,
      dropoffLng: accepted.dropoffLng,
      fareAmount: accepted.fareAmount,
      platformFee: feeBreakdown.platformFee,
      driverNetFare: feeBreakdown.driverNetFare,
    });
  };

  const handleDeclineOffer = () => {
    setActiveOffer(null);
  };

  // Quick Simulation Shortcut: Jump within 20m of pickup
  const handleSimulateArrivePickup = () => {
    if (!activeTrip) return;
    const label = session?.name ? `${session.name} (${session.vehicleModel || 'Vehicle'})` : 'Chauffeur';
    setDriverPos({
      lat: activeTrip.pickupLat + 0.0001,
      lng: activeTrip.pickupLng + 0.0001,
      heading: 90,
      label,
    });
    setTripStage('ARRIVED_AT_PICKUP');
    tripStore.updateStatus('ARRIVED_AT_PICKUP', {
      driverLat: activeTrip.pickupLat + 0.0001,
      driverLng: activeTrip.pickupLng + 0.0001,
    });
    realtimeBus.publishTripStatus({
      tripId: activeTrip.tripId,
      status: 'ARRIVED_AT_PICKUP',
      driverId: session?.userId || 'driver_current',
      driverName: session?.name || 'Chauffeur Partner',
      distanceMeters: 12,
    });
  };

  // Quick Simulation Shortcut: Jump within 20m of dropoff
  const handleSimulateArriveDropoff = () => {
    if (!activeTrip) return;
    const label = session?.name ? `${session.name} (${session.vehicleModel || 'Vehicle'})` : 'Chauffeur';
    setDriverPos({
      lat: activeTrip.dropoffLat + 0.0001,
      lng: activeTrip.dropoffLng + 0.0001,
      heading: 180,
      label,
    });
    setTripStage('ARRIVED_AT_DESTINATION');
    tripStore.updateStatus('ARRIVED_AT_DESTINATION', {
      driverLat: activeTrip.dropoffLat + 0.0001,
      driverLng: activeTrip.dropoffLng + 0.0001,
    });
    realtimeBus.publishTripStatus({
      tripId: activeTrip.tripId,
      status: 'ARRIVED_AT_DESTINATION',
      driverId: session?.userId || 'driver_current',
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
    const entered = enteredOtp.join('');
    const expected = activeTrip.otp || '8421';

    if (entered === expected) {
      setIsOtpVerified(true);
      setOtpError(false);
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
    const feeBreakdown = calculatePlatformFee(activeTrip.fareAmount, activeTrip.tipAmount || 0);

    realtimeBus.publishTripStatus({
      tripId: activeTrip.tripId,
      status: 'COMPLETED',
      driverId: session?.userId || 'drv_901',
      fareAmount: activeTrip.fareAmount,
      platformFee: feeBreakdown.platformFee,
      driverNetFare: feeBreakdown.driverNetFare,
    });

    tripStore.updateStatus('COMPLETED');
    
    // Only Net Driver Fare is credited to the driver wallet (Platform fee is retained)
    const earnedAmount = feeBreakdown.driverNetFare;
    const newEarnings = Number((earnings + earnedAmount).toFixed(2));
    const newCompletedCount = completedTrips + 1;
    const distanceKm = activeTrip.pickupLat && activeTrip.dropoffLat
      ? (getDistanceInMeters(activeTrip.pickupLat, activeTrip.pickupLng, activeTrip.dropoffLat, activeTrip.dropoffLng) / 1000).toFixed(1)
      : '3.4';

    const newLogItem: ShiftTripLog = {
      id: activeTrip.tripId,
      time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      dest: activeTrip.dropoffAddress,
      fare: `$${feeBreakdown.driverNetFare.toFixed(2)} Net`,
      grossFare: `$${feeBreakdown.grossBaseFare.toFixed(2)}`,
      platformFee: `-$${feeBreakdown.platformFee.toFixed(2)} (${feeBreakdown.feePercentage}%)`,
      miles: `${distanceKm} km`,
      rating: 5,
    };
    const updatedShiftLog = [newLogItem, ...shiftLog];

    setEarnings(newEarnings);
    setCompletedTrips(newCompletedCount);
    setShiftLog(updatedShiftLog);

    if (session?.userId) {
      localStorage.setItem(
        `urban_driver_stats_${session.userId}`,
        JSON.stringify({
          earnings: newEarnings,
          completedTrips: newCompletedCount,
          acceptanceRate,
          shiftLog: updatedShiftLog,
        })
      );
    }

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

  if (isAuthChecking) {
    return (
      <div className="h-screen w-screen bg-[#FCF9F8] flex items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="w-10 h-10 border-4 border-[#008A5E] border-t-transparent rounded-full animate-spin"></div>
          <span className="text-xs font-bold text-slate-500 uppercase tracking-widest">
            Authenticating Partner Cockpit...
          </span>
        </div>
      </div>
    );
  }

  return (
    <div className="h-screen w-screen flex bg-[#FCF9F8] overflow-hidden select-none">
      {/* Real-Time Tip Received Celebration Toast */}
      {tipToast && (
        <div className="fixed top-6 right-6 z-50 p-4 bg-white border-2 border-emerald-500 rounded-2xl shadow-2xl animate-in slide-in-from-top-4 duration-300 flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center font-bold">
            <Sparkles className="w-5 h-5" />
          </div>
          <div>
            <h4 className="text-sm font-extrabold text-slate-900">🎉 Tip Received!</h4>
            <p className="text-xs text-slate-600">
              +{tipToast.amount.toFixed(2)} credited directly to your net earnings (100% driver kept).
            </p>
          </div>
        </div>
      )}

      {/* 1. Left Fixed Sidebar (300px) */}
      <aside className="w-[300px] h-full bg-white border-r border-[#DCD9D9] flex flex-col justify-between z-30">
        <div>
          {/* Partner Branding Header */}
          <div className="h-[72px] px-6 border-b border-[#DCD9D9] flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-[#008A5E] flex items-center justify-center text-white shadow-md shadow-emerald-500/20">
              <Compass className="w-5 h-5" />
            </div>
            <div>
              <span className="font-extrabold text-lg tracking-tight text-[#1F1F1F]">
                URBAN<span className="text-[#008A5E]">PRIME</span>
              </span>
              <span className="block text-[10px] tracking-widest uppercase font-semibold text-emerald-600 -mt-1">
                Partner Cockpit
              </span>
            </div>
          </div>

          {/* Driver Profile Card */}
          <div className="p-5 border-b border-[#DCD9D9] bg-[#FCF9F8]/60 space-y-4">
            <div className="flex items-center gap-3.5">
              <div className="relative">
                <div className="w-12 h-12 rounded-full bg-slate-900 text-white font-bold flex items-center justify-center text-sm shadow-md">
                  {session?.name
                    ? session.name
                        .split(' ')
                        .map((n: string) => n[0])
                        .join('')
                    : 'MS'}
                </div>
                <div
                  className={`absolute bottom-0 right-0 w-3.5 h-3.5 rounded-full border-2 border-white ${
                    isOnline ? 'bg-emerald-500' : 'bg-slate-400'
                  }`}
                ></div>
              </div>
              <div className="flex-1 overflow-hidden">
                <h3 className="text-sm font-extrabold text-[#1F1F1F] truncate">
                  {session?.name || 'Marcus Sterling'}
                </h3>
                <p className="text-xs text-slate-500 truncate">
                  {session?.vehicleModel || 'Tesla Model S (Obsidian Black)'}
                </p>
                <div className="flex items-center gap-1.5 mt-0.5">
                  <div className="flex items-center text-amber-500">
                    <Star className="w-3 h-3 fill-amber-500 text-amber-500" />
                    <span className="text-xs font-bold text-slate-700 ml-1">
                      {calculatedDriverRating}
                    </span>
                  </div>
                  {session?.vehiclePlate && (
                    <span className="text-[10px] px-1 py-0.2 bg-white rounded font-mono font-bold text-slate-600 border border-slate-200">
                      {session.vehiclePlate}
                    </span>
                  )}
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
            <button
              type="button"
              onClick={() => {
                clearStoredDriverSession();
                router.push('/driver/login');
              }}
              className="w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-semibold text-slate-500 hover:text-red-600 hover:bg-red-50 transition-colors text-left cursor-pointer"
            >
              <LogOut className="w-4 h-4 text-slate-400" />
              Sign out
            </button>
          </nav>
        </div>

        {/* Clean Status Tag */}
        <div className="p-4 border-t border-[#DCD9D9] text-[11px] text-slate-400 font-mono flex items-center justify-between">
          <span>URBANPRIME OS v3.2</span>
          <span className="text-emerald-600 font-bold">100% TIP PASS-THROUGH</span>
        </div>
      </aside>

      {/* 2. Main Executive Cockpit Canvas */}
      <main className="flex-1 flex flex-col h-full overflow-hidden">
        {/* Top Partner Navigation Bar */}
        <header className="h-[72px] bg-white border-b border-[#DCD9D9] px-8 flex items-center justify-between shadow-xs z-20">
          <div>
            <h1 className="text-xl font-extrabold text-[#1F1F1F] tracking-tight">
              Executive Partner Cockpit
            </h1>
            <p className="text-xs text-slate-500">
              Active Shift • Progressive Platform Tier (15%-35%) • 100% Tips Kept
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
                    Net Take-Home Earnings
                  </span>
                  <div className="w-8 h-8 rounded-lg bg-emerald-50 text-[#008A5E] flex items-center justify-center">
                    <DollarSign className="w-4 h-4" />
                  </div>
                </div>
                <div className="text-2xl font-extrabold text-[#008A5E]">
                  ${earnings.toFixed(2)}
                </div>
                <span className="text-[11px] text-slate-500 font-medium mt-1 flex items-center gap-1">
                  <TrendingUp className="w-3 h-3 text-emerald-600" /> Net Fares + 100% Rider Tips
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
                  {completedTrips} {completedTrips === 1 ? 'Trip' : 'Trips'}
                </div>
                <span className="text-[11px] text-slate-400 font-medium mt-1">
                  {completedTrips > 0
                    ? `Driver Rating: ${calculatedDriverRating}★`
                    : 'Awaiting first dispatch'}
                </span>
              </div>
            </div>

            {/* Live Interactive Map Canvas */}
            <div className="flex-1 rounded-[24px] border border-[#DCD9D9] overflow-hidden relative shadow-lg bg-slate-900">
              <MapboxView
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
                showTrackingBadge={false}
                className="w-full h-full"
              />

              {/* ACTIVE TRIP FLOATING HUD ON MAP */}
              {activeTrip && (() => {
                const tripBreakdown = calculatePlatformFee(activeTrip.fareAmount, activeTrip.tipAmount || 0);
                return (
                  <div className="absolute top-4 left-4 right-4 z-20 p-5 bg-white/95 backdrop-blur-md rounded-2xl border-2 border-[#008A5E] shadow-2xl space-y-4">
                    {/* Top Header of HUD */}
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 rounded-xl bg-[#008A5E] text-white flex items-center justify-center font-bold">
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
                            <span className="px-2.5 py-0.5 rounded bg-emerald-100 text-[#008A5E] font-bold text-[10px] uppercase">
                              ${tripBreakdown.driverNetFare.toFixed(2)} Net Payout ({tripBreakdown.feePercentage}% fee deducted)
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
                        <span className="text-base font-black text-[#008A5E] font-mono">
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
                          <span>OTP Verified • Driving along road to {activeTrip.dropoffAddress}</span>
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
                          Complete Trip & Payout ${tripBreakdown.driverNetFare.toFixed(2)} Net
                        </button>
                      </div>
                    )}
                  </div>
                );
              })()}
            </div>
          </div>

          {/* Right Column: Shift Trip Log (4 cols) */}
          <div className="col-span-4 flex flex-col gap-6 h-full">
            <div className="flex-1 bg-white rounded-2xl border border-[#DCD9D9] p-5 flex flex-col shadow-xs overflow-hidden">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-extrabold text-[#1F1F1F]">Shift Trip Log</h3>
                <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-emerald-50 text-[#008A5E]">
                  {shiftLog.length} {shiftLog.length === 1 ? 'Trip' : 'Trips'}
                </span>
              </div>

              <div className="flex-1 overflow-y-auto space-y-3">
                {shiftLog.length > 0 ? (
                  shiftLog.map((item, idx) => (
                    <div
                      key={item.id || idx}
                      className="p-3.5 rounded-xl bg-[#FCF9F8] border border-[#DCD9D9] text-xs flex justify-between items-center group hover:border-[#008A5E] transition-colors"
                    >
                      <div className="overflow-hidden pr-2 space-y-0.5">
                        <div className="font-bold text-[#1F1F1F] truncate max-w-[160px]">{item.dest}</div>
                        <div className="text-[11px] text-slate-400">
                          {item.time} • {item.miles}
                        </div>
                        {item.grossFare && (
                          <div className="text-[10px] text-slate-500">
                            Gross: {item.grossFare} ({item.platformFee})
                          </div>
                        )}
                      </div>
                      <div className="text-right flex-shrink-0 space-y-0.5">
                        <div className="font-extrabold text-[#008A5E] text-sm">{item.fare}</div>
                        {item.tip && <div className="text-[10px] text-emerald-700 font-bold">{item.tip}</div>}
                        <div className="flex items-center justify-end gap-0.5 mt-0.5">
                          {[1, 2, 3, 4, 5].map((starVal) => {
                            const tripRating = item.rating !== undefined ? item.rating : 5;
                            return (
                              <Star
                                key={starVal}
                                className={`w-3 h-3 ${
                                  starVal <= tripRating
                                    ? 'fill-amber-400 text-amber-400'
                                    : 'fill-slate-200 text-slate-200'
                                }`}
                              />
                            );
                          })}
                          <span className="text-[10px] font-bold text-slate-700 ml-1 font-mono">
                            {(item.rating !== undefined ? item.rating : 5)}.0
                          </span>
                        </div>
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="h-full flex flex-col items-center justify-center text-center p-6 space-y-3 select-none">
                    <div className="w-12 h-12 rounded-2xl bg-slate-100 text-slate-400 flex items-center justify-center">
                      <Car className="w-6 h-6" />
                    </div>
                    <div>
                      <h4 className="text-xs font-bold text-slate-800">No Shift Trips Yet</h4>
                      <p className="text-[11px] text-slate-400 mt-1 leading-relaxed max-w-[200px]">
                        Stay online to accept dispatch offers. Completed trips will appear here automatically.
                      </p>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* 3. HIGH URGENCY 15-SECOND DISPATCH MODAL OVERLAY */}
        {activeOffer && (() => {
          const breakdown = calculatePlatformFee(activeOffer.fareAmount, activeOffer.tipAmount || 0);
          return (
            <div className="fixed inset-0 z-50 bg-slate-950/70 backdrop-blur-md flex items-center justify-center p-6 animate-in fade-in duration-200">
              <div className="w-full max-w-lg bg-white rounded-[28px] border-2 border-[#008A5E] p-8 shadow-2xl relative overflow-hidden">
                <div className="flex items-center justify-between mb-6">
                  <div>
                    <span className="px-2.5 py-1 rounded-full bg-emerald-100 text-[#008A5E] text-xs font-extrabold uppercase tracking-wider">
                      New Dispatch Request
                    </span>
                    <h2 className="text-2xl font-extrabold text-[#1F1F1F] mt-1">Accept Incoming Ride?</h2>
                  </div>

                  {/* SVG Countdown Ring */}
                  <div className="relative w-20 h-20 flex items-center justify-center">
                    <svg className="w-20 h-20 transform -rotate-90">
                      <circle cx="40" cy="40" r={radius} stroke="#E7F0FF" strokeWidth="6" fill="transparent" />
                      <circle
                        cx="40"
                        cy="40"
                        r={radius}
                        stroke="#008A5E"
                        strokeWidth="6"
                        fill="transparent"
                        strokeDasharray={circumference}
                        strokeDashoffset={strokeDashoffset}
                        strokeLinecap="round"
                        className="transition-all duration-1000 ease-linear"
                      />
                    </svg>
                    <div className="absolute inset-0 flex items-center justify-center font-mono font-black text-xl text-[#008A5E]">
                      {secondsRemaining}s
                    </div>
                  </div>
                </div>

                {/* Ride Details Card */}
                <div className="p-5 rounded-2xl bg-[#FCF9F8] border border-[#DCD9D9] mb-6 space-y-4">
                  <div className="flex items-center justify-between border-b border-[#DCD9D9] pb-3">
                    <div>
                      <span className="text-xs text-slate-500 font-medium">Rider Passenger</span>
                      <h4 className="text-base font-bold text-[#1F1F1F]">{activeOffer.riderName}</h4>
                    </div>
                    <div className="text-right">
                      <span className="text-[11px] text-emerald-700 font-bold uppercase tracking-wider block">
                        Net Chauffeur Payout
                      </span>
                      <div className="text-3xl font-black text-[#008A5E] font-mono">
                        ${breakdown.driverNetFare.toFixed(2)}
                      </div>
                    </div>
                  </div>

                  {/* Progressive Platform Fee Breakdown */}
                  <div className="p-3 bg-white rounded-xl border border-slate-200 space-y-2 text-xs">
                    <div className="flex justify-between text-slate-600">
                      <span>Rider Trip Fare:</span>
                      <span className="font-semibold font-mono">${breakdown.grossBaseFare.toFixed(2)}</span>
                    </div>
                    <div className="flex justify-between text-amber-700 font-medium">
                      <span>UrbanPrime Platform Fee ({breakdown.feePercentage}%):</span>
                      <span className="font-bold font-mono">-${breakdown.platformFee.toFixed(2)}</span>
                    </div>
                    <div className="flex justify-between text-[#008A5E] font-extrabold pt-1.5 border-t border-slate-100">
                      <span>Net Take-Home (after deduction):</span>
                      <span className="font-mono text-sm">${breakdown.driverNetFare.toFixed(2)}</span>
                    </div>
                    <div className="pt-1 text-[11px] text-slate-500 italic flex items-center gap-1">
                      <Sparkles className="w-3 h-3 text-amber-500 flex-shrink-0" />
                      <span>100% of rider tips are exempt from platform fees and go directly to you.</span>
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
                    Accept (${breakdown.driverNetFare.toFixed(2)} Net)
                  </button>
                </div>
              </div>
            </div>
          );
        })()}
      </main>
    </div>
  );
}
