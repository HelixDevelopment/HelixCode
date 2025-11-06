// ===========================
// HelixCode Website JavaScript
// ===========================

(function() {
    'use strict';

    // ===========================
    // Utility Functions
    // ===========================

    function debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    function throttle(func, limit) {
        let inThrottle;
        return function(...args) {
            if (!inThrottle) {
                func.apply(this, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    }

    // ===========================
    // Navigation
    // ===========================

    function initNavigation() {
        const navbar = document.getElementById('navbar');
        const mobileMenuToggle = document.getElementById('mobileMenuToggle');
        const navLinks = document.getElementById('navLinks');
        const navLinkItems = document.querySelectorAll('.nav-link');

        // Mobile menu toggle
        if (mobileMenuToggle) {
            mobileMenuToggle.addEventListener('click', function() {
                navLinks.classList.toggle('active');
                this.classList.toggle('active');

                // Animate hamburger menu
                const spans = this.querySelectorAll('span');
                if (this.classList.contains('active')) {
                    spans[0].style.transform = 'rotate(45deg) translate(5px, 5px)';
                    spans[1].style.opacity = '0';
                    spans[2].style.transform = 'rotate(-45deg) translate(7px, -6px)';
                } else {
                    spans[0].style.transform = '';
                    spans[1].style.opacity = '';
                    spans[2].style.transform = '';
                }
            });
        }

        // Close mobile menu when clicking on a link
        navLinkItems.forEach(link => {
            link.addEventListener('click', function(e) {
                // Only close if it's an internal link
                if (this.getAttribute('href').startsWith('#')) {
                    navLinks.classList.remove('active');
                    if (mobileMenuToggle) {
                        mobileMenuToggle.classList.remove('active');
                        const spans = mobileMenuToggle.querySelectorAll('span');
                        spans[0].style.transform = '';
                        spans[1].style.opacity = '';
                        spans[2].style.transform = '';
                    }
                }
            });
        });

        // Navbar scroll effect
        const handleScroll = throttle(() => {
            if (window.scrollY > 50) {
                navbar.classList.add('scrolled');
            } else {
                navbar.classList.remove('scrolled');
            }
        }, 100);

        window.addEventListener('scroll', handleScroll);

        // Active section highlighting
        if (navLinkItems.length > 0) {
            const sections = document.querySelectorAll('section[id]');

            const highlightNav = throttle(() => {
                const scrollPosition = window.scrollY + 150;

                sections.forEach(section => {
                    const sectionTop = section.offsetTop;
                    const sectionHeight = section.offsetHeight;
                    const sectionId = section.getAttribute('id');

                    if (scrollPosition >= sectionTop && scrollPosition < sectionTop + sectionHeight) {
                        navLinkItems.forEach(link => {
                            link.classList.remove('active');
                            if (link.getAttribute('href') === `#${sectionId}`) {
                                link.classList.add('active');
                            }
                        });
                    }
                });
            }, 100);

            window.addEventListener('scroll', highlightNav);
        }
    }

    // ===========================
    // Smooth Scrolling
    // ===========================

    function initSmoothScrolling() {
        // Handle all anchor links
        document.querySelectorAll('a[href^="#"]').forEach(anchor => {
            anchor.addEventListener('click', function(e) {
                const href = this.getAttribute('href');

                // Skip if it's just "#"
                if (href === '#') {
                    e.preventDefault();
                    return;
                }

                const targetId = href.substring(1);
                const targetElement = document.getElementById(targetId);

                if (targetElement) {
                    e.preventDefault();

                    const offsetTop = targetElement.offsetTop - 80; // Account for fixed navbar

                    window.scrollTo({
                        top: offsetTop,
                        behavior: 'smooth'
                    });

                    // Update URL without jumping
                    if (history.pushState) {
                        history.pushState(null, null, href);
                    }
                }
            });
        });

        // Handle hash in URL on page load
        if (window.location.hash) {
            setTimeout(() => {
                const targetId = window.location.hash.substring(1);
                const targetElement = document.getElementById(targetId);

                if (targetElement) {
                    const offsetTop = targetElement.offsetTop - 80;
                    window.scrollTo({
                        top: offsetTop,
                        behavior: 'smooth'
                    });
                }
            }, 100);
        }
    }

    // ===========================
    // Manual Sidebar Navigation
    // ===========================

    function initManualSidebar() {
        const sidebar = document.getElementById('manualSidebar');
        if (!sidebar) return;

        const tocLinks = sidebar.querySelectorAll('.toc-link, .toc-sublink');
        const sections = document.querySelectorAll('.manual-section, .subsection');

        // Highlight active section in sidebar
        const highlightSidebar = throttle(() => {
            const scrollPosition = window.scrollY + 120;

            let activeFound = false;
            sections.forEach(section => {
                const sectionTop = section.offsetTop;
                const sectionHeight = section.offsetHeight;
                const sectionId = section.getAttribute('id');

                if (!activeFound && scrollPosition >= sectionTop && scrollPosition < sectionTop + sectionHeight) {
                    tocLinks.forEach(link => {
                        link.classList.remove('active');
                        const linkHref = link.getAttribute('href');
                        if (linkHref === `#${sectionId}`) {
                            link.classList.add('active');
                            activeFound = true;
                        }
                    });
                }
            });
        }, 100);

        window.addEventListener('scroll', highlightSidebar);
        highlightSidebar(); // Run once on load
    }

    // ===========================
    // Button Actions
    // ===========================

    function initButtonActions() {
        // Download button
        const downloadBtn = document.getElementById('downloadBtn');
        if (downloadBtn) {
            downloadBtn.addEventListener('click', function(e) {
                // Link is already set in HTML, but we can add analytics here
                console.log('Download button clicked');
            });
        }

        // Get Started button - already linked in HTML to manual/#2-installation--setup
        const getStartedBtn = document.getElementById('getStartedBtn');
        if (getStartedBtn) {
            getStartedBtn.addEventListener('click', function(e) {
                console.log('Get Started button clicked');
            });
        }

        // Start Learning button - scroll to courses section
        const startLearningBtn = document.getElementById('startLearningBtn');
        if (startLearningBtn) {
            startLearningBtn.addEventListener('click', function(e) {
                console.log('Start Learning button clicked');
            });
        }

        // Explore Features button - scroll to features section
        const exploreFeaturesBtn = document.getElementById('exploreFeaturesBtn');
        if (exploreFeaturesBtn) {
            exploreFeaturesBtn.addEventListener('click', function(e) {
                e.preventDefault();
                const featuresSection = document.getElementById('features');
                if (featuresSection) {
                    const offsetTop = featuresSection.offsetTop - 80;
                    window.scrollTo({
                        top: offsetTop,
                        behavior: 'smooth'
                    });
                }
            });
        }
    }

    // ===========================
    // Card Hover Effects
    // ===========================

    function initCardEffects() {
        const cards = document.querySelectorAll('.feature-card, .provider-card, .tool-card, .doc-card');

        cards.forEach(card => {
            card.addEventListener('mouseenter', function() {
                this.style.transform = 'translateY(-4px)';
            });

            card.addEventListener('mouseleave', function() {
                this.style.transform = '';
            });
        });
    }

    // ===========================
    // Fade In on Scroll
    // ===========================

    function initFadeInOnScroll() {
        const observerOptions = {
            threshold: 0.1,
            rootMargin: '0px 0px -50px 0px'
        };

        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    entry.target.classList.add('fade-in');
                    observer.unobserve(entry.target);
                }
            });
        }, observerOptions);

        const elements = document.querySelectorAll('.feature-card, .provider-card, .tool-card, .doc-card');
        elements.forEach(el => {
            el.style.opacity = '0';
            observer.observe(el);
        });
    }

    // ===========================
    // Back to Top Button
    // ===========================

    function initBackToTop() {
        // Create back to top button if it doesn't exist
        let backToTopBtn = document.querySelector('.back-to-top');

        if (!backToTopBtn && document.querySelector('.manual-content')) {
            backToTopBtn = document.createElement('button');
            backToTopBtn.className = 'back-to-top';
            backToTopBtn.innerHTML = '↑';
            backToTopBtn.setAttribute('aria-label', 'Back to top');
            document.body.appendChild(backToTopBtn);

            backToTopBtn.addEventListener('click', () => {
                window.scrollTo({
                    top: 0,
                    behavior: 'smooth'
                });
            });
        }

        if (backToTopBtn) {
            const toggleBackToTop = throttle(() => {
                if (window.scrollY > 500) {
                    backToTopBtn.classList.add('visible');
                } else {
                    backToTopBtn.classList.remove('visible');
                }
            }, 100);

            window.addEventListener('scroll', toggleBackToTop);
        }
    }

    // ===========================
    // Copy Code Blocks
    // ===========================

    function initCodeCopy() {
        const codeBlocks = document.querySelectorAll('pre code');

        codeBlocks.forEach(block => {
            const pre = block.parentElement;

            // Create copy button
            const copyBtn = document.createElement('button');
            copyBtn.className = 'copy-code-btn';
            copyBtn.textContent = 'Copy';
            copyBtn.setAttribute('aria-label', 'Copy code to clipboard');

            // Add button to pre element
            pre.style.position = 'relative';
            pre.appendChild(copyBtn);

            copyBtn.addEventListener('click', async () => {
                const code = block.textContent;

                try {
                    await navigator.clipboard.writeText(code);
                    copyBtn.textContent = 'Copied!';
                    copyBtn.classList.add('copied');

                    setTimeout(() => {
                        copyBtn.textContent = 'Copy';
                        copyBtn.classList.remove('copied');
                    }, 2000);
                } catch (err) {
                    console.error('Failed to copy code:', err);
                    copyBtn.textContent = 'Failed';

                    setTimeout(() => {
                        copyBtn.textContent = 'Copy';
                    }, 2000);
                }
            });
        });
    }

    // ===========================
    // Search Functionality (placeholder for future implementation)
    // ===========================

    function initSearch() {
        const searchInput = document.getElementById('searchInput');
        if (!searchInput) return;

        const handleSearch = debounce((query) => {
            if (query.length < 3) return;

            // Placeholder for search functionality
            console.log('Searching for:', query);

            // In the future, implement full-text search across documentation
            // This could use a client-side search library like Fuse.js or Lunr.js
        }, 300);

        searchInput.addEventListener('input', (e) => {
            handleSearch(e.target.value);
        });
    }

    // ===========================
    // Link Validation
    // ===========================

    function validateLinks() {
        const links = document.querySelectorAll('a[href="#"]');

        // Log any links that still point to "#" (should be none after fixes)
        if (links.length > 0) {
            console.warn('Found placeholder links that need fixing:', links.length);
            links.forEach(link => {
                console.warn('Placeholder link:', link.textContent, link);
            });
        }
    }

    // ===========================
    // Performance Monitoring
    // ===========================

    function logPerformance() {
        if (window.performance && window.performance.timing) {
            window.addEventListener('load', () => {
                setTimeout(() => {
                    const perfData = window.performance.timing;
                    const pageLoadTime = perfData.loadEventEnd - perfData.navigationStart;
                    const connectTime = perfData.responseEnd - perfData.requestStart;
                    const renderTime = perfData.domComplete - perfData.domLoading;

                    console.log('Performance Metrics:');
                    console.log(`Page Load Time: ${pageLoadTime}ms`);
                    console.log(`Connection Time: ${connectTime}ms`);
                    console.log(`Render Time: ${renderTime}ms`);
                }, 0);
            });
        }
    }

    // ===========================
    // Accessibility Enhancements
    // ===========================

    function initAccessibility() {
        // Add skip to content link
        const skipLink = document.createElement('a');
        skipLink.href = '#main';
        skipLink.className = 'skip-to-content';
        skipLink.textContent = 'Skip to content';
        document.body.insertBefore(skipLink, document.body.firstChild);

        // Handle keyboard navigation
        document.addEventListener('keydown', (e) => {
            // ESC to close mobile menu
            if (e.key === 'Escape') {
                const navLinks = document.getElementById('navLinks');
                const mobileMenuToggle = document.getElementById('mobileMenuToggle');

                if (navLinks && navLinks.classList.contains('active')) {
                    navLinks.classList.remove('active');
                    if (mobileMenuToggle) {
                        mobileMenuToggle.classList.remove('active');
                    }
                }
            }
        });
    }

    // ===========================
    // Theme Detection (for future dark mode)
    // ===========================

    function detectTheme() {
        if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
            console.log('User prefers dark mode');
            // Future: Apply dark mode styles
        }

        // Listen for theme changes
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', e => {
            const newColorScheme = e.matches ? 'dark' : 'light';
            console.log('Theme changed to:', newColorScheme);
            // Future: Switch theme dynamically
        });
    }

    // ===========================
    // Course Functionality
    // ===========================

    function initCourseFiltering() {
        const categoryButtons = document.querySelectorAll('.category-btn');
        const courseCards = document.querySelectorAll('.course-card');

        if (categoryButtons.length === 0 || courseCards.length === 0) return;

        categoryButtons.forEach(button => {
            button.addEventListener('click', function() {
                const category = this.getAttribute('data-category');

                // Update active button
                categoryButtons.forEach(btn => btn.classList.remove('active'));
                this.classList.add('active');

                // Filter courses
                courseCards.forEach(card => {
                    const cardLevel = card.getAttribute('data-level');

                    if (category === 'all') {
                        card.style.display = '';
                        setTimeout(() => {
                            card.style.opacity = '1';
                            card.style.transform = 'translateY(0)';
                        }, 10);
                    } else if (cardLevel === category) {
                        card.style.display = '';
                        setTimeout(() => {
                            card.style.opacity = '1';
                            card.style.transform = 'translateY(0)';
                        }, 10);
                    } else {
                        card.style.opacity = '0';
                        card.style.transform = 'translateY(20px)';
                        setTimeout(() => {
                            card.style.display = 'none';
                        }, 300);
                    }
                });
            });
        });
    }

    function initCourseProgress() {
        // Check localStorage for course progress
        const courseProgress = JSON.parse(localStorage.getItem('helixcode_course_progress') || '{}');

        // Display progress indicators if any courses are started
        if (Object.keys(courseProgress).length > 0) {
            console.log('Course progress loaded:', courseProgress);
        }
    }

    function trackCourseStart(courseId) {
        const progress = JSON.parse(localStorage.getItem('helixcode_course_progress') || '{}');

        if (!progress[courseId]) {
            progress[courseId] = {
                started: new Date().toISOString(),
                completed: false,
                lessonsCompleted: [],
                lastAccessed: new Date().toISOString()
            };

            localStorage.setItem('helixcode_course_progress', JSON.stringify(progress));
            console.log('Course started:', courseId);
        }
    }

    function trackLessonComplete(courseId, lessonId) {
        const progress = JSON.parse(localStorage.getItem('helixcode_course_progress') || '{}');

        if (!progress[courseId]) {
            progress[courseId] = {
                started: new Date().toISOString(),
                completed: false,
                lessonsCompleted: [],
                lastAccessed: new Date().toISOString()
            };
        }

        if (!progress[courseId].lessonsCompleted.includes(lessonId)) {
            progress[courseId].lessonsCompleted.push(lessonId);
            progress[courseId].lastAccessed = new Date().toISOString();
        }

        localStorage.setItem('helixcode_course_progress', JSON.stringify(progress));
        console.log('Lesson completed:', lessonId, 'in course:', courseId);
    }

    function trackCourseComplete(courseId) {
        const progress = JSON.parse(localStorage.getItem('helixcode_course_progress') || '{}');

        if (progress[courseId]) {
            progress[courseId].completed = true;
            progress[courseId].completedDate = new Date().toISOString();

            localStorage.setItem('helixcode_course_progress', JSON.stringify(progress));
            console.log('Course completed:', courseId);

            // Show certificate notification
            showCertificateNotification(courseId);
        }
    }

    function showCertificateNotification(courseId) {
        // Future: Show a notification that certificate is available
        console.log('Certificate available for course:', courseId);
    }

    function getCourseStats() {
        const progress = JSON.parse(localStorage.getItem('helixcode_course_progress') || '{}');

        const stats = {
            totalStarted: Object.keys(progress).length,
            totalCompleted: Object.values(progress).filter(p => p.completed).length,
            totalLessons: Object.values(progress).reduce((sum, p) => sum + p.lessonsCompleted.length, 0)
        };

        return stats;
    }

    function exportCourseProgress() {
        const progress = JSON.parse(localStorage.getItem('helixcode_course_progress') || '{}');
        const stats = getCourseStats();

        const exportData = {
            exportDate: new Date().toISOString(),
            stats: stats,
            progress: progress
        };

        const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'helixcode_course_progress.json';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        console.log('Course progress exported');
    }

    // Make functions globally available
    window.HelixCodeCourses = {
        trackCourseStart,
        trackLessonComplete,
        trackCourseComplete,
        getCourseStats,
        exportCourseProgress
    };

    // ===========================
    // Initialization
    // ===========================

    function init() {
        console.log('Initializing HelixCode website...');

        // Core functionality
        initNavigation();
        initSmoothScrolling();
        initButtonActions();

        // Manual page specific
        initManualSidebar();
        initBackToTop();
        initCodeCopy();

        // Course functionality
        initCourseFiltering();
        initCourseProgress();

        // Enhancements
        initCardEffects();
        initFadeInOnScroll();
        initAccessibility();

        // Development tools
        validateLinks();
        logPerformance();
        detectTheme();

        console.log('HelixCode website initialized successfully');
    }

    // Run initialization when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

})();

// ===========================
// Copy Code Button Styles (injected dynamically)
// ===========================

(function() {
    const style = document.createElement('style');
    style.textContent = `
        .copy-code-btn {
            position: absolute;
            top: 0.5rem;
            right: 0.5rem;
            background-color: rgba(255, 255, 255, 0.1);
            color: white;
            border: 1px solid rgba(255, 255, 255, 0.2);
            padding: 0.25rem 0.75rem;
            border-radius: 0.375rem;
            font-size: 0.75rem;
            cursor: pointer;
            transition: all 0.2s;
            font-family: inherit;
        }

        .copy-code-btn:hover {
            background-color: rgba(255, 255, 255, 0.2);
        }

        .copy-code-btn.copied {
            background-color: #10b981;
            border-color: #10b981;
        }

        .skip-to-content {
            position: absolute;
            top: -40px;
            left: 0;
            background: #6366f1;
            color: white;
            padding: 8px;
            text-decoration: none;
            z-index: 100;
        }

        .skip-to-content:focus {
            top: 0;
        }
    `;
    document.head.appendChild(style);
})();
