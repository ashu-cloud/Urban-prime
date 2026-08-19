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
  phone?: string;
  vehicleModel?: string;
  vehiclePlate?: string;
  vehicleType?: 'SEDAN' | 'SUV' | 'PREMIUM' | 'BIKE';
  rating?: number;
}

export interface DriverOnboardingRequest {
  fullName: string;
  email: string;
  phone: string;
  password?: string;
  vehicleMake: string;
  vehicleModel: string;
  vehicleYear: string;
  vehicleColor: string;
  vehiclePlate: string;
  vehicleType: 'SEDAN' | 'SUV' | 'PREMIUM' | 'BIKE';
  licenseNumber: string;
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

// Session Storage Helpers for Isolated Multi-Portal Auth
export const getStoredRiderSession = (): UserSession | null => {
  if (typeof window === 'undefined') return null;
  try {
    const raw = localStorage.getItem('urban_rider_session');
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
};

export const setStoredRiderSession = (session: UserSession) => {
  if (typeof window !== 'undefined') {
    localStorage.setItem('urban_rider_session', JSON.stringify(session));
  }
};

export const clearStoredRiderSession = () => {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('urban_rider_session');
  }
};

export const getStoredDriverSession = (): UserSession | null => {
  if (typeof window === 'undefined') return null;
  try {
    const raw = localStorage.getItem('urban_driver_session');
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
};

export const setStoredDriverSession = (session: UserSession) => {
  if (typeof window !== 'undefined') {
    localStorage.setItem('urban_driver_session', JSON.stringify(session));
  }
};

export const clearStoredDriverSession = () => {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('urban_driver_session');
  }
};

// Backwards compatibility helper
export const getStoredSession = (role?: 'RIDER' | 'DRIVER'): UserSession | null => {
  if (role === 'DRIVER') return getStoredDriverSession();
  if (role === 'RIDER') return getStoredRiderSession();
  return getStoredRiderSession() || getStoredDriverSession();
};

export const setStoredSession = (session: UserSession) => {
  if (session.role === 'DRIVER') {
    setStoredDriverSession(session);
  } else {
    setStoredRiderSession(session);
  }
};

export const clearStoredSession = () => {
  clearStoredRiderSession();
  clearStoredDriverSession();
};

async function extractCleanErrorMessage(res: Response, defaultMsg: string): Promise<string> {
  if (res.status >= 500) {
    return 'Authentication service is temporarily unavailable. Please try again shortly.';
  }
  try {
    const text = await res.text();
    if (!text || text.includes('<html') || text.includes('<!DOCTYPE') || text.includes('openresty') || text.includes('APISIX')) {
      return defaultMsg;
    }
    try {
      const parsed = JSON.parse(text);
      return parsed.message || parsed.error || defaultMsg;
    } catch {
      return text.trim() || defaultMsg;
    }
  } catch {
    return defaultMsg;
  }
}

// API Methods
export const api = {
  // 1. Rider Login (/auth/login with role=RIDER)
  async loginRider(email: string, password?: string): Promise<UserSession> {
    const cleanEmail = email.trim().toLowerCase();

    // Try APISIX gateway first, then fallback to direct auth service
    const endpoints = [
      `${APISIX_BASE_URL}/auth/login`,
      `${APISIX_BASE_URL}/api/v1/auth/login`,
      'http://localhost:8080/auth/login',
    ];

    let isNetworkError = false;

    for (const url of endpoints) {
      try {
        const res = await fetch(url, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email: cleanEmail, role: 'RIDER', password }),
        });

        if (res.ok) {
          const data = await res.json();
          const session: UserSession = {
            userId: data.user?.id || data.userId || `rid_${Date.now().toString().slice(-4)}`,
            email: cleanEmail,
            role: 'RIDER',
            token: data.accessToken || data.token || 'jwt_token',
            name: data.user?.fullName || data.name || cleanEmail.split('@')[0],
            phone: data.user?.phone || data.phone,
          };
          setStoredRiderSession(session);
          return session;
        } else if (res.status >= 500) {
          // Gateway or service error, try next endpoint
          isNetworkError = true;
          continue;
        } else {
          // Explicit 401 or 400 rejection from auth-service
          const errMsg = await extractCleanErrorMessage(
            res,
            'Invalid email or password. Please check your credentials or create an account.'
          );
          throw new Error(errMsg);
        }
      } catch (err: any) {
        if (err.message && !err.message.includes('fetch') && !err.message.includes('NetworkError') && !err.message.includes('Failed to fetch')) {
          throw err;
        }
        isNetworkError = true;
      }
    }

    // If backend is unreachable and user is using the seeded demo account, allow demo access
    if (cleanEmail === 'alexander.vance@urbanprime.com') {
      const demoSession: UserSession = {
        userId: 'rid_001',
        email: cleanEmail,
        role: 'RIDER',
        token: 'mock_jwt_rider_token',
        name: 'Alexander Vance',
        phone: '+1 (555) 345-6789',
      };
      setStoredRiderSession(demoSession);
      return demoSession;
    }

    // For any other non-existent/unregistered email, strictly reject!
    throw new Error(
      'Account not found or password incorrect. Please create an account or verify your credentials.'
    );
  },

  // 2. Rider Registration (/auth/register)
  async registerRider(email: string, name: string, password?: string, phone?: string): Promise<UserSession> {
    const cleanEmail = email.trim().toLowerCase();
    const endpoints = [
      `${APISIX_BASE_URL}/auth/register`,
      `${APISIX_BASE_URL}/api/v1/auth/register`,
      'http://localhost:8080/auth/register',
    ];

    let isNetworkError = false;

    for (const url of endpoints) {
      try {
        const res = await fetch(url, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            email: cleanEmail,
            full_name: name,
            password: password || 'SecurePassword123!',
            phone: phone || `+1555${Math.floor(1000000 + Math.random() * 9000000)}`,
            role: 'RIDER',
          }),
        });

        if (res.ok) {
          const data = await res.json();
          const session: UserSession = {
            userId: data.user?.id || data.userId || `rid_${Date.now().toString().slice(-4)}`,
            email: cleanEmail,
            role: 'RIDER',
            token: data.accessToken || data.token || 'mock_rider_jwt',
            name: data.user?.fullName || name,
            phone: data.user?.phone || phone,
          };
          setStoredRiderSession(session);
          return session;
        } else if (res.status >= 500) {
          isNetworkError = true;
          continue;
        } else {
          const errMsg = await extractCleanErrorMessage(
            res,
            'Registration failed. An account with this email may already exist.'
          );
          throw new Error(errMsg);
        }
      } catch (err: any) {
        if (err.message && !err.message.includes('fetch') && !err.message.includes('NetworkError') && !err.message.includes('Failed to fetch')) {
          throw err;
        }
        isNetworkError = true;
      }
    }

    // Local registration fallback if backend service is completely unreachable
    const session: UserSession = {
      userId: `rid_${Date.now().toString().slice(-4)}`,
      email: cleanEmail,
      role: 'RIDER',
      token: 'jwt_rider_token',
      name,
      phone,
    };
    setStoredRiderSession(session);
    return session;
  },

  // 3. Driver Login (/auth/login with role=DRIVER)
  async loginDriver(email: string, password?: string): Promise<UserSession> {
    const cleanEmail = email.trim().toLowerCase();
    const endpoints = [
      `${APISIX_BASE_URL}/auth/login`,
      `${APISIX_BASE_URL}/api/v1/auth/login`,
      'http://localhost:8080/auth/login',
    ];

    let isNetworkError = false;

    for (const url of endpoints) {
      try {
        const res = await fetch(url, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email: cleanEmail, role: 'DRIVER', password }),
        });

        if (res.ok) {
          const data = await res.json();
          const session: UserSession = {
            userId: data.user?.id || data.userId || `drv_${Date.now().toString().slice(-4)}`,
            email: cleanEmail,
            role: 'DRIVER',
            token: data.accessToken || data.token || 'mock_driver_jwt',
            name: data.user?.fullName || data.name || cleanEmail.split('@')[0],
            vehicleModel: data.vehicleModel || 'Executive Fleet Vehicle',
            vehiclePlate: data.vehiclePlate || 'NYC-PRIME',
            vehicleType: (data.vehicleType as any) || 'PREMIUM',
            rating: data.rating || 5.0,
          };
          setStoredDriverSession(session);
          return session;
        } else if (res.status >= 500) {
          isNetworkError = true;
          continue;
        } else {
          const errMsg = await extractCleanErrorMessage(
            res,
            'Invalid partner credentials. Please check your work email and password.'
          );
          throw new Error(errMsg);
        }
      } catch (err: any) {
        if (err.message && !err.message.includes('fetch') && !err.message.includes('NetworkError') && !err.message.includes('Failed to fetch')) {
          throw err;
        }
        isNetworkError = true;
      }
    }

    // Seeded Demo Chauffeur fallback if backend unreachable
    if (cleanEmail === 'marcus.sterling@driver.urbanprime.com') {
      const demoDriver: UserSession = {
        userId: 'drv_901',
        email: cleanEmail,
        role: 'DRIVER',
        token: 'mock_jwt_driver_token',
        name: 'Marcus Sterling',
        rating: 5.0,
        vehicleModel: 'Tesla Model S (Obsidian Black)',
        vehiclePlate: 'NY-7890',
        vehicleType: 'PREMIUM',
      };
      setStoredDriverSession(demoDriver);
      return demoDriver;
    }

    throw new Error(
      'Partner account not found or password incorrect. Please apply as a Driver Partner or check credentials.'
    );
  },

  // Legacy login method routing
  async login(email: string, role: 'RIDER' | 'DRIVER', password?: string): Promise<UserSession> {
    if (role === 'DRIVER') {
      return this.loginDriver(email, password);
    }
    return this.loginRider(email, password);
  },

  // 4. Driver Onboarding / Registration (/api/v1/drivers/register)
  async registerDriver(req: DriverOnboardingRequest): Promise<UserSession> {
    const fullVehicleModel = `${req.vehicleMake} ${req.vehicleModel} (${req.vehicleColor || 'Obsidian Black'})`;
    const driverId = `drv_${Date.now().toString().slice(-4)}`;

    try {
      // Register in Auth Service
      await fetch(`${APISIX_BASE_URL}/api/v1/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: req.email,
          phone: req.phone,
          password: req.password || 'SecureChauffeur2026!',
          full_name: req.fullName,
          role: 'DRIVER',
        }),
      });

      // Register in Driver Service
      await fetch(`${APISIX_BASE_URL}/api/v1/drivers`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: req.fullName,
          phone: req.phone,
          email: req.email,
          vehicle_type: req.vehicleType,
          vehicle_plate: req.vehiclePlate,
          vehicle_model: fullVehicleModel,
        }),
      });
    } catch {
      // Offline fallback
    }

    const session: UserSession = {
      userId: driverId,
      email: req.email,
      role: 'DRIVER',
      token: `jwt_${Date.now()}_driver`,
      name: req.fullName,
      phone: req.phone,
      vehicleModel: fullVehicleModel,
      vehiclePlate: req.vehiclePlate,
      vehicleType: req.vehicleType,
      rating: 5.0,
    };
    setStoredSession(session);
    return session;
  },

  // 3. Trip Service via APISIX (/api/v1/trips)
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

  // 4. Driver Location Telemetry (/api/v1/location/driver)
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
