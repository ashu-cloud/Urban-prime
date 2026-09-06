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
  fareAmount: number; // Gross Rider Fare (e.g. $22.00)
  platformFee?: number; // Progressive Platform Fee (e.g. $4.40)
  driverNetFare?: number; // Net amount driver receives (e.g. $17.60)
  feePercentage?: number; // Fee rate (e.g. 20%)
  tipAmount?: number; // Tip if pre-authorized (100% to chauffeur)
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
  platformFee?: number;
  driverNetFare?: number;
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
  public onDispatchOffer(
    driverIdOrCallback: string | EventCallback<DispatchOfferEvent>,
    maybeCallback?: EventCallback<DispatchOfferEvent>
  ): () => void {
    const callback = typeof driverIdOrCallback === 'function' ? driverIdOrCallback : maybeCallback!;
    const driverId = typeof driverIdOrCallback === 'string' ? driverIdOrCallback : '';

    const handler = (event: MessageEvent) => {
      if (event.data?.type === 'DISPATCH_OFFER') {
        callback(event.data.payload);
      }
    };

    this.broadcastChannel?.addEventListener('message', handler);

    let sub: any = null;
    if (this.centrifuge && driverId) {
      try {
        sub = this.centrifuge.newSubscription(`driver#${driverId}`);
        sub.on('publication', (ctx: any) => {
          const raw = ctx.data;
          if (raw) {
            const offer: DispatchOfferEvent = {
              tripId: raw.trip_id || raw.tripId || '',
              riderId: raw.rider_id || raw.riderId || 'rider_1',
              riderName: raw.rider_name || raw.riderName || 'Rider',
              pickupAddress: raw.pickup_address || raw.pickupAddress || 'Pickup Location',
              dropoffAddress: raw.dropoff_address || raw.dropoffAddress || 'Destination',
              pickupLat: raw.pickup_lat || raw.pickupLat || 0,
              pickupLng: raw.pickup_lng || raw.pickupLng || 0,
              dropoffLat: raw.dropoff_lat || raw.dropoffLat || 0,
              dropoffLng: raw.dropoff_lng || raw.dropoffLng || 0,
              fareAmount: raw.fare_amount || raw.fareAmount || 20,
              expiresInSeconds: raw.expires_in_seconds || raw.expiresInSeconds || 15,
              otp: raw.otp || '8421',
            };
            callback(offer);
          }
        });
        sub.subscribe();
      } catch (e) {
        console.warn('[Centrifugo] Failed to subscribe to driver channel:', e);
      }
    }

    return () => {
      this.broadcastChannel?.removeEventListener('message', handler);
      sub?.unsubscribe();
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

    let sub: any = null;
    if (this.centrifuge && tripId) {
      try {
        sub = this.centrifuge.newSubscription(`trip#${tripId}`);
        sub.on('publication', (ctx: any) => {
          const raw = ctx.data;
          if (raw) {
            let status: TripLifecycleStage = 'MATCHING';
            const evt = (raw.event_type || raw.type || raw.status || '').toUpperCase();
            if (evt.includes('ACCEPTED') || evt.includes('ON_TRIP')) {
              status = 'ACCEPTED_EN_ROUTE_PICKUP';
            } else if (evt.includes('ARRIVED') && !evt.includes('DEST')) {
              status = 'ARRIVED_AT_PICKUP';
            } else if (evt.includes('IN_TRANSIT') || evt.includes('STARTED')) {
              status = 'IN_TRANSIT';
            } else if (evt.includes('DESTINATION')) {
              status = 'ARRIVED_AT_DESTINATION';
            } else if (evt.includes('COMPLETED')) {
              status = 'COMPLETED';
            } else if (evt.includes('CANCELLED')) {
              status = 'CANCELLED';
            }

            callback({
              tripId: raw.trip_id || raw.tripId || tripId,
              status,
              driverId: raw.driver_id || raw.driverId,
              driverName: raw.driver_name || raw.driverName,
              driverLat: raw.latitude || raw.driverLat,
              driverLng: raw.longitude || raw.driverLng,
            });
          }
        });
        sub.subscribe();
      } catch (e) {
        console.warn('[Centrifugo] Failed to subscribe to trip channel:', e);
      }
    }

    return () => {
      this.broadcastChannel?.removeEventListener('message', handler);
      sub?.unsubscribe();
    };
  }
}

export const realtimeBus = new RealtimeBus();
