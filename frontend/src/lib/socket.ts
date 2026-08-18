/**
 * Real-time Centrifugo WebSocket client with BroadcastChannel fallback for multi-tab simulation.
 */
import { Centrifuge } from 'centrifuge';

const CENTRIFUGO_URL = process.env.NEXT_PUBLIC_CENTRIFUGO_WS || 'ws://localhost:8000/connection/websocket';

export interface LocationPoint {
  lat: number;
  lng: number;
  address?: string;
}

export interface DriverLocationEvent {
  driverId: string;
  latitude: number;
  longitude: number;
  heading: number;
  isAvailable: boolean;
  driverName?: string;
  vehicleType?: string;
}

export interface DispatchOfferEvent {
  tripId: string;
  riderId: string;
  riderName: string;
  pickupAddress: string;
  dropoffAddress: string;
  pickupLat: number;
  pickupLng: number;
  dropoffLat: number;
  dropoffLng: number;
  fareAmount: number;
  expiresInSeconds: number;
  otp: string;
}

export type TripLifecycleStage = 
  | 'MATCHING' 
  | 'ACCEPTED_EN_ROUTE_PICKUP' 
  | 'ARRIVED_AT_PICKUP' 
  | 'IN_TRANSIT' 
  | 'ARRIVED_AT_DESTINATION' 
  | 'COMPLETED' 
  | 'CANCELLED';

export interface TripStatusEvent {
  tripId: string;
  status: TripLifecycleStage;
  driverId?: string;
  driverName?: string;
  driverRating?: number;
  vehicleModel?: string;
  licensePlate?: string;
  driverLat?: number;
  driverLng?: number;
  etaMinutes?: number;
  distanceMeters?: number;
  otp?: string;
  pickupCoords?: LocationPoint;
  dropoffCoords?: LocationPoint;
  fareAmount?: number;
  rating?: number;
  tipAmount?: number;
}

type EventCallback<T> = (data: T) => void;

/**
 * Haversine formula to compute geodesic distance in meters between two lat/lng coordinates.
 */
export function getDistanceInMeters(lat1: number, lon1: number, lat2: number, lon2: number): number {
  const R = 6371e3; // Earth radius in meters
  const φ1 = (lat1 * Math.PI) / 180;
  const φ2 = (lat2 * Math.PI) / 180;
  const Δφ = ((lat2 - lat1) * Math.PI) / 180;
  const Δλ = ((lon2 - lon1) * Math.PI) / 180;

  const a =
    Math.sin(Δφ / 2) * Math.sin(Δφ / 2) +
    Math.cos(φ1) * Math.cos(φ2) * Math.sin(Δλ / 2) * Math.sin(Δλ / 2);
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));

  return Math.round(R * c);
}

class RealtimeBus {
  private centrifuge: Centrifuge | null = null;
  private broadcastChannel: BroadcastChannel | null = null;
  private isConnected = false;

  constructor() {
    if (typeof window !== 'undefined') {
      try {
        this.broadcastChannel = new BroadcastChannel('urban_prime_realtime_mesh');
      } catch {
        // BroadcastChannel unsupported
      }
      this.initCentrifugo();
    }
  }

  private initCentrifugo() {
    try {
      this.centrifuge = new Centrifuge(CENTRIFUGO_URL, {
        token: '',
      });

      this.centrifuge.on('connected', () => {
        this.isConnected = true;
        console.log('[Centrifugo] Connected to real-time WebSocket');
      });

      this.centrifuge.on('disconnected', () => {
        this.isConnected = false;
      });

      this.centrifuge.connect();
    } catch {
      // Fallback to local mesh
    }
  }

  // Publish Driver GPS Position
  public publishDriverLocation(data: DriverLocationEvent) {
    this.broadcastChannel?.postMessage({
      type: 'DRIVER_LOCATION',
      payload: data,
    });

    if (this.isConnected && this.centrifuge) {
      try {
        const sub = this.centrifuge.newSubscription('driver.location.v1');
        sub.publish(data);
      } catch {
        // Ignore
      }
    }
  }

  // Publish Dispatch Offer to Drivers
  public publishDispatchOffer(data: DispatchOfferEvent) {
    this.broadcastChannel?.postMessage({
      type: 'DISPATCH_OFFER',
      payload: data,
    });
  }

  // Publish Trip Status Transition (Accepted, Arriving, Verified In-Transit, Completed)
  public publishTripStatus(data: TripStatusEvent) {
    this.broadcastChannel?.postMessage({
      type: 'TRIP_STATUS',
      payload: data,
    });
  }

  // Subscribe to Driver Locations
  public onDriverLocation(callback: EventCallback<DriverLocationEvent>): () => void {
    const handler = (event: MessageEvent) => {
      if (event.data?.type === 'DRIVER_LOCATION') {
        callback(event.data.payload);
      }
    };

    this.broadcastChannel?.addEventListener('message', handler);

    let sub: any = null;
    if (this.centrifuge) {
      try {
        sub = this.centrifuge.newSubscription('driver.location.v1');
        sub.on('publication', (ctx: any) => {
          callback(ctx.data as DriverLocationEvent);
        });
        sub.subscribe();
      } catch {
        // Fallback
      }
    }

    return () => {
      this.broadcastChannel?.removeEventListener('message', handler);
      sub?.unsubscribe();
    };
  }

  // Subscribe to Incoming Dispatch Offers (Driver Cockpit)
  public onDispatchOffer(callback: EventCallback<DispatchOfferEvent>): () => void {
    const handler = (event: MessageEvent) => {
      if (event.data?.type === 'DISPATCH_OFFER') {
        callback(event.data.payload);
      }
    };

    this.broadcastChannel?.addEventListener('message', handler);

    return () => {
      this.broadcastChannel?.removeEventListener('message', handler);
    };
  }

  // Subscribe to Trip Status Updates (Rider & Driver Screens)
  public onTripStatus(tripId: string, callback: EventCallback<TripStatusEvent>): () => void {
    const handler = (event: MessageEvent) => {
      if (event.data?.type === 'TRIP_STATUS' && (!tripId || event.data.payload.tripId === tripId)) {
        callback(event.data.payload);
      }
    };

    this.broadcastChannel?.addEventListener('message', handler);

    return () => {
      this.broadcastChannel?.removeEventListener('message', handler);
    };
  }
}

export const realtimeBus = new RealtimeBus();
