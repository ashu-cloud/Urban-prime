/**
 * Progressive Platform Fee Calculation System
 *
 * Tiered Platform Commission (15% to 35% based on base trip fare):
 * - Base Fare <= $20: 15% Platform Fee
 * - Base Fare $20 < F <= $40: 20% Platform Fee
 * - Base Fare $40 < F <= $70: 25% Platform Fee
 * - Base Fare $70 < F <= $100: 30% Platform Fee
 * - Base Fare > $100: 35% Platform Fee
 *
 * NOTE: 100% of Chauffeur Tips are exempt from platform fees and go directly to the driver.
 */

export interface FareBreakdown {
  grossBaseFare: number;
  feeRate: number;
  feePercentage: number;
  platformFee: number;
  driverNetFare: number;
  tipAmount: number;
  totalRiderCharged: number;
  totalDriverTakeHome: number;
}

export function calculatePlatformFee(baseFare: number, tip: number = 0): FareBreakdown {
  const fare = Math.max(0, Number(baseFare) || 0);
  const tipAmount = Math.max(0, Number(tip) || 0);

  let feeRate = 0.15; // Baseline 15%

  if (fare <= 20) {
    feeRate = 0.15;
  } else if (fare <= 40) {
    feeRate = 0.20;
  } else if (fare <= 70) {
    feeRate = 0.25;
  } else if (fare <= 100) {
    feeRate = 0.30;
  } else {
    feeRate = 0.35;
  }

  const platformFee = Number((fare * feeRate).toFixed(2));
  const driverNetFare = Number((fare - platformFee).toFixed(2));
  const feePercentage = Math.round(feeRate * 100);
  const totalRiderCharged = Number((fare + tipAmount).toFixed(2));
  const totalDriverTakeHome = Number((driverNetFare + tipAmount).toFixed(2));

  return {
    grossBaseFare: fare,
    feeRate,
    feePercentage,
    platformFee,
    driverNetFare,
    tipAmount,
    totalRiderCharged,
    totalDriverTakeHome,
  };
}
