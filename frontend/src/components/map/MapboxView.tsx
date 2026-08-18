'use client';

import React, { useEffect, useRef, useState, useMemo } from 'react';
import Map, { Marker, Source, Layer, NavigationControl, MapRef } from 'react-map-gl/mapbox';
import { MapPin, Navigation, Car, AlertCircle } from 'lucide-react';
import type { LineString } from 'geojson';

export interface MarkerLocation {
  lat: number;
  lng: number;
  label?: string;
  type?: 'pickup' | 'dropoff' | 'driver';
  heading?: number;
  id?: string;
}

interface MapboxViewProps {
  center?: [number, number]; // [lng, lat]
  zoom?: number;
  pickup?: MarkerLocation | null;
  dropoff?: MarkerLocation | null;
  drivers?: MarkerLocation[];
  interactive?: boolean;
  className?: string;
  onMapClick?: (coords: { lat: number; lng: number }) => void;
}

const MAPBOX_TOKEN = process.env.NEXT_PUBLIC_MAPBOX_TOKEN;

export default function MapboxView({
  center = [-73.9851, 40.7484], // Default NYC Empire State
  zoom = 13.5,
  pickup,
  dropoff,
  drivers = [],
  interactive = true,
  className = 'w-full h-full',
  onMapClick,
}: MapboxViewProps) {
  const mapRef = useRef<MapRef | null>(null);
  const [viewState, setViewState] = useState({
    longitude: center[0],
    latitude: center[1],
    zoom: zoom,
    pitch: 30,
    bearing: 0,
  });

  // Fit bounds or fly to center when pickup/center changes
  useEffect(() => {
    if (pickup && mapRef.current) {
      mapRef.current.flyTo({
        center: [pickup.lng, pickup.lat],
        zoom: 14.5,
        duration: 1500,
      });
    }
  }, [pickup]);

  // Route Polyline GeoJSON
  const routeGeoJSON = useMemo(() => {
    if (!pickup || !dropoff) return null;
    const data: GeoJSON.Feature<LineString> = {
      type: 'Feature',
      properties: {},
      geometry: {
        type: 'LineString',
        coordinates: [
          [pickup.lng, pickup.lat],
          // Add realistic curvature waypoint
          [(pickup.lng + dropoff.lng) / 2 + 0.003, (pickup.lat + dropoff.lat) / 2 - 0.002],
          [dropoff.lng, dropoff.lat],
        ],
      },
    };
    return data;
  }, [pickup, dropoff]);

  // Handle Token Missing State with a graceful UI
  if (!MAPBOX_TOKEN || MAPBOX_TOKEN.includes('yourusername')) {
    return (
      <div className={`relative flex flex-col items-center justify-center bg-slate-900 text-white p-8 rounded-2xl overflow-hidden ${className}`}>
        {/* Stylized background grid simulating a map */}
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
          <p className="text-xs text-slate-400">
            Once saved, restart Next.js or refresh the page to load high-resolution vector maps.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className={`relative ${className}`}>
      <Map
        ref={mapRef}
        {...viewState}
        onMove={(evt) => setViewState(evt.viewState)}
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

        {/* Route Polyline Layer */}
        {routeGeoJSON && (
          <Source id="route-source" type="geojson" data={routeGeoJSON}>
            {/* Outer Glow */}
            <Layer
              id="route-glow"
              type="line"
              layout={{ 'line-join': 'round', 'line-cap': 'round' }}
              paint={{
                'line-color': '#276EF1',
                'line-width': 8,
                'line-opacity': 0.35,
              }}
            />
            {/* Core Route */}
            <Layer
              id="route-line"
              type="line"
              layout={{ 'line-join': 'round', 'line-cap': 'round' }}
              paint={{
                'line-color': '#276EF1',
                'line-width': 4,
                'line-opacity': 0.95,
              }}
            />
          </Source>
        )}

        {/* Pickup Marker */}
        {pickup && (
          <Marker longitude={pickup.lng} latitude={pickup.lat} anchor="bottom">
            <div className="flex flex-col items-center group cursor-pointer">
              <div className="px-2.5 py-1 bg-white text-slate-900 font-bold text-[11px] rounded-full shadow-lg border border-slate-200 mb-1 tracking-tight">
                {pickup.label || 'Pickup'}
              </div>
              <div className="relative">
                <div className="w-8 h-8 rounded-full bg-blue-600/30 flex items-center justify-center animate-pulse-ring absolute inset-0"></div>
                <div className="w-8 h-8 rounded-full bg-[#276EF1] text-white flex items-center justify-center shadow-lg border-2 border-white">
                  <MapPin className="w-4 h-4 fill-white" />
                </div>
              </div>
            </div>
          </Marker>
        )}

        {/* Dropoff Marker */}
        {dropoff && (
          <Marker longitude={dropoff.lng} latitude={dropoff.lat} anchor="bottom">
            <div className="flex flex-col items-center group cursor-pointer">
              <div className="px-2.5 py-1 bg-slate-900 text-white font-bold text-[11px] rounded-full shadow-lg border border-slate-700 mb-1 tracking-tight">
                {dropoff.label || 'Destination'}
              </div>
              <div className="w-8 h-8 rounded-full bg-slate-900 text-white flex items-center justify-center shadow-lg border-2 border-white">
                <Navigation className="w-4 h-4" />
              </div>
            </div>
          </Marker>
        )}

        {/* Driver Fleet Markers */}
        {drivers.map((drv, idx) => (
          <Marker
            key={drv.id || idx}
            longitude={drv.lng}
            latitude={drv.lat}
            anchor="center"
          >
            <div
              className="relative transition-transform duration-700 ease-out cursor-pointer"
              style={{
                transform: `rotate(${drv.heading || 0}deg)`,
              }}
              title={drv.label || 'Driver'}
            >
              {/* Radar pulse for active online drivers */}
              <div className="absolute -inset-1.5 rounded-full bg-emerald-500/25 animate-ping"></div>
              
              <div className="w-9 h-9 rounded-full bg-slate-950 border-2 border-emerald-400 text-white flex items-center justify-center shadow-xl">
                <Car className="w-5 h-5 text-emerald-400" />
              </div>
            </div>
          </Marker>
        ))}
      </Map>
    </div>
  );
}
