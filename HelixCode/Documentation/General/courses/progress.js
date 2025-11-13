/**
 * Progress Tracking System
 * Manages course and chapter progress using localStorage
 */

class ProgressTrackingSystem {
    constructor() {
        this.storageKey = 'helixcode_course_progress';
        this.progress = this.loadProgress();
    }

    loadProgress() {
        try {
            const saved = localStorage.getItem(this.storageKey);
            return saved ? JSON.parse(saved) : {};
        } catch (error) {
            console.error('Error loading progress:', error);
            return {};
        }
    }

    saveProgress() {
        try {
            localStorage.setItem(this.storageKey, JSON.stringify(this.progress));
        } catch (error) {
            console.error('Error saving progress:', error);
        }
    }

    getChapterProgress(chapterId) {
        return this.progress[chapterId]?.currentTime || 0;
    }

    updateChapterProgress(chapterId, currentTime) {
        if (!this.progress[chapterId]) {
            this.progress[chapterId] = {
                started: new Date().toISOString(),
                completed: false,
                currentTime: 0,
                lastWatched: new Date().toISOString()
            };
        }

        this.progress[chapterId].currentTime = currentTime;
        this.progress[chapterId].lastWatched = new Date().toISOString();
        this.saveProgress();
    }

    markChapterComplete(chapterId) {
        if (!this.progress[chapterId]) {
            this.progress[chapterId] = {
                started: new Date().toISOString()
            };
        }

        this.progress[chapterId].completed = true;
        this.progress[chapterId].completedAt = new Date().toISOString();
        this.saveProgress();
    }

    isChapterComplete(chapterId) {
        return this.progress[chapterId]?.completed || false;
    }

    getCourseProgress(courseId) {
        const course = CourseData.getCourseById(courseId);
        if (!course) return { completed: 0, total: 0, percentage: 0 };

        const total = course.chapters.length;
        const completed = course.chapters.filter(ch => this.isChapterComplete(ch.id)).length;
        const percentage = Math.round((completed / total) * 100);

        return { completed, total, percentage };
    }

    isCourseComplete(courseId) {
        const progress = this.getCourseProgress(courseId);
        return progress.percentage === 100;
    }

    getAllProgress() {
        return this.progress;
    }

    getRecentlyWatched(limit = 5) {
        const chapters = Object.entries(this.progress)
            .filter(([_, data]) => data.lastWatched)
            .map(([chapterId, data]) => ({
                chapterId,
                lastWatched: new Date(data.lastWatched),
                ...data
            }))
            .sort((a, b) => b.lastWatched - a.lastWatched)
            .slice(0, limit);

        return chapters;
    }

    getCompletedChapters() {
        return Object.entries(this.progress)
            .filter(([_, data]) => data.completed)
            .map(([chapterId, data]) => ({ chapterId, ...data }));
    }

    getTotalWatchTime() {
        // Calculate total watch time based on progress
        let totalSeconds = 0;

        Object.entries(this.progress).forEach(([chapterId, data]) => {
            if (data.completed || data.currentTime > 0) {
                totalSeconds += data.currentTime || 0;
            }
        });

        return this.formatDuration(totalSeconds);
    }

    getStudyStreak() {
        // Calculate consecutive days of studying
        const watchDates = Object.values(this.progress)
            .filter(data => data.lastWatched)
            .map(data => new Date(data.lastWatched).toDateString())
            .filter((date, index, self) => self.indexOf(date) === index)
            .sort((a, b) => new Date(b) - new Date(a));

        if (watchDates.length === 0) return 0;

        let streak = 1;
        const today = new Date().toDateString();

        if (watchDates[0] !== today) {
            const yesterday = new Date();
            yesterday.setDate(yesterday.getDate() - 1);
            if (watchDates[0] !== yesterday.toDateString()) {
                return 0; // Streak broken
            }
        }

        for (let i = 0; i < watchDates.length - 1; i++) {
            const current = new Date(watchDates[i]);
            const next = new Date(watchDates[i + 1]);
            const diffDays = Math.floor((current - next) / (1000 * 60 * 60 * 24));

            if (diffDays === 1) {
                streak++;
            } else {
                break;
            }
        }

        return streak;
    }

    resetChapterProgress(chapterId) {
        if (this.progress[chapterId]) {
            delete this.progress[chapterId];
            this.saveProgress();
        }
    }

    resetCourseProgress(courseId) {
        const course = CourseData.getCourseById(courseId);
        if (!course) return;

        course.chapters.forEach(chapter => {
            if (this.progress[chapter.id]) {
                delete this.progress[chapter.id];
            }
        });

        this.saveProgress();
    }

    resetAllProgress() {
        if (confirm('Are you sure you want to reset all progress? This cannot be undone.')) {
            this.progress = {};
            this.saveProgress();
            location.reload();
        }
    }

    exportProgress() {
        const exportData = {
            version: '1.0',
            exportDate: new Date().toISOString(),
            progress: this.progress,
            stats: this.getStats()
        };

        const blob = new Blob([JSON.stringify(exportData, null, 2)], {
            type: 'application/json'
        });

        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `helixcode-progress-${new Date().toISOString().split('T')[0]}.json`;
        a.click();
        URL.revokeObjectURL(url);
    }

    importProgress(file) {
        const reader = new FileReader();

        reader.onload = (e) => {
            try {
                const importData = JSON.parse(e.target.result);

                if (importData.version && importData.progress) {
                    if (confirm('Import progress? This will overwrite your current progress.')) {
                        this.progress = importData.progress;
                        this.saveProgress();
                        location.reload();
                    }
                } else {
                    alert('Invalid progress file format.');
                }
            } catch (error) {
                console.error('Error importing progress:', error);
                alert('Error importing progress file.');
            }
        };

        reader.readAsText(file);
    }

    getStats() {
        const allCourses = CourseData.getAllCourses();

        const stats = {
            totalCourses: allCourses.length,
            coursesStarted: 0,
            coursesCompleted: 0,
            totalChapters: 0,
            chaptersCompleted: 0,
            totalWatchTime: this.getTotalWatchTime(),
            studyStreak: this.getStudyStreak(),
            lastActivity: this.getLastActivity(),
            courseProgress: {}
        };

        allCourses.forEach(course => {
            const progress = this.getCourseProgress(course.id);
            stats.totalChapters += progress.total;
            stats.chaptersCompleted += progress.completed;

            if (progress.completed > 0) {
                stats.coursesStarted++;
            }

            if (progress.percentage === 100) {
                stats.coursesCompleted++;
            }

            stats.courseProgress[course.id] = progress;
        });

        return stats;
    }

    getLastActivity() {
        const recentChapters = this.getRecentlyWatched(1);
        if (recentChapters.length === 0) return null;

        return recentChapters[0].lastWatched;
    }

    getCertificateProgress() {
        const stats = this.getStats();
        const requiredCourses = 3; // Number of courses needed for certificate
        const requiredCompletion = 90; // Minimum completion percentage

        const eligible = stats.coursesCompleted >= requiredCourses;
        const overallCompletion = Math.round(
            (stats.chaptersCompleted / stats.totalChapters) * 100
        );

        return {
            eligible,
            coursesCompleted: stats.coursesCompleted,
            requiredCourses,
            overallCompletion,
            requiredCompletion,
            chaptersCompleted: stats.chaptersCompleted,
            totalChapters: stats.totalChapters
        };
    }

    formatDuration(seconds) {
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);

        if (hours > 0) {
            return `${hours}h ${minutes}m`;
        }
        return `${minutes}m`;
    }

    // Achievement system
    checkAchievements() {
        const stats = this.getStats();
        const achievements = [];

        if (stats.coursesStarted >= 1) {
            achievements.push({
                id: 'first-course',
                title: 'Getting Started',
                description: 'Started your first course',
                icon: '🎯'
            });
        }

        if (stats.chaptersCompleted >= 5) {
            achievements.push({
                id: 'dedicated-learner',
                title: 'Dedicated Learner',
                description: 'Completed 5 chapters',
                icon: '📚'
            });
        }

        if (stats.studyStreak >= 7) {
            achievements.push({
                id: 'week-streak',
                title: 'Week Warrior',
                description: '7-day study streak',
                icon: '🔥'
            });
        }

        if (stats.coursesCompleted >= 1) {
            achievements.push({
                id: 'course-complete',
                title: 'Course Master',
                description: 'Completed your first course',
                icon: '🎓'
            });
        }

        if (stats.coursesCompleted >= 3) {
            achievements.push({
                id: 'expert-learner',
                title: 'Expert Learner',
                description: 'Completed 3 courses',
                icon: '👑'
            });
        }

        return achievements;
    }

    getNextMilestone() {
        const stats = this.getStats();

        if (stats.chaptersCompleted === 0) {
            return {
                title: 'Complete your first chapter',
                progress: 0,
                target: 1
            };
        }

        if (stats.coursesCompleted === 0) {
            return {
                title: 'Complete your first course',
                progress: stats.courseProgress[Object.keys(stats.courseProgress)[0]]?.percentage || 0,
                target: 100
            };
        }

        if (stats.coursesCompleted < 3) {
            return {
                title: 'Complete 3 courses for certificate',
                progress: stats.coursesCompleted,
                target: 3
            };
        }

        return {
            title: 'All courses completed!',
            progress: stats.coursesCompleted,
            target: stats.totalCourses
        };
    }
}

// Initialize global instance
const ProgressTracker = new ProgressTrackingSystem();

// Export for use in other scripts
if (typeof window !== 'undefined') {
    window.ProgressTracker = ProgressTracker;
}
