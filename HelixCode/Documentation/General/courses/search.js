/**
 * Course Catalog and Search Functionality
 */

class CourseCatalog {
    constructor() {
        this.courses = [];
        this.filteredCourses = [];
        this.currentFilter = 'all';
        this.currentSort = 'newest';
        this.searchQuery = '';
    }

    init() {
        this.loadCourses();
        this.setupEventListeners();
        this.renderCourses();
        this.renderContinueLearning();
    }

    loadCourses() {
        this.courses = CourseData.getAllCourses();
        this.filteredCourses = [...this.courses];
    }

    setupEventListeners() {
        // Search input
        const searchInput = document.getElementById('courseSearch');
        if (searchInput) {
            let searchTimeout;
            searchInput.addEventListener('input', (e) => {
                clearTimeout(searchTimeout);
                searchTimeout = setTimeout(() => {
                    this.searchQuery = e.target.value.toLowerCase();
                    this.applyFilters();
                }, 300);
            });
        }

        // Filter buttons
        document.querySelectorAll('.filter-btn[data-filter]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                document.querySelectorAll('.filter-btn[data-filter]').forEach(b =>
                    b.classList.remove('active')
                );
                e.target.classList.add('active');
                this.currentFilter = e.target.dataset.filter;
                this.applyFilters();
            });
        });

        // Sort select
        const sortSelect = document.getElementById('sortSelect');
        if (sortSelect) {
            sortSelect.addEventListener('change', (e) => {
                this.currentSort = e.target.value;
                this.applySorting();
            });
        }
    }

    applyFilters() {
        let filtered = [...this.courses];

        // Apply level filter
        if (this.currentFilter !== 'all') {
            filtered = filtered.filter(course => course.level === this.currentFilter);
        }

        // Apply search query
        if (this.searchQuery) {
            filtered = filtered.filter(course =>
                course.title.toLowerCase().includes(this.searchQuery) ||
                course.description.toLowerCase().includes(this.searchQuery) ||
                course.chapters.some(ch =>
                    ch.title.toLowerCase().includes(this.searchQuery) ||
                    ch.description.toLowerCase().includes(this.searchQuery)
                )
            );
        }

        this.filteredCourses = filtered;
        this.applySorting();
    }

    applySorting() {
        const sorted = [...this.filteredCourses];

        switch (this.currentSort) {
            case 'newest':
                // Keep original order (newest first)
                break;

            case 'popular':
                // Sort by progress (courses with progress first)
                sorted.sort((a, b) => {
                    const progressA = ProgressTracker.getCourseProgress(a.id);
                    const progressB = ProgressTracker.getCourseProgress(b.id);
                    return progressB.completed - progressA.completed;
                });
                break;

            case 'level':
                // Sort by level (beginner, intermediate, advanced)
                const levelOrder = { beginner: 1, intermediate: 2, advanced: 3 };
                sorted.sort((a, b) => levelOrder[a.level] - levelOrder[b.level]);
                break;

            case 'duration':
                // Sort by duration
                sorted.sort((a, b) => {
                    const durationA = this.parseDuration(a.duration);
                    const durationB = this.parseDuration(b.duration);
                    return durationA - durationB;
                });
                break;
        }

        this.filteredCourses = sorted;
        this.renderCourses();
    }

    parseDuration(duration) {
        // Parse duration strings like "2h 30m" into minutes
        const hours = duration.match(/(\d+)h/);
        const minutes = duration.match(/(\d+)m/);

        return (hours ? parseInt(hours[1]) * 60 : 0) + (minutes ? parseInt(minutes[1]) : 0);
    }

    renderCourses() {
        const container = document.getElementById('coursesGrid');
        const emptyState = document.getElementById('emptyState');

        if (this.filteredCourses.length === 0) {
            container.innerHTML = '';
            emptyState.style.display = 'block';
            return;
        }

        emptyState.style.display = 'none';
        container.innerHTML = this.filteredCourses.map(course => this.createCourseCard(course)).join('');

        // Add click handlers
        container.querySelectorAll('.course-card').forEach(card => {
            card.addEventListener('click', () => {
                const courseId = card.dataset.courseId;
                this.openCourse(courseId);
            });
        });
    }

    createCourseCard(course) {
        const progress = ProgressTracker.getCourseProgress(course.id);
        const hasProgress = progress.completed > 0;

        const badgeClass = {
            beginner: 'badge-beginner',
            intermediate: 'badge-intermediate',
            advanced: 'badge-advanced'
        }[course.level];

        return `
            <div class="course-card" data-course-id="${course.id}">
                <div class="course-thumbnail">
                    <span>🎓</span>
                    ${hasProgress ? `
                        <div class="course-progress-overlay">
                            <div class="course-progress-bar" style="width: ${progress.percentage}%"></div>
                        </div>
                    ` : ''}
                </div>
                <div class="course-content">
                    <div class="course-meta">
                        <span class="course-badge ${badgeClass}">${course.level}</span>
                        ${hasProgress ? `<span class="course-badge" style="background-color: rgba(99, 102, 241, 0.1); color: var(--primary-color);">${progress.percentage}% Complete</span>` : ''}
                    </div>
                    <h3 class="course-title">${course.title}</h3>
                    <p class="course-description">${course.description}</p>
                    <div class="course-stats">
                        <div class="stat-item">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <circle cx="12" cy="12" r="10"></circle>
                                <polyline points="12 6 12 12 16 14"></polyline>
                            </svg>
                            <span>${course.duration}</span>
                        </div>
                        <div class="stat-item">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path>
                                <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path>
                            </svg>
                            <span>${course.chapters.length} chapters</span>
                        </div>
                    </div>
                    <div class="course-action">
                        <button class="btn ${hasProgress ? 'btn-primary' : 'btn-secondary'}" style="width: 100%;">
                            ${hasProgress ? 'Continue Learning' : 'Start Course'}
                        </button>
                    </div>
                </div>
            </div>
        `;
    }

    renderContinueLearning() {
        const recentChapters = ProgressTracker.getRecentlyWatched(3);

        if (recentChapters.length === 0) {
            return; // Hide section if no recent activity
        }

        const section = document.getElementById('continueLearning');
        const container = document.getElementById('continueGrid');

        // Get unique courses from recent chapters
        const recentCourses = new Map();

        recentChapters.forEach(chapterData => {
            // Find which course this chapter belongs to
            const course = this.courses.find(c =>
                c.chapters.some(ch => ch.id === chapterData.chapterId)
            );

            if (course && !recentCourses.has(course.id)) {
                recentCourses.set(course.id, {
                    course,
                    lastWatched: chapterData.lastWatched
                });
            }
        });

        if (recentCourses.size === 0) {
            return;
        }

        section.style.display = 'block';
        container.innerHTML = Array.from(recentCourses.values())
            .map(data => this.createCourseCard(data.course))
            .join('');

        // Add click handlers
        container.querySelectorAll('.course-card').forEach(card => {
            card.addEventListener('click', () => {
                const courseId = card.dataset.courseId;
                this.openCourse(courseId);
            });
        });
    }

    openCourse(courseId) {
        const course = this.courses.find(c => c.id === courseId);
        if (!course) return;

        // Find last watched chapter or first chapter
        const progress = ProgressTracker.getCourseProgress(courseId);
        let startChapter = course.chapters[0].id;

        // Find the last incomplete chapter
        for (const chapter of course.chapters) {
            if (!ProgressTracker.isChapterComplete(chapter.id)) {
                startChapter = chapter.id;
                break;
            }
        }

        // Navigate to player
        window.location.href = `index.html?course=${courseId}&chapter=${startChapter}`;
    }

    exportCourseList() {
        const exportData = {
            courses: this.courses.map(course => ({
                id: course.id,
                title: course.title,
                level: course.level,
                chapters: course.chapters.length,
                duration: course.duration
            }))
        };

        const blob = new Blob([JSON.stringify(exportData, null, 2)], {
            type: 'application/json'
        });

        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'helixcode-courses.json';
        a.click();
        URL.revokeObjectURL(url);
    }

    getRecommendations(courseId) {
        const currentCourse = this.courses.find(c => c.id === courseId);
        if (!currentCourse) return [];

        // Simple recommendation: courses of similar or next level
        const levelOrder = { beginner: 1, intermediate: 2, advanced: 3 };
        const currentLevel = levelOrder[currentCourse.level];

        return this.courses
            .filter(course => {
                const level = levelOrder[course.level];
                return course.id !== courseId && Math.abs(level - currentLevel) <= 1;
            })
            .slice(0, 3);
    }

    getCoursesByInstructor(instructor) {
        return this.courses.filter(course => course.instructor === instructor);
    }

    getTotalCourseDuration() {
        return this.courses.reduce((total, course) => {
            return total + this.parseDuration(course.duration);
        }, 0);
    }

    getCourseStats() {
        const stats = ProgressTracker.getStats();

        return {
            totalCourses: this.courses.length,
            totalChapters: this.courses.reduce((sum, c) => sum + c.chapters.length, 0),
            totalDuration: this.formatDuration(this.getTotalCourseDuration()),
            byLevel: {
                beginner: this.courses.filter(c => c.level === 'beginner').length,
                intermediate: this.courses.filter(c => c.level === 'intermediate').length,
                advanced: this.courses.filter(c => c.level === 'advanced').length
            },
            progress: {
                started: stats.coursesStarted,
                completed: stats.coursesCompleted,
                chaptersCompleted: stats.chaptersCompleted
            }
        };
    }

    formatDuration(minutes) {
        const hours = Math.floor(minutes / 60);
        const mins = minutes % 60;

        if (hours > 0) {
            return `${hours}h ${mins}m`;
        }
        return `${mins}m`;
    }
}

// Make available globally
if (typeof window !== 'undefined') {
    window.CourseCatalog = CourseCatalog;
}
