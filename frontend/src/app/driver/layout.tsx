import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Driver App',
  description: 'Urban Prime Chauffeur Partner Cockpit',
};

export default function DriverLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
