# DMTool Calendar & Instagram Publishing - Chat Context

## Overview
Ongoing work to improve the content calendar scheduler UI, fix backend startup issues, and align the frontend with actual Instagram Graph API capabilities.

## Project Structure
- **Frontend**: Next.js 16.2 + React + TypeScript (Tailwind CSS styling)
- **Backend**: Go + Gin API on port :8080
- **Database**: SQLite with GORM ORM
- **Authentication**: OAuth (Meta/Facebook, Google)
- **Third-party APIs**: Meta Graph API (Instagram/Facebook publishing), RapidAPI, DataForSEO

## Key Files Modified

### `frontend/app/(dashboard)/system/calendar/page.tsx`
Main calendar scheduler component with day selection, draft content management, and publishing.

**Recent Changes:**
1. Removed unsupported Instagram metadata fields:
   - Location autocomplete removed
   - Music discovery removed
   - Tags field removed
   - Thumbnail preview references removed

2. Simplified form submission:
   - Only sends `caption`, `media URL`, `content_type`, and `time` to backend
   - Platform is now derived from the selected project (not manually selectable)
   - Title is auto-generated from caption snippet (first 75 chars)

3. Platform Lock Implementation:
   - Added `getProjectPlatform()` function that inspects project handles:
     - Returns `instagram` if `IGHandle` or `ig_handle` exists
     - Returns `facebook` if `FBHandle`, `facebook_handle`, or `fb_handle` exists
     - Returns `twitter` if `twitter_handle` exists
     - Returns `linkedin` if `linkedin_handle` exists
     - Defaults to `instagram`
   - Platform dropdown removed from scheduler modal
   - All draft rows now use the project's platform automatically

4. Modal Layout Fix:
   - Reorganized grid columns to prevent field overlap:
     - Post Type: 3 cols
     - Content File: 5 cols
     - Time: 2 cols
     - Remove button: 2 cols (flex-end)
     - Caption: full 12 cols (moved to bottom)
   - Added `min-h-[130px]` to caption textarea for better visibility
   - Increased modal max-width to accommodate cleaner layout

5. Calendar Date Rendering Fix:
   - Updated `dayPosts` filter to handle both explicit dates and `dueDate` fallback:
     ```typescript
     const dayPosts = posts.filter((p) => {
       const hasExplicitDate = typeof p.day === 'number' && typeof p.month === 'number' && typeof p.year === 'number';
       if (hasExplicitDate) {
         return p.day === day && p.month === currentDate.getMonth() && p.year === currentDate.getFullYear();
       }
       const dueDate = p.dueDate ? new Date(p.dueDate) : null;
       return dueDate
         && dueDate.getFullYear() === currentDate.getFullYear()
         && dueDate.getMonth() === currentDate.getMonth()
         && dueDate.getDate() === day;
     });
     ```
   - This ensures scheduled content shows up correctly for current and future dates

## Backend Integration Points

### Meta/Instagram Publishing
**File**: `backend/internal/services/meta.go`

Publishing flow:
1. `PublishInstagramContent(igUserID, accessToken, task)` is called
2. Creates a `media` container on Instagram Graph API with caption and media URL
3. Waits for container to finish processing (polls status)
4. Calls `media_publish` on Graph API to publish the container

**Supported fields**:
- Caption (text, max 2200 chars)
- Media URL (image or video)
- Media type (post, story, reel)

**NOT supported via Graph API**:
- Location tagging
- Music/sound tagging
- Hashtag suggestions
- Custom tags

### Calendar Publishing Worker
**File**: `backend/internal/workers/calendar_publisher.go`

Background worker that:
1. Queries tasks with `publish_status = "scheduled"` and `due_date <= now`
2. For Instagram tasks:
   - Checks that project has `IGHandle` configured
   - Resolves the Instagram account from OAuth credentials
   - Calls `PublishInstagramContent`
   - Updates task status to "published" on success

### Project Model
**File**: `backend/internal/models/project.go`

```go
type Project struct {
  ID              uint
  IGHandle        string `json:"ig_handle"`      // Instagram username
  FBHandle        string `json:"facebook_handle"`
  TwitterHandle   string `json:"twitter_handle"`
  LinkedinHandle  string `json:"linkedin_handle"`
  // ... other fields
}
```

## API Endpoints

### Create Calendar Event
**POST** `/system/calendar`
- Request: FormData with `project_id`, `title`, `platform`, `content_type`, `caption`, `due_date`, `file`
- Response: Created task

### Update Calendar Event
**PUT** `/system/calendar/{id}`
- Request: FormData (same as create)
- Response: Updated task

### Get Calendar Events
**GET** `/system/calendar?project_id={id}`
- Response: Array of tasks for project

### Delete Calendar Event
**DELETE** `/system/calendar/{id}`
- Response: Success

## State Management

### Frontend Draft State
```typescript
type ScheduledDraft = {
  title: string;              // Auto-generated from caption
  platform: string;           // Locked to project platform
  contentType: string;        // "post" | "story" | "reel"
  time: string;              // HH:mm format
  caption: string;           // Required
  location: string;          // DEPRECATED - being removed
  music: string;            // DEPRECATED - being removed
  musicPreviewUrl: string;  // DEPRECATED - being removed
  tags: string;            // DEPRECATED - being removed
  assetName: string;       // Display filename
  file: File | null;       // Content media file
  thumbnail: File | null;  // UNUSED
  thumbnailName: string;   // UNUSED
  thumbnailUrl?: string;   // UNUSED
}
```

### API Form Submission
Current fields sent to backend:
- `project_id`
- `title` (auto-generated from caption)
- `platform` (project-derived)
- `content_type` (post/story/reel)
- `caption`
- `due_date` (ISO datetime string)
- `file` (multipart)

**NOT sent**:
- location
- music
- tags
- thumbnail

## Issues Fixed

### Issue 1: Backend Port Conflict (:8080)
**Symptom**: Backend failed to start with "address already in use" error
**Root Cause**: Another process was bound to port 8080
**Solution**: Kill process with `lsof -i :8080 | grep LISTEN | awk '{print $2}' | xargs kill -9`

### Issue 2: Unsupported Instagram Fields in UI
**Symptom**: Calendar form had location, music, and tags inputs, but backend never used them
**Root Cause**: Instagram Graph API (Meta's official API) doesn't support location tagging, music selection, or custom tags through `media_publish`
**Solution**: Removed all UI fields and form submissions for these fields

### Issue 3: Modal Fields Covering Content
**Symptom**: Modal form fields were overlapping due to tight grid layout
**Root Cause**: md:col-span allocations were too narrow for file input + label + preview text
**Solution**: Reorganized to 3-5-2-2 + full-width caption layout

### Issue 4: Tomorrow's Scheduled Posts Not Showing
**Symptom**: Content scheduled for tomorrow (or future dates) wasn't visible on calendar
**Root Cause**: Calendar day filter only checked explicit `day/month/year` fields, ignored `due_date` fallback
**Solution**: Updated filter to check `due_date` field when explicit date components are missing

### Issue 5: Platform Selector Confusion
**Symptom**: User could select a platform in the form different from the project's configured account
**Root Cause**: No validation linking platform selection to project capabilities
**Solution**: Removed platform dropdown; platform now auto-derived from project handles

## Testing Checklist

- [x] Calendar builds without TypeScript errors
- [x] Modal opens and closes correctly
- [x] Draft row add/remove works
- [x] File upload input appears correctly
- [x] Caption textarea is visible and properly sized
- [x] Platform is locked to project platform
- [x] Save button submits correct FormData
- [x] Calendar day count reflects both explicit dates and due_date
- [ ] Calendar publishes to Instagram successfully (requires running backend + OAuth)
- [ ] Scheduled posts appear on correct dates when fetched

## Configuration & Environment

### Backend Environment Variables (inferred from code)
```
META_APP_ID=<Meta App ID>
META_APP_SECRET=<Meta App Secret>
GOOGLE_CLIENT_ID=<Google OAuth Client ID>
GOOGLE_CLIENT_SECRET=<Google OAuth Client Secret>
DATABASE_URL=<SQLite path or connection string>
JWT_SECRET=<JWT signing secret>
```

### Frontend API Client
**File**: `frontend/lib/api-client.ts`
- Base URL: Configurable (typically http://localhost:8080 or https://api.dmtool.com)
- Calendar endpoints:
  - `getCalendar(projectId)`
  - `createCalendarEvent(formData)`
  - `updateCalendarEvent(id, formData)`
  - `deleteCalendarEvent(id)`

## Next Steps / Roadmap

1. **Backend Validation**:
   - Validate that `platform` in task matches project's configured handles
   - Return error if user tries to publish Instagram content but project has no `IGHandle`

2. **UI Improvements**:
   - Display project platform in modal header so user is aware which account they're scheduling for
   - Show warning if project has no handles configured for selected date
   - Add platform-specific content type restrictions (e.g., stories not supported on LinkedIn)

3. **Content Publishing**:
   - Test full calendar → Meta Graph API publishing flow
   - Implement retry logic for failed publishes
   - Add publish status indicators on calendar UI (scheduled, published, failed)

4. **Multi-platform Support**:
   - Extend backend to support Facebook, Twitter, LinkedIn publishing
   - Update form to allow per-platform content variants

5. **Draft Management**:
   - Persist unsaved drafts to localStorage
   - Add draft preview before publishing
   - Implement content approval workflow

## Debugging Commands

**Kill port 8080**:
```bash
lsof -i :8080 | grep LISTEN | awk '{print $2}' | xargs kill -9
```

**Start backend (dev mode)**:
```bash
cd backend && ./run-dev.sh
# or
go run ./cmd/api/main.go
```

**Start frontend (dev mode)**:
```bash
cd frontend && npm run dev
```

**Verify frontend build**:
```bash
cd frontend && npm run build
```

## Key Learnings

1. **API Limitations**: Instagram Graph API's `media_publish` endpoint is deliberately limited to prevent spam/abuse. Location tags, music, and hashtag suggestions are not supported.

2. **Date Handling**: Mixed use of explicit date fields (`day`, `month`, `year`) and ISO datetime (`due_date`) in the same model requires defensive filtering.

3. **Platform Binding**: Users should not be able to select a platform if the project doesn't have credentials for that platform. Better to auto-derive platform from project configuration.

4. **Form Layout**: Modal forms need careful grid layout planning to prevent overlaps. Full-width fields should be placed after narrow fields for clarity.

## Related Documentation

- [Instagram Graph API Docs](https://developers.facebook.com/docs/instagram-graph-api)
- [Meta Business Platform](https://www.facebook.com/business/help)
- Project README: `/Users/aryan/Desktop/dm tool/dmtool/README.md`
- System Architecture: `/Users/aryan/Desktop/dm tool/dmtool/SYSTEM_ARCHITECTURE.md`
- Project Documentation: `/Users/aryan/Desktop/dm tool/dmtool/PROJECT_DOCUMENTATION.md`

---

**Last Updated**: June 1, 2026
**Status**: Active Development
**Frontend Build**: ✅ Passing
**Backend**: Requires startup (port :8080)
