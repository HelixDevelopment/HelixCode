# HelixCode Video Course Player System

A professional, production-ready video course platform for the HelixCode website. This system provides a complete learning experience with video playback, progress tracking, interactive features, and course management.

## 📁 File Structure

```
courses/
├── index.html          # Main video player page
├── catalog.html        # Course catalog/listing page
├── course.html         # Individual course detail page
├── player.css          # Custom video player styles
├── player.js           # Video player controller
├── course-data.js      # Course metadata management
├── progress.js         # Progress tracking system
├── search.js           # Search and filtering functionality
└── README.md           # This file
```

## 🎯 Features

### Video Player (`index.html` + `player.js`)

**Core Playback:**
- HTML5 video with custom controls
- Play/pause, seek, volume control
- Playback speed control (0.5x - 2x)
- Quality selector (placeholder for future)
- Fullscreen support with API integration

**Advanced Features:**
- Interactive transcript synchronized with video
- Bookmark functionality with notes
- Auto-advance to next chapter
- Resume from last position
- Chapter navigation with sidebar
- Loading states and buffering indicators

**Keyboard Shortcuts:**
- `Space` - Play/Pause
- `→` - Forward 10 seconds
- `←` - Backward 10 seconds
- `↑` - Volume up
- `↓` - Volume down
- `F` - Toggle fullscreen
- `M` - Mute/unmute
- `C` - Toggle subtitles
- `N` - Toggle notes panel
- `B` - Add bookmark
- `?` - Show shortcuts help

**Progress Tracking:**
- Real-time progress saving to localStorage
- Course completion percentage
- Chapter completion status
- Recently watched tracking
- Study streak calculation

**Note Taking:**
- Collapsible notes panel
- Auto-save functionality
- Character count
- Per-course storage

### Course Catalog (`catalog.html` + `search.js`)

**Discovery:**
- Grid layout of all courses
- Course cards with thumbnails and metadata
- "Continue Learning" section for resumed courses
- Search functionality across titles and descriptions
- Filter by difficulty level (Beginner/Intermediate/Advanced)
- Sort by newest, popular, level, or duration

**Course Cards Display:**
- Course level badge
- Progress indicator for started courses
- Duration and chapter count
- Description and instructor
- Start/Continue button

### Course Detail Page (`course.html`)

**Information:**
- Comprehensive course overview
- Learning objectives list
- Complete chapter list with descriptions
- Prerequisites and requirements
- Instructor information
- Course includes (duration, resources, certificate)

**Preview:**
- Course preview thumbnail
- Progress tracking display
- Quick start functionality
- Chapter navigation

### Progress Tracking System (`progress.js`)

**Tracking Capabilities:**
- Chapter-level progress (time watched)
- Course completion percentage
- Last watched timestamp
- Completion status
- Study streaks

**Statistics:**
- Total watch time
- Courses started/completed
- Chapters completed
- Recent activity
- Achievement system

**Data Management:**
- localStorage persistence
- Export/import functionality
- Progress reset options
- Certificate eligibility checking

**Achievements:**
- Getting Started (first course)
- Dedicated Learner (5 chapters)
- Week Warrior (7-day streak)
- Course Master (first completion)
- Expert Learner (3 courses)

### Course Data Management (`course-data.js`)

**Course Structure:**
```javascript
{
  id: 'course-id',
  title: 'Course Title',
  description: 'Course description',
  instructor: 'Instructor Name',
  duration: '2h 30m',
  level: 'beginner|intermediate|advanced',
  chapters: [
    {
      id: 'chapter-id',
      number: 1,
      title: 'Chapter Title',
      description: 'Chapter description',
      duration: '15:00',
      videoUrl: 'video-url',
      transcript: [
        { time: 0, text: 'Transcript text' }
      ],
      resources: [
        { name: 'Resource Name', url: 'resource-url' }
      ]
    }
  ]
}
```

**Sample Courses Included:**
1. Introduction to HelixCode (Beginner, 2h 30m, 6 chapters)
2. Advanced HelixCode Features (Advanced, 3h 15m, 4 chapters)
3. Production Deployment (Intermediate, 2h 45m, 4 chapters)

## 🎨 Design Features

### Responsive Design
- Desktop: Full sidebar and notes panel
- Tablet: Collapsible sidebar
- Mobile: Bottom navigation, vertical layout

### Theme Support
- Light theme (default)
- Dark theme (auto-detection via prefers-color-scheme)
- Matches HelixCode brand colors

### Accessibility
- ARIA labels on all interactive elements
- Keyboard navigation support
- Focus visible indicators
- Screen reader friendly
- Semantic HTML structure

### Performance
- Lazy loading of video content
- Efficient DOM updates
- Debounced search input
- Optimized localStorage usage
- Smooth animations (CSS transitions)

## 🚀 Setup & Usage

### Integration with Existing Site

1. **Add to Navigation:**
   Update main site navigation to include courses link:
   ```html
   <li><a href="courses/catalog.html" class="nav-link">Courses</a></li>
   ```

2. **Video Content:**
   Currently uses sample videos. Replace with actual course videos:
   - Update `videoUrl` in `course-data.js`
   - Add your video files to appropriate hosting
   - Supported formats: MP4, WebM, OGG

3. **Customization:**
   - Colors: Edit CSS variables in `player.css`
   - Branding: Update logos and text
   - Course content: Modify `course-data.js`

### Adding New Courses

1. **Define Course Data:**
   ```javascript
   CourseData.courses['new-course-id'] = {
     id: 'new-course-id',
     title: 'New Course Title',
     description: 'Course description',
     instructor: 'Instructor Name',
     duration: '3h 00m',
     level: 'intermediate',
     chapters: [/* chapter objects */]
   };
   ```

2. **Add Chapter Content:**
   ```javascript
   {
     id: 'ch1-new',
     number: 1,
     title: 'Chapter Title',
     description: 'Chapter description',
     duration: '20:00',
     videoUrl: 'path/to/video.mp4',
     transcript: [
       { time: 0, text: 'Transcript text' },
       { time: 15, text: 'More text' }
     ],
     resources: [
       { name: 'Slides (PDF)', url: 'path/to/resource.pdf' }
     ]
   }
   ```

3. **Test:**
   - Load catalog page
   - Click on new course
   - Verify video playback
   - Check progress tracking

### Transcript Format

Transcripts are arrays of time-stamped text:
```javascript
transcript: [
  { time: 0, text: 'Welcome to this course.' },
  { time: 15, text: 'In this section, we will cover...' },
  { time: 45, text: 'Let\'s get started with...' }
]
```

Time is in seconds from video start.

## 💾 Data Storage

### localStorage Keys

- `helixcode_course_progress` - Chapter progress data
- `helixcode_bookmarks` - Video bookmarks
- `helixcode_notes_{courseId}` - Course notes

### Data Structure

**Progress:**
```javascript
{
  "chapter-id": {
    started: "2025-01-01T00:00:00.000Z",
    completed: true,
    completedAt: "2025-01-02T00:00:00.000Z",
    currentTime: 1234.56,
    lastWatched: "2025-01-02T00:00:00.000Z"
  }
}
```

**Bookmarks:**
```javascript
[
  {
    id: 1234567890,
    chapterId: "ch1-intro",
    time: 123.45,
    note: "Important concept",
    timestamp: "2025-01-01T00:00:00.000Z"
  }
]
```

## 🔧 API Reference

### VideoPlayer Class

```javascript
// Main player controller
const player = new VideoPlayer();

// Methods
player.loadChapter(chapterId);
player.togglePlay();
player.skip(seconds);
player.addBookmark();
player.toggleFullscreen();
player.setPlaybackSpeed(speed);
```

### ProgressTracker Class

```javascript
// Progress tracking
const tracker = new ProgressTrackingSystem();

// Methods
tracker.updateChapterProgress(chapterId, currentTime);
tracker.markChapterComplete(chapterId);
tracker.isChapterComplete(chapterId);
tracker.getCourseProgress(courseId);
tracker.getStats();
tracker.exportProgress();
```

### CourseData Class

```javascript
// Course data management
const data = new CourseDataManager();

// Methods
data.loadCourse(courseId);
data.getChapter(chapterId);
data.getAllCourses();
data.searchCourses(query);
data.filterCoursesByLevel(level);
```

### CourseCatalog Class

```javascript
// Catalog functionality
const catalog = new CourseCatalog();

// Methods
catalog.applyFilters();
catalog.applySorting();
catalog.renderCourses();
catalog.getRecommendations(courseId);
```

## 🎓 Certificate System

Users can earn certificates by:
1. Completing 3 or more courses (configurable)
2. Achieving 90%+ overall completion rate

Check eligibility:
```javascript
const certProgress = ProgressTracker.getCertificateProgress();
if (certProgress.eligible) {
  // Show certificate generation option
}
```

## 🔒 Security Considerations

- All data stored in client-side localStorage (no server)
- No authentication required (public courses)
- XSS protection via proper escaping
- Video URLs should use HTTPS
- Consider adding video access tokens for paid content

## 🚧 Future Enhancements

**Planned Features:**
- [ ] Video quality switching implementation
- [ ] Live streaming support
- [ ] Interactive quizzes between chapters
- [ ] Discussion/comments system
- [ ] Download for offline viewing
- [ ] Multi-language subtitle support
- [ ] Picture-in-picture mode
- [ ] Watch party feature
- [ ] Mobile apps (iOS/Android)
- [ ] Analytics dashboard

**Backend Integration:**
- [ ] User accounts and authentication
- [ ] Cloud progress sync
- [ ] Course purchases/subscriptions
- [ ] Instructor dashboard
- [ ] Content management system
- [ ] Video encoding pipeline
- [ ] CDN integration

## 📱 Browser Support

- Chrome 90+ ✅
- Firefox 88+ ✅
- Safari 14+ ✅
- Edge 90+ ✅
- Opera 76+ ✅
- Mobile Safari (iOS 14+) ✅
- Chrome Mobile (Android 8+) ✅

## 🐛 Troubleshooting

**Video won't play:**
- Check video URL in course-data.js
- Verify video format (MP4 recommended)
- Check browser console for errors
- Ensure CORS headers if using external CDN

**Progress not saving:**
- Check localStorage quota (usually 5-10MB)
- Verify browser allows localStorage
- Check browser console for errors

**Keyboard shortcuts not working:**
- Ensure focus is not in input/textarea
- Check browser console for conflicts
- Verify keyboard shortcuts modal shows correct keys

**Styles not loading:**
- Verify path to player.css is correct
- Check for CSS conflicts with parent site
- Clear browser cache

## 📄 License

This course player system is part of the HelixCode project and follows the same MIT License.

## 🤝 Contributing

To contribute to the course system:

1. Add new courses in `course-data.js`
2. Test across different devices
3. Update documentation
4. Submit improvements via pull request

## 📞 Support

For issues or questions:
- Check troubleshooting section above
- Review browser console for errors
- Open issue on GitHub
- Contact HelixCode team

---

**Built with ❤️ for the HelixCode community**

Last updated: January 2025
