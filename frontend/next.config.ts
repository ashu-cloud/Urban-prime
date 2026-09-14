import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      // ─── Auth Service ────────────────────────────────────
      {
        source: '/auth/:path*',
        destination: 'https://urbanprime-auth.onrender.com/auth/:path*',
      },
      // ─── Trip Service ────────────────────────────────────
      {
        source: '/api/v1/trips',
        destination: 'https://urbanprime-trip.onrender.com/api/v1/trips',
      },
      {
        source: '/api/v1/trips/:path*',
        destination: 'https://urbanprime-trip.onrender.com/api/v1/trips/:path*',
      },
      {
        source: '/trips',
        destination: 'https://urbanprime-trip.onrender.com/trips',
      },
      {
        source: '/trips/:path*',
        destination: 'https://urbanprime-trip.onrender.com/trips/:path*',
      },
      // ─── Driver Service ──────────────────────────────────
      {
        source: '/api/v1/drivers',
        destination: 'https://urbanprime-driver.onrender.com/api/v1/drivers',
      },
      {
        source: '/api/v1/drivers/:path*',
        destination: 'https://urbanprime-driver.onrender.com/api/v1/drivers/:path*',
      },
      {
        source: '/api/v1/dispatch/:path*',
        destination: 'https://urbanprime-driver.onrender.com/api/v1/dispatch/:path*',
      },
      {
        source: '/drivers',
        destination: 'https://urbanprime-driver.onrender.com/drivers',
      },
      {
        source: '/drivers/:path*',
        destination: 'https://urbanprime-driver.onrender.com/drivers/:path*',
      },
      // ─── Location Service ─────────────────────────────────
      {
        source: '/api/v1/location',
        destination: 'https://urbanprime-location.onrender.com/api/v1/location',
      },
      {
        source: '/api/v1/location/:path*',
        destination: 'https://urbanprime-location.onrender.com/api/v1/location/:path*',
      },
      // ─── Payment Service ──────────────────────────────────
      {
        source: '/api/v1/payments',
        destination: 'https://urbanprime-payment.onrender.com/api/v1/payments',
      },
      {
        source: '/api/v1/payments/:path*',
        destination: 'https://urbanprime-payment.onrender.com/api/v1/payments/:path*',
      },
      {
        source: '/payment/:path*',
        destination: 'https://urbanprime-payment.onrender.com/payment/:path*',
      },
    ];
  },
};

export default nextConfig;
