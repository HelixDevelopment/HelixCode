/**
 * HelixCode Video Course Player
 * Professional video player with advanced features
 */

class VideoPlayer {
    constructor() {
        this.video = document.getElementById('videoPlayer');
        this.container = document.getElementById('videoContainer');
        this.controls = document.getElementById('videoControls');
        this.currentChapterId = null;
        this.isFullscreen = false;
        this.isPlaying = false;
        this.bookmarks = [];

        this.init();
    }

    init() {
        this.setupVideoControls();
        this.setupKeyboardShortcuts();
        this.loadCourseFromURL();
        this.setupAutoSave();
    }

    setupVideoControls() {
        const playPauseBtn = document.getElementById('playPauseBtn');
        const volumeBtn = document.getElementById('volumeBtn');
        const volumeSlider = document.getElementById('volumeSlider');
        const fullscreenBtn = document.getElementById('fullscreenBtn');
        const speedBtn = document.getElementById('speedBtn');
        const qualityBtn = document.getElementById('qualityBtn');
        const subtitlesBtn = document.getElementById('subtitlesBtn');
        const bookmarkBtn = document.getElementById('bookmarkBtn');
        const videoProgress = document.getElementById('videoProgress');
        const playOverlay = document.getElementById('playOverlay');

        // Play/Pause
        playPauseBtn.addEventListener('click', () => this.togglePlay());
        playOverlay.addEventListener('click', () => this.togglePlay());
        this.video.addEventListener('click', () => this.togglePlay());

        // Volume
        volumeBtn.addEventListener('click', () => this.toggleMute());
        volumeSlider.addEventListener('input', (e) => {
            this.video.volume = e.target.value / 100;
            this.updateVolumeIcon();
        });

        // Fullscreen
        fullscreenBtn.addEventListener('click', () => this.toggleFullscreen());

        // Speed control
        speedBtn.addEventListener('click', () => {
            document.getElementById('speedMenu').classList.toggle('active');
        });

        document.querySelectorAll('.speed-option').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const speed = parseFloat(e.target.dataset.speed);
                this.setPlaybackSpeed(speed);
            });
        });

        // Quality control
        qualityBtn.addEventListener('click', () => {
            document.getElementById('qualityMenu').classList.toggle('active');
        });

        document.querySelectorAll('.quality-option').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const quality = e.target.dataset.quality;
                this.setQuality(quality);
            });
        });

        // Subtitles
        subtitlesBtn.addEventListener('click', () => this.toggleSubtitles());

        // Bookmarks
        bookmarkBtn.addEventListener('click', () => this.addBookmark());

        // Progress bar
        videoProgress.addEventListener('click', (e) => this.seek(e));
        videoProgress.addEventListener('mousemove', (e) => this.updateTimeTooltip(e));

        // Video events
        this.video.addEventListener('loadedmetadata', () => this.onVideoLoaded());
        this.video.addEventListener('timeupdate', () => this.onTimeUpdate());
        this.video.addEventListener('ended', () => this.onVideoEnded());
        this.video.addEventListener('play', () => this.onPlay());
        this.video.addEventListener('pause', () => this.onPause());
        this.video.addEventListener('waiting', () => this.showLoading());
        this.video.addEventListener('canplay', () => this.hideLoading());

        // Update buffered progress
        this.video.addEventListener('progress', () => this.updateBuffered());

        // Hide controls after inactivity
        let controlsTimeout;
        this.container.addEventListener('mousemove', () => {
            this.container.classList.add('controls-visible');
            clearTimeout(controlsTimeout);
            controlsTimeout = setTimeout(() => {
                if (this.isPlaying) {
                    this.container.classList.remove('controls-visible');
                }
            }, 3000);
        });
    }

    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Don't trigger shortcuts when typing in inputs
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
                return;
            }

            switch (e.key.toLowerCase()) {
                case ' ':
                    e.preventDefault();
                    this.togglePlay();
                    break;
                case 'arrowright':
                    e.preventDefault();
                    this.skip(10);
                    break;
                case 'arrowleft':
                    e.preventDefault();
                    this.skip(-10);
                    break;
                case 'arrowup':
                    e.preventDefault();
                    this.adjustVolume(0.1);
                    break;
                case 'arrowdown':
                    e.preventDefault();
                    this.adjustVolume(-0.1);
                    break;
                case 'f':
                    e.preventDefault();
                    this.toggleFullscreen();
                    break;
                case 'm':
                    e.preventDefault();
                    this.toggleMute();
                    break;
                case 'c':
                    e.preventDefault();
                    this.toggleSubtitles();
                    break;
                case 'n':
                    e.preventDefault();
                    toggleNotes();
                    break;
                case 'b':
                    e.preventDefault();
                    this.addBookmark();
                    break;
                case '?':
                    e.preventDefault();
                    showShortcutsModal();
                    break;
            }
        });

        // Fullscreen change
        document.addEventListener('fullscreenchange', () => {
            this.isFullscreen = !!document.fullscreenElement;
            this.updateFullscreenIcon();
        });
    }

    togglePlay() {
        if (this.video.paused) {
            this.video.play();
        } else {
            this.video.pause();
        }
    }

    onPlay() {
        this.isPlaying = true;
        this.container.classList.add('playing');
        document.querySelector('.play-icon').style.display = 'none';
        document.querySelector('.pause-icon').style.display = 'block';
    }

    onPause() {
        this.isPlaying = false;
        this.container.classList.remove('playing');
        document.querySelector('.play-icon').style.display = 'block';
        document.querySelector('.pause-icon').style.display = 'none';
    }

    toggleMute() {
        this.video.muted = !this.video.muted;
        this.updateVolumeIcon();
    }

    updateVolumeIcon() {
        const volumeIcon = document.querySelector('.volume-icon');
        const muteIcon = document.querySelector('.volume-mute-icon');

        if (this.video.muted || this.video.volume === 0) {
            volumeIcon.style.display = 'none';
            muteIcon.style.display = 'block';
        } else {
            volumeIcon.style.display = 'block';
            muteIcon.style.display = 'none';
        }
    }

    adjustVolume(delta) {
        const newVolume = Math.max(0, Math.min(1, this.video.volume + delta));
        this.video.volume = newVolume;
        document.getElementById('volumeSlider').value = newVolume * 100;
        this.updateVolumeIcon();
    }

    toggleFullscreen() {
        if (!this.isFullscreen) {
            if (this.container.requestFullscreen) {
                this.container.requestFullscreen();
            } else if (this.container.webkitRequestFullscreen) {
                this.container.webkitRequestFullscreen();
            } else if (this.container.msRequestFullscreen) {
                this.container.msRequestFullscreen();
            }
        } else {
            if (document.exitFullscreen) {
                document.exitFullscreen();
            } else if (document.webkitExitFullscreen) {
                document.webkitExitFullscreen();
            } else if (document.msExitFullscreen) {
                document.msExitFullscreen();
            }
        }
    }

    updateFullscreenIcon() {
        const fsIcon = document.querySelector('.fullscreen-icon');
        const fsExitIcon = document.querySelector('.fullscreen-exit-icon');

        if (this.isFullscreen) {
            fsIcon.style.display = 'none';
            fsExitIcon.style.display = 'block';
        } else {
            fsIcon.style.display = 'block';
            fsExitIcon.style.display = 'none';
        }
    }

    setPlaybackSpeed(speed) {
        this.video.playbackRate = speed;
        document.getElementById('speedText').textContent = `${speed}x`;
        document.querySelectorAll('.speed-option').forEach(btn => {
            btn.classList.toggle('active', parseFloat(btn.dataset.speed) === speed);
        });
        document.getElementById('speedMenu').classList.remove('active');
    }

    setQuality(quality) {
        // Placeholder for quality switching
        console.log('Quality changed to:', quality);
        document.querySelectorAll('.quality-option').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.quality === quality);
        });
        document.getElementById('qualityMenu').classList.remove('active');

        // In production, this would switch video sources
        // const currentTime = this.video.currentTime;
        // this.video.src = getVideoSourceByQuality(quality);
        // this.video.currentTime = currentTime;
        // this.video.play();
    }

    toggleSubtitles() {
        const track = this.video.textTracks[0];
        if (track) {
            track.mode = track.mode === 'showing' ? 'hidden' : 'showing';
        }
    }

    skip(seconds) {
        this.video.currentTime += seconds;
    }

    seek(e) {
        const rect = e.currentTarget.getBoundingClientRect();
        const pos = (e.clientX - rect.left) / rect.width;
        this.video.currentTime = pos * this.video.duration;
    }

    updateTimeTooltip(e) {
        const rect = e.currentTarget.getBoundingClientRect();
        const pos = (e.clientX - rect.left) / rect.width;
        const time = pos * this.video.duration;

        const tooltip = document.getElementById('timeTooltip');
        tooltip.textContent = this.formatTime(time);
        tooltip.style.left = `${e.clientX - rect.left}px`;
    }

    onVideoLoaded() {
        document.getElementById('duration').textContent = this.formatTime(this.video.duration);
        this.hideLoading();
    }

    onTimeUpdate() {
        const progress = (this.video.currentTime / this.video.duration) * 100;
        document.getElementById('videoProgressFilled').style.width = `${progress}%`;
        document.getElementById('currentTime').textContent = this.formatTime(this.video.currentTime);

        // Update progress in storage
        if (this.currentChapterId) {
            ProgressTracker.updateChapterProgress(this.currentChapterId, this.video.currentTime);
        }

        // Sync transcript
        this.syncTranscript();
    }

    updateBuffered() {
        if (this.video.buffered.length > 0) {
            const buffered = (this.video.buffered.end(this.video.buffered.length - 1) / this.video.duration) * 100;
            document.getElementById('videoProgressBuffered').style.width = `${buffered}%`;
        }
    }

    syncTranscript() {
        const currentTime = this.video.currentTime;
        const transcriptItems = document.querySelectorAll('.transcript-item');

        transcriptItems.forEach(item => {
            const time = parseFloat(item.dataset.time);
            const nextItem = item.nextElementSibling;
            const nextTime = nextItem ? parseFloat(nextItem.dataset.time) : Infinity;

            if (currentTime >= time && currentTime < nextTime) {
                item.classList.add('active');
            } else {
                item.classList.remove('active');
            }
        });
    }

    onVideoEnded() {
        // Mark chapter as complete
        if (this.currentChapterId) {
            ProgressTracker.markChapterComplete(this.currentChapterId);
            updateProgress();
        }

        // Auto-advance to next chapter
        const nextBtn = document.getElementById('nextChapterBtn');
        if (nextBtn && !nextBtn.disabled) {
            setTimeout(() => {
                if (confirm('Chapter complete! Continue to next chapter?')) {
                    nextBtn.click();
                }
            }, 1000);
        }
    }

    addBookmark() {
        const time = this.video.currentTime;
        const note = prompt('Add a note for this bookmark (optional):');

        if (note !== null) {
            const bookmark = {
                id: Date.now(),
                chapterId: this.currentChapterId,
                time: time,
                note: note || `Bookmark at ${this.formatTime(time)}`,
                timestamp: new Date().toISOString()
            };

            this.bookmarks.push(bookmark);
            this.saveBookmarks();
            this.renderBookmarks();

            // Show feedback
            this.showNotification('Bookmark added!');
        }
    }

    saveBookmarks() {
        localStorage.setItem('helixcode_bookmarks', JSON.stringify(this.bookmarks));
    }

    loadBookmarks() {
        const saved = localStorage.getItem('helixcode_bookmarks');
        this.bookmarks = saved ? JSON.parse(saved) : [];
        this.renderBookmarks();
    }

    renderBookmarks() {
        const container = document.getElementById('bookmarksList');
        const chapterBookmarks = this.bookmarks.filter(b => b.chapterId === this.currentChapterId);

        if (chapterBookmarks.length === 0) {
            container.innerHTML = '<p class="no-bookmarks">No bookmarks yet. Click the bookmark button while watching to add one.</p>';
            return;
        }

        container.innerHTML = chapterBookmarks.map(bookmark => `
            <div class="bookmark-item" data-bookmark-id="${bookmark.id}" onclick="player.seekToBookmark(${bookmark.time})">
                <div class="bookmark-info">
                    <div class="bookmark-time">${this.formatTime(bookmark.time)}</div>
                    <div class="bookmark-note">${bookmark.note}</div>
                </div>
                <div class="bookmark-actions">
                    <button class="btn-icon" onclick="event.stopPropagation(); player.deleteBookmark(${bookmark.id})" aria-label="Delete bookmark">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M18 6L6 18M6 6l12 12"></path>
                        </svg>
                    </button>
                </div>
            </div>
        `).join('');
    }

    seekToBookmark(time) {
        this.video.currentTime = time;
        if (this.video.paused) {
            this.video.play();
        }
    }

    deleteBookmark(id) {
        if (confirm('Delete this bookmark?')) {
            this.bookmarks = this.bookmarks.filter(b => b.id !== id);
            this.saveBookmarks();
            this.renderBookmarks();
        }
    }

    showLoading() {
        this.container.classList.add('loading');
    }

    hideLoading() {
        this.container.classList.remove('loading');
    }

    showNotification(message) {
        // Simple notification - could be enhanced with a toast system
        const notification = document.createElement('div');
        notification.textContent = message;
        notification.style.cssText = `
            position: fixed;
            bottom: 20px;
            right: 20px;
            background: var(--primary-color);
            color: white;
            padding: 12px 24px;
            border-radius: 8px;
            z-index: 10000;
            animation: slideIn 0.3s ease-out;
        `;
        document.body.appendChild(notification);

        setTimeout(() => {
            notification.remove();
        }, 3000);
    }

    loadChapter(chapterId) {
        this.currentChapterId = chapterId;
        const chapter = CourseData.getChapter(chapterId);

        if (!chapter) {
            console.error('Chapter not found:', chapterId);
            return;
        }

        // Update video source
        this.video.src = chapter.videoUrl;

        // Update chapter info
        document.getElementById('chapterTitle').textContent = chapter.title;
        document.getElementById('chapterNumber').textContent = `Chapter ${chapter.number}`;
        document.getElementById('chapterDuration').textContent = chapter.duration;
        document.getElementById('chapterDescription').textContent = chapter.description;

        // Load transcript
        this.loadTranscript(chapter.transcript);

        // Load resources
        this.loadResources(chapter.resources);

        // Load saved progress
        const savedTime = ProgressTracker.getChapterProgress(chapterId);
        if (savedTime > 0) {
            this.video.currentTime = savedTime;
        }

        // Update navigation buttons
        this.updateNavigationButtons();

        // Load bookmarks
        this.loadBookmarks();

        // Update URL
        this.updateURL(chapterId);
    }

    loadTranscript(transcript) {
        const container = document.getElementById('transcriptContent');

        if (!transcript || transcript.length === 0) {
            container.innerHTML = '<p>No transcript available for this chapter.</p>';
            return;
        }

        container.innerHTML = transcript.map(item => `
            <p class="transcript-item" data-time="${item.time}" onclick="player.video.currentTime = ${item.time}">
                <span class="transcript-time">${this.formatTime(item.time)}</span>
                ${item.text}
            </p>
        `).join('');
    }

    loadResources(resources) {
        const container = document.getElementById('resourceList');

        if (!resources || resources.length === 0) {
            container.innerHTML = '<li class="resource-item">No resources available for this chapter.</li>';
            return;
        }

        container.innerHTML = resources.map(resource => `
            <li class="resource-item">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path>
                    <polyline points="13 2 13 9 20 9"></polyline>
                </svg>
                <a href="${resource.url}" download>${resource.name}</a>
            </li>
        `).join('');
    }

    updateNavigationButtons() {
        const course = CourseData.getCurrentCourse();
        if (!course) return;

        const currentIndex = course.chapters.findIndex(ch => ch.id === this.currentChapterId);

        const prevBtn = document.getElementById('prevChapterBtn');
        const nextBtn = document.getElementById('nextChapterBtn');

        prevBtn.disabled = currentIndex === 0;
        nextBtn.disabled = currentIndex === course.chapters.length - 1;

        prevBtn.onclick = () => {
            if (currentIndex > 0) {
                this.loadChapter(course.chapters[currentIndex - 1].id);
            }
        };

        nextBtn.onclick = () => {
            if (currentIndex < course.chapters.length - 1) {
                this.loadChapter(course.chapters[currentIndex + 1].id);
            }
        };
    }

    loadCourseFromURL() {
        const params = new URLSearchParams(window.location.search);
        const courseId = params.get('course') || 'intro-to-helixcode';
        const chapterId = params.get('chapter') || null;

        CourseData.loadCourse(courseId).then(() => {
            const course = CourseData.getCurrentCourse();
            if (course) {
                document.getElementById('navCourseTitle').textContent = course.title;

                // Load chapters in sidebar
                loadChapterList(course);

                // Load first chapter or specified chapter
                const startChapter = chapterId || course.chapters[0].id;
                this.loadChapter(startChapter);
            }
        });
    }

    updateURL(chapterId) {
        const params = new URLSearchParams(window.location.search);
        params.set('chapter', chapterId);
        const newURL = `${window.location.pathname}?${params.toString()}`;
        window.history.replaceState({}, '', newURL);
    }

    setupAutoSave() {
        // Auto-save notes
        const notesTextarea = document.getElementById('notesTextarea');
        let notesTimeout;

        notesTextarea.addEventListener('input', () => {
            clearTimeout(notesTimeout);
            document.getElementById('notesStatus').textContent = 'Saving...';

            notesTimeout = setTimeout(() => {
                this.saveNotes();
                document.getElementById('notesStatus').textContent = 'Notes saved';
            }, 1000);

            // Update character count
            document.getElementById('notesCount').textContent = `${notesTextarea.value.length} characters`;
        });

        // Load saved notes
        this.loadNotes();
    }

    saveNotes() {
        const notes = document.getElementById('notesTextarea').value;
        const courseId = new URLSearchParams(window.location.search).get('course');
        localStorage.setItem(`helixcode_notes_${courseId}`, notes);
    }

    loadNotes() {
        const courseId = new URLSearchParams(window.location.search).get('course');
        const notes = localStorage.getItem(`helixcode_notes_${courseId}`) || '';
        document.getElementById('notesTextarea').value = notes;
        document.getElementById('notesCount').textContent = `${notes.length} characters`;
    }

    formatTime(seconds) {
        if (isNaN(seconds)) return '0:00';

        const hrs = Math.floor(seconds / 3600);
        const mins = Math.floor((seconds % 3600) / 60);
        const secs = Math.floor(seconds % 60);

        if (hrs > 0) {
            return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
        }
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }
}

// UI Helper Functions
function loadChapterList(course) {
    const container = document.getElementById('chapterList');

    container.innerHTML = course.chapters.map(chapter => {
        const isCompleted = ProgressTracker.isChapterComplete(chapter.id);
        const isActive = player.currentChapterId === chapter.id;

        return `
            <div class="chapter-item ${isCompleted ? 'completed' : ''} ${isActive ? 'active' : ''}"
                 onclick="player.loadChapter('${chapter.id}')"
                 data-chapter-id="${chapter.id}">
                <div class="chapter-item-header">
                    <div class="chapter-check">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                            <polyline points="20 6 9 17 4 12"></polyline>
                        </svg>
                    </div>
                    <h4 class="chapter-item-title">${chapter.number}. ${chapter.title}</h4>
                    <span class="chapter-item-duration">${chapter.duration}</span>
                </div>
                <p class="chapter-item-description">${chapter.description}</p>
            </div>
        `;
    }).join('');
}

function updateProgress() {
    const course = CourseData.getCurrentCourse();
    if (!course) return;

    const completed = course.chapters.filter(ch => ProgressTracker.isChapterComplete(ch.id)).length;
    const total = course.chapters.length;
    const percentage = Math.round((completed / total) * 100);

    document.getElementById('courseProgress').textContent = `${percentage}%`;
    document.getElementById('courseProgressBar').style.width = `${percentage}%`;
    document.getElementById('completedChapters').textContent = completed;
    document.getElementById('totalChapters').textContent = total;

    // Update chapter list
    loadChapterList(course);
}

function toggleSidebar() {
    document.querySelector('.player-container').classList.toggle('sidebar-collapsed');
    document.querySelector('.player-sidebar').classList.toggle('active');
}

function toggleNotes() {
    document.querySelector('.player-container').classList.toggle('notes-open');
}

function showShortcutsModal() {
    document.getElementById('shortcutsModal').classList.add('active');
}

function hideShortcutsModal() {
    document.getElementById('shortcutsModal').classList.remove('active');
}

// Tab Switching
function setupTabs() {
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.dataset.tab;

            // Update buttons
            document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');

            // Update panels
            document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
            document.getElementById(`${tab}Panel`).classList.add('active');
        });
    });
}

// Mark chapter as complete
document.getElementById('markCompleteBtn')?.addEventListener('click', function() {
    if (player.currentChapterId) {
        ProgressTracker.markChapterComplete(player.currentChapterId);
        updateProgress();
        this.textContent = 'Completed!';
        this.disabled = true;

        setTimeout(() => {
            this.textContent = 'Mark as Complete';
            this.disabled = false;
        }, 2000);
    }
});

// Exit player
document.getElementById('exitPlayerBtn')?.addEventListener('click', () => {
    if (confirm('Exit course player? Your progress will be saved.')) {
        window.location.href = 'catalog.html';
    }
});

// Sidebar toggle
document.getElementById('sidebarToggle')?.addEventListener('click', toggleSidebar);

// Notes toggle
document.getElementById('toggleNotesBtn')?.addEventListener('click', toggleNotes);
document.getElementById('closeNotesBtn')?.addEventListener('click', toggleNotes);

// Shortcuts modal
document.getElementById('closeShortcutsBtn')?.addEventListener('click', hideShortcutsModal);
document.getElementById('shortcutsModal')?.addEventListener('click', (e) => {
    if (e.target.id === 'shortcutsModal') {
        hideShortcutsModal();
    }
});

// Transcript search
document.getElementById('transcriptSearch')?.addEventListener('input', (e) => {
    const query = e.target.value.toLowerCase();
    document.querySelectorAll('.transcript-item').forEach(item => {
        const text = item.textContent.toLowerCase();
        item.style.display = text.includes(query) ? 'block' : 'none';
    });
});

// Initialize
let player;
document.addEventListener('DOMContentLoaded', () => {
    player = new VideoPlayer();
    setupTabs();
    updateProgress();
});
