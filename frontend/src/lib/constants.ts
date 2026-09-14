export interface VehicleTier {
  id: 'SEDAN' | 'SUV' | 'PREMIUM' | 'BIKE';
  name: string;
  subtitle: string;
  basePrice: number;
  perKm: number;
  capacity: number;
  icon: string;
  popular?: boolean;
}

export const VEHICLE_TIERS: VehicleTier[] = [
  {
    id: 'PREMIUM',
    name: 'Prime Black',
    subtitle: 'Luxury Electric Sedan',
    basePrice: 25.00,
    perKm: 4.50,
    capacity: 4,
    icon: '⚡',
    popular: true,
  },
  {
    id: 'SEDAN',
    name: 'Urban Comfort',
    subtitle: 'Standard 4-Door Hybrid',
    basePrice: 15.00,
    perKm: 3.20,
    capacity: 4,
    icon: '🚘',
  },
  {
    id: 'SUV',
    name: 'Executive SUV',
    subtitle: 'Spacious 6-Passenger SUV',
    basePrice: 35.00,
    perKm: 5.80,
    capacity: 6,
    icon: '🚙',
  },
  {
    id: 'BIKE',
    name: 'Prime Express',
    subtitle: 'Rapid Solo Courier',
    basePrice: 8.00,
    perKm: 2.00,
    capacity: 1,
    icon: '🛵',
  },
];
