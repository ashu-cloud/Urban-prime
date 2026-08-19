/**
 * Persistent Single Source of Truth for Real-Time Trips
 * Survives page refreshes and keeps Rider & Driver states synchronized.
 */

import { TripLifecycleStage } from './socket';

export interface PersistedTripState {
  tripId: string;
  riderId: string;
  riderName: string;
  driverId?: string;
  driverName?: string;
  driverLat?: number;
  driverLng?: number;
  driverHeading?: number;
  driverRating?: number;
  vehicleModel?: string;
  licensePlate?: string;
  status: TripLifecycleStage;
  pickupAddress: string;
  pickupLat: number;
  pickupLng: number;
  dropoffAddress: string;
  dropoffLat: number;
  dropoffLng: number;
  vehicleType: string;
  fareAmount: number;
  platformFee?: number;
  driverNetFare?: number;
  feePercentage?: number;
  tipAmount?: number;
  otp: string;
  createdAt: number;
}

const STORAGE_KEY = 'URBAN_PRIME_ACTIVE_TRIP';

export const tripStore = {
  save(trip: PersistedTripState) {
    if (typeof window !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(trip));
    }
  },

  get(): PersistedTripState | null {
    if (typeof window !== 'undefined') {
      const data = localStorage.getItem(STORAGE_KEY);
      if (data) {
        try {
          return JSON.parse(data) as PersistedTripState;
        } catch {
          return null;
        }
      }
    }
    return null;
  },

  updateStatus(status: TripLifecycleStage, extra?: Partial<PersistedTripState>): PersistedTripState | null {
    const current = tripStore.get();
    if (current) {
      const updated: PersistedTripState = {
        ...current,
        status,
        ...extra,
      };
      tripStore.save(updated);
      return updated;
    }
    return null;
  },

  clear() {
    if (typeof window !== 'undefined') {
      localStorage.removeItem(STORAGE_KEY);
    }
  },
};
