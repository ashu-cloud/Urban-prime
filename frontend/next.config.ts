import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: '/auth/:path*',
        destination: 'https://urbanprime-auth.onrender.com/auth/:path*',
      },
      {
        source: '/trip/:path*',
        destination: 'https://urbanprime-trip.onrender.com/trip/:path*',
      },
      {
        source: '/driver/:path*',
        destination: 'https://urbanprime-driver.onrender.com/driver/:path*',
      },
      {
        source: '/location/:path*',
        destination: 'https://urbanprime-location.onrender.com/location/:path*',
      },
      {
        source: '/payment/:path*',
        destination: 'https://urbanprime-payment.onrender.com/payment/:path*',
      },
    ];
  },
};

export default nextConfig;
