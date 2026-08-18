# Urban Prime: Technical Specification & Design System Ground Truth

This document serves as the authoritative reference for the Urban Prime mobility ecosystem, encompassing both the **Rider Desktop App** and the **Driver Partner Cockpit**. Use these specifications to ensure high-fidelity implementation across any development or generation tool.

---

## 1. Brand Identity & Design System: "Urban Prime"

The **Urban Prime** design system is built on principles of clarity, trust, and premium consumer experience. It avoids complex neons and high-density layouts in favor of breathable white space and a primary "Tech Blue" identity.

### 1.1 Color Tokens
- **Primary (Tech Blue)**: `#276EF1` (Used for primary buttons, active states, and brand marks)
- **Primary Container**: `#E7F0FF` (Light blue backgrounds for subtle accents)
- **Surface**: `#FCF9F8` (Main app background)
- **Surface-Dim**: `#DCD9D9` (Secondary backgrounds or borders)
- **Text-On-Surface**: `#1F1F1F` (High-contrast body and heading text)
- **Success (Online/Acceptance)**: `#008A5E` (Used for the "Go Online" toggle and high ratings)

### 1.2 Typography (Font: Inter)
- **Display/Brand**: `font-headline-md` (Bold, tracking-tight for logo and major headers)
- **Headlines**: `text-headline-sm` / `font-bold` (Card titles and primary navigation)
- **Body**: `text-body-md` / `font-normal` (Standard info and descriptions)
- **Labels**: `text-label-md` / `uppercase font-semibold` (Secondary metadata and status tags)

### 1.3 Geometry & Effects
- **Corner Radius**: `16px` (Standard for cards, buttons, and input fields)
- **Shadows**: `shadow-sm` (Light elevation for cards to create depth without clutter)
- **Borders**: `1px solid #DCD9D9` (Used for subtle separation in list items and sidebars)

---

## 2. Core Image Assets

These assets are consistent across both platforms.

- **Primary Logo**: {{DATA:IMAGE:IMAGE_8}} (Wordmark with integrated location pin/movement motif)
- **Driver Lifestyle Reference**: {{DATA:IMAGE:IMAGE_9}} (High-end, trustworthy imagery for the dashboard)
- **Vehicle Asset Reference**: {{DATA:IMAGE:IMAGE_10}} (Modern white electric sedan for fleet selection)
- **Background Pattern**: {{DATA:IMAGE:IMAGE_7}} (Minimalist urban map illustration at low opacity)

---

## 3. Rider App: Desktop Architecture

The Rider experience is defined by a **Map-Centric Hybrid Layout**.

### 3.1 Layout Structure
- **Navigation (Top)**: Fixed height (~72px). Contains Logo (left), Navigation Links (Center: Ride, Drive, Activity, Help), and Profile/Notifications (Right).
- **Control Panel (Side)**: Floating or docked left-aligned panel. `Width: 400px`. `Background: Surface-Container-Lowest`. `Radius: 16px`.
- **Map View (Background)**: Full-bleed interactive map with route polyline overlays in Tech Blue (#276EF1).

### 3.2 Key Components: "Fleet Selection"
- **Vehicle Cards**: 
  - Standard State: White background, 1px border.
  - Active State: Tech Blue border (2px), Light Blue background (#E7F0FF).
  - Content: Vehicle Type, ETA, Price, and Capacity Icon.

---

## 4. Driver App: Partner Cockpit Architecture

The Driver experience is defined by a **Metrics-First Management Layout**.

### 4.1 Layout Structure
- **Navigation (Side)**: Docked left-aligned sidebar. `Width: 320px`. `Background: Surface-Container-Lowest`.
  - Includes user profile summary, navigation links (Home, Dashboard, Tracking, Profile), and the global "Go Online" CTA.
- **Dashboard (Main Content)**:
  - **Hero Header**: "Dashboard Overview" with quick performance summary.
  - **Metrics Row**: Three-column layout for Today's Earnings, Acceptance Rate, and Total Trips.
  - **Data Table**: "Recent Trips" log with Time, Destination, Distance, Earnings, and Rating.

### 4.2 Key Interactions
- **The "Go Online" Toggle**: A high-visibility button (`#008A5E`) used in both the sidebar and main header to manage active status.
- **Data Visualization**: Clean, high-contrast earnings numbers using `#276EF1` for primary emphasis.

---

## 5. Development Guidelines
- **Framework**: Tailwind CSS (Utility-first)
- **Responsive Logic**: Desktop-first (1440px target). Fluid scaling for containers.
- **Interactive States**: Buttons should have `active:scale-95 transition-transform` for tactile feedback.
- **Asset Usage**: All images must maintain original aspect ratios; use `object-cover` for hero photography.
