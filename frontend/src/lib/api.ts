/**
 * API Client for Urban Prime connecting to APISIX Gateway (http://localhost:9080)
 * Includes graceful mock fallbacks for standalone UI testing and simulation.
 */

import { TripLifecycleStage } from './socket';

const APISIX_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9080';

export interface UserSession {
  userId: string;
  email: string;
  role: 'RIDER' | 'DRIVER';
  token: string;
  name: string;
}

export interface TripRequest {
  riderId: string;
  pickupAddress: string;
  pickupLat: number;
  pickupLng: number;
  dropoffAddress: string;
  dropoffLat: number;
  dropoffLng: number;
  vehicleType: 'SEDAN' | 'SUV' | 'PREMIUM' | 'BIKE';
  fareAmount: number;
  paymentMethodId?: string;
}

export interface TripResponse {
  tripId: string;
  riderId?: string;
  status: TripLifecycleStage | 'PENDING' | 'ACCEPTED' | 'DRIVER_ARRIVING';
  fare?: number;
  fareAmount?: number;
  currency?: string;
  driverId?: string;
  vehicleModel?: string;
  driverName?: string;
  driverRating?: number;
  estimatedMinutes?: number;
  pickupLocation?: { latitude: number; longitude: number; address: string };
  dropoffLocation?: { latitude: number; longitude: number; address: string };
  vehicleType?: string;
  createdAt?: string;
}

// Session Storage Helpers
export const getStoredSession = (): UserSession | null => {
  if (typeof window === 'undefined') return null;
  try {
    const raw = localStorage.getItem('urban_prime_session');
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
};

export const setStoredSession = (session: UserSession) => {
  if (typeof window !== 'undefined') {
    localStorage.setItem('urban_prime_session', JSON.stringify(session));
  }
};

export const clearStoredSession = () => {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('urban_prime_session');
  }
};

// API Methods
export const api = {
  // 1. Auth Service via APISIX (/api/v1/auth/login)
  async login(email: string, role: 'RIDER' | 'DRIVER'): Promise<UserSession> {
    try {
      const res = await fetch(`${APISIX_BASE_URL}/api/v1/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, role }),
      });

      if (res.ok) {
        const data = await res.json();
        const session: UserSession = {
          userId: data.userId || (role === 'DRIVER' ? 'drv_901' : 'rid_001'),
          email,
          role,
          token: data.token || 'mock_jwt_token',
          name: data.name || (role === 'DRIVER' ? 'Marcus Sterling' : 'Alexander Vance'),
        };
        setStoredSession(session);
        return session;
      }
    } catch {
      // Mock Fallback
    }

    const fallbackSession: UserSession = {
      userId: role === 'DRIVER' ? 'drv_901' : 'rid_001',
      email,
      role,
      token: 'jwt_mock_token_123',
      name: role === 'DRIVER' ? 'Marcus Sterling' : 'Alexander Vance',
    };
    setStoredSession(fallbackSession);
    return fallbackSession;
  },

  // 2. Trip Service via APISIX (/api/v1/trips)
  async createTrip(req: TripRequest): Promise<TripResponse> {
    try {
      const res = await fetch(`${APISIX_BASE_URL}/api/v1/trips`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      });

      if (res.ok) {
        return await res.json();
      }
    } catch {
      // Mock Fallback
    }

    return {
      tripId: `trip_${Date.now()}`,
      riderId: req.riderId,
      status: 'MATCHING',
      fare: req.fareAmount,
      fareAmount: req.fareAmount,
      currency: 'USD',
      pickupLocation: { latitude: req.pickupLat, longitude: req.pickupLng, address: req.pickupAddress },
      dropoffLocation: { latitude: req.dropoffLat, longitude: req.dropoffLng, address: req.dropoffAddress },
      vehicleType: req.vehicleType,
      createdAt: new Date().toISOString(),
    };
  },

  // 3. Driver Location Telemetry (/api/v1/location/driver)
  async updateLocation(driverId: string, lat: number, lng: number, heading: number) {
    try {
      await fetch(`${APISIX_BASE_URL}/api/v1/location/driver`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ driverId, latitude: lat, longitude: lng, heading }),
      });
    } catch {
      // Offline fallback
    }
  },
};
