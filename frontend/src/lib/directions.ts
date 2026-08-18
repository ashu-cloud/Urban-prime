/**
 * Mapbox Directions API Integration for exact road routing and real driving distance/ETA.
 */

export interface RouteResult {
  coordinates: [number, number][];
  distanceMeters: number;
  distanceKm: number;
  durationMinutes: number;
  durationFormatted: string;
}

export async function fetchMapboxDirections(
  startLng: number,
  startLat: number,
  endLng: number,
  endLat: number,
  token?: string
): Promise<RouteResult | null> {
  const mapboxToken = token || process.env.NEXT_PUBLIC_MAPBOX_TOKEN;
  if (!mapboxToken || mapboxToken.includes('yourusername')) return null;

  try {
    const url = `https://api.mapbox.com/directions/v5/mapbox/driving/${startLng},${startLat};${endLng},${endLat}?geometries=geojson&overview=full&access_token=${mapboxToken}`;
    const res = await fetch(url);
    if (!res.ok) return null;

    const data = await res.json();
    if (!data.routes || data.routes.length === 0) return null;

    const primaryRoute = data.routes[0];
    const distanceMeters = primaryRoute.distance;
    const distanceKm = Number((distanceMeters / 1000).toFixed(1));
    const durationMinutes = Math.max(1, Math.round(primaryRoute.duration / 60));

    return {
      coordinates: primaryRoute.geometry.coordinates,
      distanceMeters,
      distanceKm,
      durationMinutes,
      durationFormatted: `${durationMinutes} min`,
    };
  } catch (err) {
    console.error('Failed to fetch real road directions from Mapbox:', err);
    return null;
  }
}
