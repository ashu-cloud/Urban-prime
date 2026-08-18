/**
 * API Client for Urban Prime connecting to APISIX Gateway (http://localhost:9080)
 * Includes graceful mock fallbacks for standalone UI testing and simulation.
 */

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
  status: 'PENDING' | 'MATCHING' | 'ACCEPTED' | 'DRIVER_ARRIVING' | 'IN_TRANSIT' | 'COMPLETED' | 'CANCELLED';
  fare: number;
  driverId?: string;
  vehicleModel?: string;
  driverName?: string;
  driverRating?: number;
  estimatedMinutes?: number;
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
  // Authentication
  login: async (email: string, role: 'RIDER' | 'DRIVER' = 'RIDER'): Promise<UserSession> => {
    try {
      const res = await fetch(`${APISIX_BASE_URL}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password: 'password123', role }),
      });
      if (res.ok) {
        const data = await res.json();
        const session: UserSession = {
          userId: data.user_id || `usr_${Math.random().toString(36).substring(2, 8)}`,
          email,
          role,
          token: data.token || 'mock_jwt_token',
          name: email.split('@')[0].replace('.', ' ').toUpperCase(),
        };
        setStoredSession(session);
        return session;
      }
    } catch {
      // Fallback mock session if APISIX is not reachable
    }

    // Mock fallback
    const session: UserSession = {
      userId: role === 'RIDER' ? 'rid_001' : 'drv_901',
      email,
      role,
      token: 'jwt_mock_token_demo',
      name: role === 'RIDER' ? 'Alexander Vance' : 'Marcus Sterling',
    };
    setStoredSession(session);
    return session;
  },

  // Create Trip (Rider -> Saga Orchestrator)
  createTrip: async (payload: TripRequest): Promise<TripResponse> => {
    const session = getStoredSession();
    try {
      const res = await fetch(`${APISIX_BASE_URL}/v1/trips`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${session?.token || ''}`,
        },
        body: JSON.stringify(payload),
      });
      if (res.ok) {
        return await res.json();
      }
    } catch {
      // Fallback
    }

    // Return realistic trip simulation response
    return {
      tripId: `trip_${Math.random().toString(36).substring(2, 9)}`,
      status: 'MATCHING',
      fare: payload.fareAmount,
      estimatedMinutes: 4,
    };
  },

  // Driver Location Update (Driver GPS Firehose)
  updateDriverLocation: async (driverId: string, lat: number, lng: number, heading: number, isAvailable: boolean) => {
    const session = getStoredSession();
    try {
      await fetch(`${APISIX_BASE_URL}/v1/location/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${session?.token || ''}`,
        },
        body: JSON.stringify({
          driver_id: driverId,
          latitude: lat,
          longitude: lng,
          heading,
          is_available: isAvailable,
          timestamp: new Date().toISOString(),
        }),
      });
    } catch {
      // Non-blocking location update
    }
  },

  // Driver Accept/Decline Dispatch
  respondToDispatch: async (tripId: string, driverId: string, accepted: boolean) => {
    const session = getStoredSession();
    try {
      await fetch(`${APISIX_BASE_URL}/v1/drivers/dispatch/respond`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${session?.token || ''}`,
        },
        body: JSON.stringify({
          trip_id: tripId,
          driver_id: driverId,
          action: accepted ? 'ACCEPT' : 'DECLINE',
        }),
      });
    } catch {
      // Non-blocking
    }
  },
};
