'use client';

import React, { useEffect, useRef, useState, useMemo } from 'react';
import Map, { Marker, Source, Layer, NavigationControl, MapRef } from 'react-map-gl/mapbox';
import { MapPin, Navigation, Car, AlertCircle, LocateFixed } from 'lucide-react';
import { fetchMapboxDirections } from '@/lib/directions';
import type { LineString } from 'geojson';

export interface MarkerLocation {
  lat: number;
  lng: number;
  label?: string;
  type?: 'pickup' | 'dropoff' | 'driver';
  heading?: number;
  id?: string;
}

export type RouteLegType = 'TO_PICKUP' | 'TO_DESTINATION' | 'NONE';

interface MapboxViewProps {
  center?: [number, number]; // [lng, lat]
  zoom?: number;
  pickup?: MarkerLocation | null;
  dropoff?: MarkerLocation | null;
  drivers?: MarkerLocation[];
  activeLeg?: RouteLegType;
  activePinMode?: 'PICKUP' | 'DROPOFF' | null;
  showTrackingBadge?: boolean;
  interactive?: boolean;
  className?: string;
  onMapClick?: (coords: { lat: number; lng: number }) => void;
  onPickupDrag?: (coords: { lat: number; lng: number }) => void;
  onDropoffDrag?: (coords: { lat: number; lng: number }) => void;
}

const MAPBOX_TOKEN = process.env.NEXT_PUBLIC_MAPBOX_TOKEN;

export default function MapboxView({
  center = [-73.9851, 40.7484],
  zoom = 14,
  pickup,
  dropoff,
  drivers = [],
  activeLeg = 'NONE',
  activePinMode = null,
  showTrackingBadge = false,
  interactive = true,
  className = 'w-full h-full',
  onMapClick,
  onPickupDrag,
  onDropoffDrag,
}: MapboxViewProps) {
  const mapRef = useRef<MapRef | null>(null);
  const [viewState, setViewState] = useState({
    longitude: center[0],
    latitude: center[1],
    zoom: zoom,
    pitch: 35,
    bearing: 0,
  });

  const [followCar, setFollowCar] = useState(true);

  // Mouse screen coordinates for live floating location-dot cursor
  const [mousePos, setMousePos] = useState<{ x: number; y: number } | null>(null);

  // Exact Street Driving Route Coordinates from Mapbox Directions API
  const [roadCoordinates, setRoadCoordinates] = useState<[number, number][] | null>(null);

  const primaryDriver = drivers[0];

  // Fetch real road directions whenever route points or navigation leg changes
  useEffect(() => {
    let isMounted = true;

    async function updateRoadRoute() {
      if (activeLeg === 'TO_PICKUP' && primaryDriver && pickup) {
        const result = await fetchMapboxDirections(
          primaryDriver.lng,
          primaryDriver.lat,
          pickup.lng,
          pickup.lat,
          MAPBOX_TOKEN
        );
        if (isMounted && result) {
          setRoadCoordinates(result.coordinates);
        }
      } else if (activeLeg === 'TO_DESTINATION' && primaryDriver && dropoff) {
        const result = await fetchMapboxDirections(
          primaryDriver.lng,
          primaryDriver.lat,
          dropoff.lng,
          dropoff.lat,
          MAPBOX_TOKEN
        );
        if (isMounted && result) {
          setRoadCoordinates(result.coordinates);
        }
      } else if (pickup && dropoff) {
        const result = await fetchMapboxDirections(
          pickup.lng,
          pickup.lat,
          dropoff.lng,
          dropoff.lat,
          MAPBOX_TOKEN
        );
        if (isMounted && result) {
          setRoadCoordinates(result.coordinates);
        }
      } else {
        if (isMounted) setRoadCoordinates(null);
      }
    }

    updateRoadRoute();

    return () => {
      isMounted = false;
    };
  }, [activeLeg, primaryDriver?.lat, primaryDriver?.lng, pickup?.lat, pickup?.lng, dropoff?.lat, dropoff?.lng]);

  // LIVE CAMERA TRACKING: Automatically follow the live moving car during active trip legs!
  useEffect(() => {
    if (!mapRef.current) return;

    if ((activeLeg === 'TO_PICKUP' || activeLeg === 'TO_DESTINATION') && primaryDriver && followCar) {
      mapRef.current.easeTo({
        center: [primaryDriver.lng, primaryDriver.lat],
        zoom: 16,
        pitch: 45,
        bearing: primaryDriver.heading || 0,
        duration: 800,
      });
    } else if (activeLeg === 'NONE' && pickup && !activePinMode) {
      mapRef.current.flyTo({
        center: [pickup.lng, pickup.lat],
        zoom: 14.5,
        duration: 1000,
      });
    }
  }, [activeLeg, primaryDriver?.lat, primaryDriver?.lng, primaryDriver?.heading, followCar, activePinMode]);

  // Recenter explicitly on car
  const handleCenterOnCar = () => {
    if (!mapRef.current || !primaryDriver) return;
    setFollowCar(true);
    mapRef.current.flyTo({
      center: [primaryDriver.lng, primaryDriver.lat],
      zoom: 16.5,
      pitch: 50,
      bearing: primaryDriver.heading || 0,
      duration: 1000,
    });
  };

  // GeoJSON LineString for Real Street Road Route
  const routeGeoJSON = useMemo(() => {
    if (!roadCoordinates || roadCoordinates.length < 2) return null;

    const data: GeoJSON.Feature<LineString> = {
      type: 'Feature',
      properties: { leg: activeLeg },
      geometry: {
        type: 'LineString',
        coordinates: roadCoordinates,
      },
    };
    return data;
  }, [roadCoordinates, activeLeg]);

  if (!MAPBOX_TOKEN || MAPBOX_TOKEN.includes('yourusername')) {
    return (
      <div className={`relative flex flex-col items-center justify-center bg-slate-900 text-white p-8 rounded-2xl overflow-hidden ${className}`}>
        <div className="absolute inset-0 opacity-20 bg-[radial-gradient(#3b82f6_1px,transparent_1px)] [background-size:24px_24px]"></div>
        <div className="relative z-10 max-w-md text-center p-6 bg-slate-800/80 backdrop-blur-md rounded-2xl border border-slate-700 shadow-2xl">
          <div className="w-12 h-12 mx-auto mb-4 rounded-xl bg-blue-500/20 text-blue-400 flex items-center justify-center">
            <AlertCircle className="w-6 h-6" />
          </div>
          <h3 className="text-lg font-bold mb-2">Mapbox API Key Required</h3>
          <p className="text-sm text-slate-300 mb-4 leading-relaxed">
            Please paste your free Mapbox Public Token into <code className="text-blue-300 bg-slate-900 px-2 py-0.5 rounded font-mono text-xs">frontend/.env</code> as:
          </p>
          <div className="p-3 bg-slate-950 rounded-lg text-xs font-mono text-left text-slate-400 overflow-x-auto select-all mb-4 border border-slate-800">
            NEXT_PUBLIC_MAPBOX_TOKEN=pk.eyJ1I...
          </div>
        </div>
      </div>
    );
  }

  const targetWaypoint = activeLeg === 'TO_PICKUP' ? pickup : activeLeg === 'TO_DESTINATION' ? dropoff : null;

  return (
    <div
      className={`relative ${className} ${activePinMode ? 'pin-picking-active' : ''}`}
      onMouseMove={(e) => {
        if (activePinMode) {
          setMousePos({ x: e.clientX, y: e.clientY });
        }
      }}
      onMouseLeave={() => setMousePos(null)}
    >
      {/* FontAwesome location-dot FLOATING MOUSE CURSOR */}
      {activePinMode && mousePos && (
        <div
          className="pointer-events-none fixed z-[9999] transition-none flex flex-col items-center select-none"
          style={{
            left: `${mousePos.x}px`,
            top: `${mousePos.y}px`,
            transform: 'translate(-50%, -100%)',
          }}
        >
          {/* Label Tooltip */}
          <div className="px-2.5 py-0.5 bg-slate-950/90 text-white font-extrabold text-[10px] rounded-full shadow-2xl border border-[rgb(116,192,252)]/60 mb-1 tracking-tight flex items-center gap-1.5 whitespace-nowrap">
            <span className="w-2 h-2 rounded-full bg-[rgb(116,192,252)] animate-ping"></span>
            <span>{activePinMode === 'PICKUP' ? 'Set Pickup Here' : 'Set Dropoff Here'}</span>
          </div>

          {/* FontAwesome Location Dot SVG */}
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 384 512"
            className="w-10 h-10 drop-shadow-[0_8px_16px_rgba(0,0,0,0.6)] animate-bounce"
          >
            <g transform="scale(-1, 1) translate(-384, 0)">
              <path
                fill="rgb(116, 192, 252)"
                stroke="#FFFFFF"
                strokeWidth="20"
                strokeLinejoin="round"
                d="M215.7 499.2C267 435 384 279.4 384 192C384 86 298 0 192 0S0 86 0 192c0 87.4 117 243 168.3 307.2c12.3 15.3 35.1 15.3 47.4 0z"
              />
              <circle cx="192" cy="192" r="64" fill="#FFFFFF" />
            </g>
          </svg>

          {/* Laser Target Tip */}
          <div className="w-2 h-2 rounded-full bg-white border border-[rgb(116,192,252)] -mt-1 shadow-md"></div>
        </div>
      )}

      {/* Floating Interactive Pin Helper Banner */}
      {activePinMode && (
        <div className="absolute top-6 left-1/2 -translate-x-1/2 z-30 px-6 py-3 bg-slate-950/90 backdrop-blur-md text-white rounded-full border border-[rgb(116,192,252)]/50 shadow-2xl flex items-center gap-3 animate-bounce select-none pointer-events-none">
          <div className="w-8 h-8 rounded-full flex items-center justify-center text-white bg-[rgb(116,192,252)] shadow-md">
            <MapPin className="w-4 h-4 text-slate-950" />
          </div>
          <div className="text-xs">
            <span className="font-extrabold text-sm block">
              {activePinMode === 'PICKUP' ? 'Drop Pickup Location Pin' : 'Drop Destination Location Pin'}
            </span>
            <span className="text-[11px] text-blue-200">
              Click exact road point on map to place pin
            </span>
          </div>
        </div>
      )}

      {/* Floating Recenter Action Button (Only on Rider screen when tracking chauffeur) */}
      {showTrackingBadge && (activeLeg === 'TO_PICKUP' || activeLeg === 'TO_DESTINATION') && primaryDriver && (
        <div className="absolute top-6 right-6 z-30 flex items-center gap-2">
          <button
            onClick={handleCenterOnCar}
            className={`px-4 py-2 rounded-full font-bold text-xs flex items-center gap-2 shadow-xl backdrop-blur-md transition-all border ${
              followCar
                ? 'bg-[#276EF1] text-white border-blue-400 shadow-blue-500/25'
                : 'bg-slate-950/90 text-slate-200 border-slate-700 hover:bg-slate-900'
            }`}
          >
            <LocateFixed className="w-3.5 h-3.5 text-emerald-400" />
            <span>{followCar ? 'Tracking Chauffeur' : 'Recenter on Car'}</span>
          </button>
        </div>
      )}

      <Map
        ref={mapRef}
        {...viewState}
        onMove={(evt) => setViewState(evt.viewState)}
        onDragStart={() => {
          if (activeLeg === 'TO_PICKUP' || activeLeg === 'TO_DESTINATION') {
            setFollowCar(false);
          }
        }}
        mapboxAccessToken={MAPBOX_TOKEN}
        style={{ width: '100%', height: '100%' }}
        mapStyle="mapbox://styles/mapbox/navigation-night-v1"
        interactive={interactive}
        onClick={(e) => {
          if (onMapClick) {
            onMapClick({ lat: e.lngLat.lat, lng: e.lngLat.lng });
          }
        }}
      >
        <NavigationControl position="bottom-right" />

        {/* Real Street Route Polyline Layer */}
        {routeGeoJSON && (
          <Source id="route-source" type="geojson" data={routeGeoJSON}>
            {/* Outer Route Glow */}
            <Layer
              id="route-glow"
              type="line"
              layout={{ 'line-join': 'round', 'line-cap': 'round' }}
              paint={{
                'line-color': activeLeg === 'TO_PICKUP' ? '#00E5FF' : '#276EF1',
                'line-width': 9,
                'line-opacity': 0.45,
              }}
            />
            {/* Inner Street Route */}
            <Layer
              id="route-line"
              type="line"
              layout={{ 'line-join': 'round', 'line-cap': 'round' }}
              paint={{
                'line-color': activeLeg === 'TO_PICKUP' ? '#00E5FF' : '#276EF1',
                'line-width': 5,
                'line-opacity': 0.95,
              }}
            />
          </Source>
        )}

        {/* 20m Proximity Radar Circle */}
        {targetWaypoint && (
          <Marker longitude={targetWaypoint.lng} latitude={targetWaypoint.lat} anchor="center">
            <div className="relative flex items-center justify-center pointer-events-none">
              <div className="w-20 h-20 rounded-full border border-emerald-400/40 bg-emerald-500/10 animate-ping absolute"></div>
              <div className="w-12 h-12 rounded-full border border-emerald-400/60 bg-emerald-500/15 animate-pulse absolute"></div>
              <span className="text-[9px] font-black text-emerald-400 bg-slate-950/80 px-1.5 py-0.5 rounded-full border border-emerald-500/30 uppercase tracking-widest -mt-10">
                20m Zone
              </span>
            </div>
          </Marker>
        )}

        {/* Draggable Pickup Marker (Hidden once rider is picked up and in transit to destination) */}
        {pickup && activeLeg !== 'TO_DESTINATION' && (
          <Marker
            longitude={pickup.lng}
            latitude={pickup.lat}
            anchor="bottom"
            draggable={interactive && activeLeg === 'NONE'}
            onDragEnd={(e) => {
              if (onPickupDrag) {
                onPickupDrag({ lat: e.lngLat.lat, lng: e.lngLat.lng });
              }
            }}
          >
            <div className="flex flex-col items-center group cursor-grab active:cursor-grabbing z-10">
              <div className="px-2.5 py-1 bg-white text-slate-900 font-extrabold text-[11px] rounded-full shadow-lg border border-slate-200 mb-1 tracking-tight flex items-center gap-1 group-hover:scale-105 transition-transform">
                <span className="w-2 h-2 rounded-full bg-[rgb(116,192,252)]"></span>
                {pickup.label || 'Pickup'}
              </div>
              <div className="relative">
                <div className="w-8 h-8 rounded-full bg-[rgb(116,192,252)]/30 flex items-center justify-center animate-pulse-ring absolute inset-0"></div>
                <div className="w-8 h-8 rounded-full bg-[rgb(116,192,252)] text-slate-950 flex items-center justify-center shadow-lg border-2 border-white">
                  <MapPin className="w-4 h-4 fill-slate-950" />
                </div>
              </div>
            </div>
          </Marker>
        )}

        {/* Draggable Dropoff Marker */}
        {dropoff && (
          <Marker
            longitude={dropoff.lng}
            latitude={dropoff.lat}
            anchor="bottom"
            draggable={interactive && activeLeg === 'NONE'}
            onDragEnd={(e) => {
              if (onDropoffDrag) {
                onDropoffDrag({ lat: e.lngLat.lat, lng: e.lngLat.lng });
              }
            }}
          >
            <div className="flex flex-col items-center group cursor-grab active:cursor-grabbing z-10">
              <div className="px-2.5 py-1 bg-slate-900 text-white font-extrabold text-[11px] rounded-full shadow-lg border border-slate-700 mb-1 tracking-tight flex items-center gap-1 group-hover:scale-105 transition-transform">
                <span className="w-2 h-2 rounded-full bg-[rgb(116,192,252)]"></span>
                {dropoff.label || 'Destination'}
              </div>
              <div className="w-8 h-8 rounded-full bg-slate-900 text-[rgb(116,192,252)] flex items-center justify-center shadow-lg border-2 border-white">
                <Navigation className="w-4 h-4" />
              </div>
            </div>
          </Marker>
        )}

        {/* Driver Fleet Markers with Live Heading Orientation */}
        {drivers.map((drv, idx) => (
          <Marker
            key={drv.id || idx}
            longitude={drv.lng}
            latitude={drv.lat}
            anchor="center"
          >
            <div
              className="relative transition-transform duration-700 ease-out cursor-pointer z-20"
              style={{
                transform: `rotate(${drv.heading || 0}deg)`,
              }}
              title={drv.label || 'Chauffeur Vehicle'}
            >
              <div className="absolute -inset-2 rounded-full bg-emerald-500/40 animate-ping"></div>
              <div className="w-10 h-10 rounded-full bg-slate-950 border-2 border-emerald-400 text-white flex items-center justify-center shadow-2xl ring-4 ring-emerald-500/20">
                <Car className="w-5 h-5 text-emerald-400" />
              </div>
            </div>
          </Marker>
        ))}
      </Map>
    </div>
  );
}
