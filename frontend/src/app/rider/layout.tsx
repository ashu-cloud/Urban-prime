import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Customer App',
  description: 'Urban Prime Customer & Chauffeur Booking Portal',
};

export default function RiderLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
