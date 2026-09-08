import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Urban Prime",
  description: "Next-Generation Intelligent Urban Mobility Platform",
  icons: {
    icon: [
      { url: "/urban-prime-logo.svg", type: "image/svg+xml" },
    ],
    shortcut: "/urban-prime-logo.svg",
    apple: "/urban-prime-logo.svg",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <head>
        <link rel="icon" href="/urban-prime-logo.svg" type="image/svg+xml" />
      </head>
      <body className="min-h-full flex flex-col">{children}</body>
    </html>
  );
}
