import { calculatePlatformFee } from './fare.ts';

const cases = [
  [0, 0, 0.15],
  [20, 0, 0.15],
  [20.01, 5, 0.2],
  [40, 0, 0.2],
  [70, 0, 0.25],
  [100, 0, 0.3],
  [150, 10, 0.35],
];

for (const [fare, tip, rate] of cases) {
  const result = calculatePlatformFee(fare, tip);
  if (result.feeRate !== rate) {
    throw new Error(`fare ${fare}: feeRate ${result.feeRate} want ${rate}`);
  }
  if (result.platformFee + result.driverNetFare !== result.grossBaseFare) {
    throw new Error(`fare ${fare}: fee + driver net must equal gross`);
  }
  if (result.totalDriverTakeHome !== Number((result.driverNetFare + tip).toFixed(2))) {
    throw new Error(`fare ${fare}: tip must go 100% to driver`);
  }
}

const negative = calculatePlatformFee(-40, -8);
if (negative.grossBaseFare !== 0 || negative.tipAmount !== 0) {
  throw new Error('negative inputs must clamp to zero');
}

console.log('fare tests passed');
