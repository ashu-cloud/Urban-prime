/**
 * Real-time Centrifugo WebSocket client with BroadcastChannel fallback for multi-tab simulation.
 */
import { Centrifuge } from 'centrifuge';

const CENTRIFUGO_URL = process.env.NEXT_PUBLIC_CENTRIFUGO_WS || 'ws://localhost:8000/connection/websocket';

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
}

export interface TripStatusEvent {
  tripId: string;
  status: 'MATCHING' | 'ACCEPTED' | 'DRIVER_ARRIVING' | 'IN_TRANSIT' | 'COMPLETED' | 'CANCELLED';
  driverId?: string;
  driverName?: string;
  driverRating?: number;
  vehicleModel?: string;
  licensePlate?: string;
  driverLat?: number;
  driverLng?: number;
  etaMinutes?: number;
}

type EventCallback<T> = (data: T) => void;

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
        token: '', // Development mode allows anonymous or custom token
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
      // Fallback
    }
  }

  // Publish Driver GPS Position
  public publishDriverLocation(data: DriverLocationEvent) {
    // 1. BroadcastChannel for local cross-tab instant sync
    this.broadcastChannel?.postMessage({
      type: 'DRIVER_LOCATION',
      payload: data,
    });

    // 2. Centrifugo publication if connected
    if (this.isConnected && this.centrifuge) {
      try {
        const sub = this.centrifuge.newSubscription('driver.location.v1');
        sub.publish(data);
      } catch {
        // Ignore if channel not configured
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

  // Publish Trip Status Transition (Accepted, Arriving, Completed)
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
        // Fallback to BroadcastChannel
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

  // Subscribe to Trip Status Updates (Rider Screen)
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
