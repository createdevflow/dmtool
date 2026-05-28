# DMTool - Unified Digital Marketing Intelligence Portal

## 🚀 Project Overview
DMTool is an enterprise-grade SaaS platform designed to centralize digital marketing intelligence. It integrates data from **Google Search Console** and **Meta (Facebook/Instagram)** to provide real-time insights, AI-driven content generation, and technical SEO audits in a single, premium dashboard.

### Tech Stack
- **Frontend**: Next.js 14 (App Router), TypeScript, TailwindCSS, Framer Motion (Animations), Lucide (Icons).
- **Backend**: Golang (Gin Framework), GORM (ORM), JWT (Authentication).
- **Database**: SQLite (Local development/demo).
- **External APIs**: Google Search Console API, Facebook Graph API.

---

## 🏛️ Architecture & System Design

### 1. Multi-Tenancy & User Isolation
The system is built with a strictly isolated multi-tenant architecture. 
- **Authentication**: JWT-based session management.
- **Data Filtering**: Every backend query is filtered by `UserID` (extracted from JWT) and `ProjectID` (passed via query parameters). This ensures that users only see their own data and can switch between multiple marketing properties (projects) seamlessly.

### 2. Frontend Structure
- **(auth)**: Registration and Login flows with persistent storage of tokens.
- **(dashboard)**: Protected layout containing:
  - **Sidebar**: Context-aware navigation that shifts based on the project's primary goal (SEO vs. Social).
  - **Topbar**: Real-time status, user profile, and **Logout** functionality.
  - **DashboardHeader**: Global project switcher allowing users to toggle between all connected domains.

### 3. Backend Structure
- **Handlers**: Modularized by domain (`auth`, `onboarding`, `dashboard`, `tools`, `system`).
- **Services**: Abstracted logic for third-party API interaction (`thirdparty.go`) and SEO analysis (`analyzer.go`).
- **Models**: Unified schema representing the entire marketing stack.

---

## 🛠️ Key Features

### 📊 Real-Time Dashboard
- **Growth Snapshot**: Displays live Clicks, Impressions (GSC), and Reach, Engagement, Followers (Meta).
- **Today's Focus**: AI-prioritized task of the day.
- **Trend Charts**: Visualizing traffic and engagement over time with percentage change indicators.

### 🌊 Intelligent Onboarding
- A multi-step flow that captures project details (URL, Goals) and social handles.
- **Auto-Seeding**: Automatically generates initial "realistic" mock data if real APIs aren't connected yet, allowing for an immediate "WOW" factor.

### 🔍 SEO Intelligence
- **Site Explorer**: Real-time technical audit including H1 checks, Meta description analysis, and response time metrics.
- **Keyword Research**: Seed-based keyword generation with Volume and Keyword Difficulty (KD) metrics.
- **Audit Reports**: Visualized list of technical issues with severity levels.

### 📱 Social Media Analyzer
- **Profile Analysis**: Real-time follower counts (Fan Count), reach, and engagement rates for Facebook and Instagram.
- **Channel Overview**: Side-by-side comparison of performance across all connected social channels.

### 🧠 Content AI
- AI-driven prompt system that generates platform-specific content (LinkedIn posts, Blog titles, Twitter hooks) based on user goals.

---

## 💾 Data Models (Schema)

### Core Models
- **User**: Name, Email, PasswordHash, Projects.
- **Project**: Name, URL, Goal, Social Handles (IG, Twitter, FB, LinkedIn), Health Score.
- **SocialMetric**: Platform, Reach, Engagement, Followers (fans), ProjectID.
- **Metric**: Time-series Traffic and Engagement data.
- **Insight**: AI-generated recommendations with priority types (CRITICAL, OPPORTUNITY).
- **Task**: To-do items for the Action Center.
- **SEOIssue**: Findings from site audits.
- **OAuthCredential**: Securely stores access tokens for Google and Meta integrations.

---

## 🔌 API Endpoints

### Authentication
- `POST /api/auth/register`
- `POST /api/auth/login`

### Dashboard & Metrics
- `GET /api/dashboard/snapshot?project_id={id}`: The main data feed.
- `GET /api/dashboard/metrics?project_id={id}`: Time-series graph data.
- `GET /api/dashboard/insights?project_id={id}`: Recommendation list.

### Tools & Analysis
- `POST /api/seo/audit/run`: Triggers a real-time site crawl.
- `GET /api/social/insights?project_id={id}`: Detailed social profile stats.
- `POST /api/content/generate`: AI content generation endpoint.

### System
- `GET /api/integrations`: List of connected apps.
- `PATCH /api/tasks/:id/toggle`: Mark to-dos as complete.

---

## 🔧 Setup & Configuration

### Prerequisites
- Go 1.21+
- Node.js 18+
- SQLite3

### Backend Setup
1. Navigate to `/backend`.
2. Run `go mod tidy` to install dependencies.
3. Start the server: `go run cmd/api/main.go`.
4. Server runs on `http://localhost:8080`.

### Frontend Setup
1. Navigate to `/frontend`.
2. Run `npm install`.
3. Start dev server: `npm run dev`.
4. Portal accessible on `http://localhost:3000`.

### OAuth Configuration
To enable real data, you must provide your own API keys for Google and Meta in the onboarding or integrations settings.
- **Google Search Console**: Requires OAuth 2.0 Client ID and Secret.
- **Meta (Facebook/Instagram)**: Requires Facebook App ID and App Secret.

---

## 💎 Design Philosophy
The project adheres to a **Premium, High-Contrast Aesthetic**:
- **Typography**: Uses modern sans-serif fonts for readability and an "Apple-like" feel.
- **Visual Feedback**: Every interaction (connecting an app, completing a task) provides immediate visual feedback via a custom Toaster system.
- **Glassmorphism**: Subtle blurs and translucent backgrounds on overlays and menus.
- **Micro-animations**: Powered by Framer Motion for smooth transitions between views.
