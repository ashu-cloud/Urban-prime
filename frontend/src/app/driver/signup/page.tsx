'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api, DriverOnboardingRequest, getStoredDriverSession } from '@/lib/api';
import {
  Compass,
  Car,
  ShieldCheck,
  ArrowRight,
  ArrowLeft,
  CheckCircle2,
  Lock,
  Mail,
  Phone,
  User,
  Sparkles,
  Award,
  FileText,
  Zap,
  Check,
} from 'lucide-react';

const VEHICLE_TIERS = [
  {
    id: 'PREMIUM',
    name: 'Prime Black',
    subtitle: 'Luxury Electric / Executive Sedan',
    icon: '⚡',
    examples: 'Tesla Model S, Mercedes EQE, BMW i4, Lucid Air',
  },
  {
    id: 'SEDAN',
    name: 'Urban Comfort',
    subtitle: 'Standard 4-Door Hybrid / Sedan',
    icon: '🚘',
    examples: 'Toyota Camry, Honda Accord, Hyundai Ioniq',
  },
  {
    id: 'SUV',
    name: 'Executive SUV',
    subtitle: 'Spacious 6+ Passenger SUV',
    icon: '🚙',
    examples: 'Cadillac Escalade, Chevy Suburban, Lincoln Navigator',
  },
  {
    id: 'BIKE',
    name: 'Prime Express',
    subtitle: 'Rapid Solo Courier',
    icon: '🛵',
    examples: 'Zero Electric, BMW CE 04, Ducati Scrambler',
  },
];

export default function DriverSignupPage() {
  const router = useRouter();
  const [currentStep, setCurrentStep] = useState<1 | 2 | 3>(1);
  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    const session = getStoredDriverSession();
    if (session && session.role === 'DRIVER') {
      router.replace('/driver');
    }
  }, [router]);

  // Form State
  const [formData, setFormData] = useState<DriverOnboardingRequest>({
    fullName: '',
    email: '',
    phone: '',
    password: '',
    vehicleMake: 'Tesla',
    vehicleModel: 'Model S',
    vehicleYear: '2025',
    vehicleColor: 'Obsidian Black',
    vehiclePlate: 'NY-7890',
    vehicleType: 'PREMIUM',
    licenseNumber: 'CDL-NY-8947291',
  });

  const handleChange = (field: keyof DriverOnboardingRequest, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    setErrorMessage('');
  };

  const handleNextStep = (e: React.FormEvent) => {
    e.preventDefault();

    if (currentStep === 1) {
      if (!formData.fullName || !formData.email || !formData.phone || !formData.password) {
        setErrorMessage('Please fill in all personal credentials to continue.');
        return;
      }
      setCurrentStep(2);
    } else if (currentStep === 2) {
      if (!formData.vehicleMake || !formData.vehicleModel || !formData.vehiclePlate) {
        setErrorMessage('Please provide complete vehicle make, model, and plate number.');
        return;
      }
      setCurrentStep(3);
    } else if (currentStep === 3) {
      if (!formData.licenseNumber) {
        setErrorMessage('Commercial driver license number is required for verification.');
        return;
      }
      handleSubmit();
    }
  };

  const handleSubmit = async () => {
    setIsLoading(true);
    setErrorMessage('');
    try {
      await api.registerDriver(formData);
      router.replace('/driver');
    } catch (err: any) {
      setErrorMessage(err.message || 'Failed to complete registration. Please try again.');
      setIsLoading(false);
    }
  };

  // Quick Autofill for Fast Demo Review
  const handleQuickDemoFill = () => {
    setFormData({
      fullName: 'Marcus Sterling',
      email: 'marcus.sterling@driver.urbanprime.com',
      phone: '+1 (555) 894-2091',
      password: 'ChauffeurExecutive2026!',
      vehicleMake: 'Tesla',
      vehicleModel: 'Model S',
      vehicleYear: '2025',
      vehicleColor: 'Obsidian Black',
      vehiclePlate: 'NY-7890',
      vehicleType: 'PREMIUM',
      licenseNumber: 'CDL-NY-8947291',
    });
  };

  return (
    <div className="min-h-screen bg-[#FCF9F8] flex flex-col justify-between select-none">
      {/* Top Header */}
      <header className="h-[72px] px-8 flex items-center justify-between border-b border-[#DCD9D9] bg-white">
        <div className="flex items-center gap-2.5">
          <div className="w-10 h-10 rounded-xl bg-[#008A5E] flex items-center justify-center text-white shadow-md shadow-emerald-500/20">
            <Compass className="w-5 h-5" />
          </div>
          <div>
            <span className="font-extrabold text-xl tracking-tight text-[#1F1F1F]">
              URBAN<span className="text-[#008A5E]">PRIME</span>
            </span>
            <span className="block text-[10px] tracking-widest uppercase font-bold text-emerald-600 -mt-1">
              Partner Onboarding
            </span>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={handleQuickDemoFill}
            className="px-3.5 py-1.5 rounded-full bg-emerald-50 hover:bg-emerald-100 text-[#008A5E] border border-emerald-200 text-xs font-bold transition-all flex items-center gap-1.5 shadow-2xs cursor-pointer"
          >
            <Sparkles className="w-3.5 h-3.5" />
            Auto-Fill Demo
          </button>
          <Link
            href="/driver/login"
            className="text-xs font-bold text-slate-600 hover:text-slate-900 transition-colors"
          >
            Already a Partner? <span className="text-[#008A5E]">Sign In</span>
          </Link>
        </div>
      </header>

      {/* Main Wizard Container */}
      <main className="flex-1 flex items-center justify-center p-6 py-10">
        <div className="w-full max-w-xl bg-white rounded-[28px] border border-[#DCD9D9] p-8 shadow-2xl space-y-6">
          {/* Wizard Step Progress Bar */}
          <div className="space-y-3">
            <div className="flex items-center justify-between text-xs font-extrabold">
              <span
                className={`flex items-center gap-1.5 ${
                  currentStep >= 1 ? 'text-[#276EF1]' : 'text-slate-400'
                }`}
              >
                <span className="w-5 h-5 rounded-full bg-[#E7F0FF] flex items-center justify-center text-[10px]">
                  1
                </span>
                Credentials
              </span>
              <span
                className={`flex items-center gap-1.5 ${
                  currentStep >= 2 ? 'text-[#276EF1]' : 'text-slate-400'
                }`}
              >
                <span className="w-5 h-5 rounded-full bg-[#E7F0FF] flex items-center justify-center text-[10px]">
                  2
                </span>
                Vehicle & Fleet
              </span>
              <span
                className={`flex items-center gap-1.5 ${
                  currentStep === 3 ? 'text-[#276EF1]' : 'text-slate-400'
                }`}
              >
                <span className="w-5 h-5 rounded-full bg-[#E7F0FF] flex items-center justify-center text-[10px]">
                  3
                </span>
                Verification
              </span>
            </div>

            {/* Visual Progress Bar */}
            <div className="h-1.5 w-full bg-slate-100 rounded-full overflow-hidden">
              <div
                className="h-full bg-[#276EF1] transition-all duration-300 rounded-full"
                style={{ width: `${(currentStep / 3) * 100}%` }}
              ></div>
            </div>
          </div>

          {/* Form Header */}
          <div className="text-center space-y-1">
            <h1 className="text-2xl font-black text-[#1F1F1F] tracking-tight">
              {currentStep === 1 && 'Create Chauffeur Partner Account'}
              {currentStep === 2 && 'Register Fleet Vehicle & Tier'}
              {currentStep === 3 && 'Licensing & Document Verification'}
            </h1>
            <p className="text-xs text-slate-500">
              {currentStep === 1 && 'Enter your personal contact details to access high-value dispatches'}
              {currentStep === 2 && 'Tell us about the executive vehicle you will be driving'}
              {currentStep === 3 && 'Commercial licensing verification for instant activation'}
            </p>
          </div>

          {/* Error Message */}
          {errorMessage && (
            <div className="p-3.5 bg-red-50 border border-red-200 rounded-xl text-xs font-bold text-red-700 animate-pulse">
              {errorMessage}
            </div>
          )}

          <form onSubmit={handleNextStep} className="space-y-4">
            {/* STEP 1: PERSONAL CREDENTIALS */}
            {currentStep === 1 && (
              <div className="space-y-3.5 animate-in fade-in duration-200">
                <div>
                  <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                    Full Legal Name
                  </label>
                  <div className="relative">
                    <User className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                    <input
                      type="text"
                      required
                      value={formData.fullName}
                      onChange={(e) => handleChange('fullName', e.target.value)}
                      placeholder="e.g. Marcus Sterling"
                      className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl pl-10 pr-4 py-3 text-xs font-semibold text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] focus:ring-2 focus:ring-blue-100 transition-all"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                    Work Email Address
                  </label>
                  <div className="relative">
                    <Mail className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                    <input
                      type="email"
                      required
                      value={formData.email}
                      onChange={(e) => handleChange('email', e.target.value)}
                      placeholder="marcus.sterling@driver.urbanprime.com"
                      className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl pl-10 pr-4 py-3 text-xs font-semibold text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] focus:ring-2 focus:ring-blue-100 transition-all"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                      Phone Number
                    </label>
                    <div className="relative">
                      <Phone className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                      <input
                        type="tel"
                        required
                        value={formData.phone}
                        onChange={(e) => handleChange('phone', e.target.value)}
                        placeholder="+1 (555) 000-0000"
                        className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl pl-10 pr-4 py-3 text-xs font-semibold text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] focus:ring-2 focus:ring-blue-100 transition-all"
                      />
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                      Security Password
                    </label>
                    <div className="relative">
                      <Lock className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                      <input
                        type="password"
                        required
                        value={formData.password}
                        onChange={(e) => handleChange('password', e.target.value)}
                        placeholder="••••••••••••"
                        className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl pl-10 pr-4 py-3 text-xs font-semibold text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] focus:ring-2 focus:ring-blue-100 transition-all"
                      />
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* STEP 2: VEHICLE & FLEET CLASSIFICATION */}
            {currentStep === 2 && (
              <div className="space-y-4 animate-in fade-in duration-200">
                {/* Vehicle Tier Selection */}
                <div>
                  <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-2">
                    Select Vehicle Fleet Tier
                  </label>
                  <div className="grid grid-cols-2 gap-2.5">
                    {VEHICLE_TIERS.map((tier) => {
                      const isSelected = formData.vehicleType === tier.id;
                      return (
                        <div
                          key={tier.id}
                          onClick={() => handleChange('vehicleType', tier.id)}
                          className={`p-3 rounded-2xl border cursor-pointer transition-all flex flex-col justify-between ${
                            isSelected
                              ? 'bg-[#E7F0FF] border-[#276EF1] shadow-xs'
                              : 'bg-white border-[#DCD9D9] hover:border-slate-400'
                          }`}
                        >
                          <div className="flex items-center justify-between mb-1">
                            <span className="text-xl">{tier.icon}</span>
                            {isSelected && (
                              <div className="w-4 h-4 rounded-full bg-[#276EF1] text-white flex items-center justify-center">
                                <Check className="w-3 h-3" />
                              </div>
                            )}
                          </div>
                          <div>
                            <h4 className="text-xs font-bold text-[#1F1F1F]">{tier.name}</h4>
                            <p className="text-[10px] text-slate-500 mt-0.5">{tier.subtitle}</p>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>

                {/* Make / Model / Year / Color */}
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                      Vehicle Make & Model
                    </label>
                    <input
                      type="text"
                      required
                      value={`${formData.vehicleMake} ${formData.vehicleModel}`}
                      onChange={(e) => {
                        const parts = e.target.value.split(' ');
                        handleChange('vehicleMake', parts[0] || 'Tesla');
                        handleChange('vehicleModel', parts.slice(1).join(' ') || 'Model S');
                      }}
                      placeholder="e.g. Tesla Model S"
                      className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl px-3.5 py-2.5 text-xs font-semibold text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] transition-all"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                      License Plate Number
                    </label>
                    <input
                      type="text"
                      required
                      value={formData.vehiclePlate}
                      onChange={(e) => handleChange('vehiclePlate', e.target.value.toUpperCase())}
                      placeholder="e.g. NY-7890"
                      className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl px-3.5 py-2.5 text-xs font-mono font-extrabold uppercase text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] transition-all"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                      Exterior Color
                    </label>
                    <input
                      type="text"
                      value={formData.vehicleColor}
                      onChange={(e) => handleChange('vehicleColor', e.target.value)}
                      placeholder="Obsidian Black"
                      className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl px-3.5 py-2.5 text-xs font-semibold text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] transition-all"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                      Model Year
                    </label>
                    <input
                      type="text"
                      value={formData.vehicleYear}
                      onChange={(e) => handleChange('vehicleYear', e.target.value)}
                      placeholder="2025"
                      className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl px-3.5 py-2.5 text-xs font-semibold text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] transition-all"
                    />
                  </div>
                </div>
              </div>
            )}

            {/* STEP 3: LICENSING & VERIFICATION */}
            {currentStep === 3 && (
              <div className="space-y-4 animate-in fade-in duration-200">
                <div>
                  <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 mb-1.5">
                    Commercial Driver License (CDL / TLC Number)
                  </label>
                  <div className="relative">
                    <FileText className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                    <input
                      type="text"
                      required
                      value={formData.licenseNumber}
                      onChange={(e) => handleChange('licenseNumber', e.target.value.toUpperCase())}
                      placeholder="CDL-NY-8947291"
                      className="w-full bg-[#FCF9F8] border border-[#DCD9D9] rounded-xl pl-10 pr-4 py-3 text-xs font-mono font-bold text-[#1F1F1F] focus:outline-none focus:border-[#276EF1] transition-all"
                    />
                  </div>
                </div>

                {/* Consent Checkboxes */}
                <div className="p-4 bg-[#FCF9F8] rounded-2xl border border-[#DCD9D9] space-y-3 text-xs text-slate-700">
                  <div className="flex items-start gap-2.5">
                    <CheckCircle2 className="w-4 h-4 text-emerald-600 flex-shrink-0 mt-0.5" />
                    <p className="leading-snug font-medium">
                      I certify that my vehicle meets Urban Prime safety and vehicle inspection standards.
                    </p>
                  </div>
                  <div className="flex items-start gap-2.5">
                    <CheckCircle2 className="w-4 h-4 text-emerald-600 flex-shrink-0 mt-0.5" />
                    <p className="leading-snug font-medium">
                      I agree to maintain a minimum 4.8★ service rating and uphold executive chauffeur etiquette.
                    </p>
                  </div>
                </div>

                {/* Summary Pill */}
                <div className="p-3 bg-blue-50 border border-blue-200 rounded-xl flex items-center justify-between text-xs font-bold text-[#276EF1]">
                  <span>Tier Assigned: {formData.vehicleType}</span>
                  <span className="font-mono text-[11px] text-slate-700">
                    {formData.vehicleMake} {formData.vehicleModel} ({formData.vehiclePlate})
                  </span>
                </div>
              </div>
            )}

            {/* Navigation Buttons */}
            <div className="flex items-center gap-3 pt-2">
              {currentStep > 1 && (
                <button
                  type="button"
                  onClick={() => setCurrentStep((s) => (s - 1) as any)}
                  className="py-3.5 px-5 bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold text-xs rounded-xl transition-all flex items-center gap-1.5 active:scale-95"
                >
                  <ArrowLeft className="w-4 h-4" />
                  Back
                </button>
              )}

              <button
                type="submit"
                disabled={isLoading}
                className="flex-1 py-3.5 bg-[#276EF1] hover:bg-[#1A54C9] text-white font-extrabold text-xs uppercase tracking-wider rounded-xl transition-all shadow-lg shadow-blue-500/25 active:scale-95 flex items-center justify-center gap-2 cursor-pointer"
              >
                {isLoading ? (
                  <span className="inline-block animate-spin">⟳</span>
                ) : currentStep < 3 ? (
                  <>
                    <span>Continue to Step {currentStep + 1}</span>
                    <ArrowRight className="w-4 h-4" />
                  </>
                ) : (
                  <>
                    <Zap className="w-4 h-4 fill-white" />
                    <span>Complete Onboarding & Go Online</span>
                  </>
                )}
              </button>
            </div>
          </form>
        </div>
      </main>

      {/* Footer */}
      <footer className="h-[60px] px-8 flex items-center justify-between border-t border-[#DCD9D9] bg-white text-xs text-slate-400">
        <div>Urban Prime Partner Network • Executive Mobility OS</div>
        <div>256-Bit Encrypted Partner Gateway</div>
      </footer>
    </div>
  );
}
