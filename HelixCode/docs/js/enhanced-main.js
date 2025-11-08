// Enhanced HelixCode Website JavaScript

// Wait for DOM to be fully loaded
document.addEventListener('DOMContentLoaded', function() {
    initializeWebsite();
});

// Main website initialization
function initializeWebsite() {
    console.log('🚀 HelixCode Enhanced Website Initialized');
    
    // Initialize all components
    initializeNavigation();
    initializeAnimations();
    initializeSmoothScrolling();
    initializeInteractions();
    initializePerformanceOptimizations();
    initializeAnalytics();
    
    // Add loading complete animation
    setTimeout(() => {
        document.body.classList.add('loaded');
        showWelcomeAnimation();
    }, 500);
}

// Navigation functionality
function initializeNavigation() {
    const navbar = document.getElementById('navbar');
    const mobileMenuToggle = document.getElementById('mobileMenuToggle');
    const navLinks = document.getElementById('navLinks');
    const navLinksItems = document.querySelectorAll('.nav-link');
    
    // Mobile menu toggle
    if (mobileMenuToggle && navLinks) {
        mobileMenuToggle.addEventListener('click', function() {
            navLinks.classList.toggle('active');
            mobileMenuToggle.classList.toggle('active');
        });
    }
    
    // Close mobile menu when clicking outside
    document.addEventListener('click', function(event) {
        if (!navbar.contains(event.target) && navLinks.classList.contains('active')) {
            navLinks.classList.remove('active');
            mobileMenuToggle.classList.remove('active');
        }
    });
    
    // Active navigation highlighting
    function updateActiveNavLink() {
        const sections = document.querySelectorAll('section[id]');
        const scrollY = window.pageYOffset;
        
        sections.forEach(section => {
            const sectionHeight = section.offsetHeight;
            const sectionTop = section.offsetTop - 100;
            const sectionId = section.getAttribute('id');
            
            if (scrollY > sectionTop && scrollY <= sectionTop + sectionHeight) {
                navLinksItems.forEach(item => {
                    item.classList.remove('active');
                    if (item.getAttribute('href') === `#${sectionId}`) {
                        item.classList.add('active');
                    }
                });
            }
        });
    }
    
    // Update active nav link on scroll
    window.addEventListener('scroll', updateActiveNavLink);
    updateActiveNavLink(); // Initial call
    
    // Navbar background on scroll
    function updateNavbarBackground() {
        if (window.scrollY > 50) {
            navbar.style.background = 'rgba(255, 255, 255, 0.98)';
            navbar.style.boxShadow = '0 4px 6px -1px rgba(0, 0, 0, 0.1)';
        } else {
            navbar.style.background = 'rgba(255, 255, 255, 0.95)';
            navbar.style.boxShadow = 'none';
        }
    }
    
    window.addEventListener('scroll', updateNavbarBackground);
    updateNavbarBackground();
}

// Animation initialization
function initializeAnimations() {
    // Intersection Observer for scroll animations
    const animationElements = document.querySelectorAll('.why-card, .feature-card, .section-header');
    
    const animationObserver = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('animate-in');
                
                // Add stagger animation for cards
                if (entry.target.classList.contains('why-card') || entry.target.classList.contains('feature-card')) {
                    const cards = entry.target.parentElement.querySelectorAll('.why-card, .feature-card');
                    cards.forEach((card, index) => {
                        setTimeout(() => {
                            card.classList.add('animate-in');
                        }, index * 100);
                    });
                }
            }
        });
    }, {
        threshold: 0.1,
        rootMargin: '0px 0px -50px 0px'
    });
    
    animationElements.forEach(element => {
        animationObserver.observe(element);
    });
}

// Smooth scrolling
function initializeSmoothScrolling() {
    const links = document.querySelectorAll('a[href^="#"]');
    
    links.forEach(link => {
        link.addEventListener('click', function(e) {
            const href = this.getAttribute('href');
            
            if (href !== '#' && href.startsWith('#')) {
                e.preventDefault();
                
                const targetElement = document.querySelector(href);
                if (targetElement) {
                    const targetPosition = targetElement.offsetTop - 80; // Account for fixed navbar
                    
                    window.scrollTo({
                        top: targetPosition,
                        behavior: 'smooth'
                    });
                    
                    // Update active nav link
                    document.querySelectorAll('.nav-link').forEach(item => {
                        item.classList.remove('active');
                    });
                    document.querySelector(`.nav-link[href="${href}"]`).classList.add('active');
                }
            }
        });
    });
}

// Interactive elements
function initializeInteractions() {
    // Enhanced button interactions
    const buttons = document.querySelectorAll('.btn');
    buttons.forEach(button => {
        button.addEventListener('mouseenter', function(e) {
            createRippleEffect(e, this);
        });
        
        button.addEventListener('click', function(e) {
            handleButtonClick(e, this);
        });
    });
    
    // Feature card interactions
    const featureCards = document.querySelectorAll('.feature-card');
    featureCards.forEach(card => {
        card.addEventListener('mouseenter', function() {
            this.classList.add('hovered');
        });
        
        card.addEventListener('mouseleave', function() {
            this.classList.remove('hovered');
        });
    });
    
    // Provider and tool tag interactions
    const tags = document.querySelectorAll('.provider-tag, .use-case-tag, .tool-category');
    tags.forEach(tag => {
        tag.addEventListener('click', function() {
            handleTagClick(this);
        });
    });
    
    // Initialize tooltips
    initializeTooltips();
    
    // Initialize form interactions (if any)
    initializeFormInteractions();
}

// Ripple effect for buttons
function createRippleEffect(event, button) {
    const ripple = document.createElement('span');
    const rect = button.getBoundingClientRect();
    const size = Math.max(rect.width, rect.height);
    const x = event.clientX - rect.left - size / 2;
    const y = event.clientY - rect.top - size / 2;
    
    ripple.style.width = ripple.style.height = size + 'px';
    ripple.style.left = x + 'px';
    ripple.style.top = y + 'px';
    ripple.classList.add('ripple');
    
    button.appendChild(ripple);
    
    setTimeout(() => {
        ripple.remove();
    }, 600);
}

// Button click handling
function handleButtonClick(event, button) {
    const buttonText = button.textContent.trim();
    
    // Track button clicks
    trackUserEvent('button_click', {
        button_text: buttonText,
        button_type: button.className,
        timestamp: new Date().toISOString()
    });
    
    // Show click feedback
    showClickFeedback(button);
    
    // Handle specific button actions
    if (button.classList.contains('btn-primary')) {
        handlePrimaryButtonClick(button);
    } else if (button.classList.contains('btn-secondary')) {
        handleSecondaryButtonClick(button);
    } else if (button.id === 'downloadBtn') {
        handleDownloadClick();
    } else if (button.id === 'startLearningBtn') {
        handleLearningClick();
    } else if (button.id === 'getStartedBtn') {
        handleGetStartedClick();
    }
}

// Show click feedback
function showClickFeedback(button) {
    const originalText = button.textContent;
    button.textContent = '✓ ' + originalText;
    button.style.transform = 'scale(0.95)';
    
    setTimeout(() => {
        button.textContent = originalText;
        button.style.transform = 'scale(1)';
    }, 300);
}

// Handle specific button actions
function handlePrimaryButtonClick(button) {
    console.log('Primary button clicked:', button);
    // Add specific primary button logic here
}

function handleSecondaryButtonClick(button) {
    console.log('Secondary button clicked:', button);
    // Add specific secondary button logic here
}

function handleDownloadClick() {
    console.log('Download initiated');
    trackUserEvent('download_initiated', {
        source: 'hero_section',
        timestamp: new Date().toISOString()
    });
    
    // Redirect to download page
    window.location.href = 'manual/#2-installation--setup';
}

function handleLearningClick() {
    console.log('Learning initiated');
    trackUserEvent('learning_initiated', {
        source: 'hero_section',
        timestamp: new Date().toISOString()
    });
    
    // Scroll to courses section
    document.getElementById('courses').scrollIntoView({ behavior: 'smooth' });
}

function handleGetStartedClick() {
    console.log('Get started initiated');
    trackUserEvent('get_started_initiated', {
        source: 'hero_section',
        timestamp: new Date().toISOString()
    });
    
    // Redirect to getting started page
    window.location.href = 'manual/#2-installation--setup';
}

// Tag click handling
function handleTagClick(tag) {
    const tagText = tag.textContent.trim();
    
    trackUserEvent('tag_click', {
        tag_text: tagText,
        tag_type: tag.className,
        timestamp: new Date().toISOString()
    });
    
    // Show tag info
    showTagInfo(tag, tagText);
}

// Show tag information
function showTagInfo(tag, text) {
    // Create info popup
    const popup = document.createElement('div');
    popup.className = 'tag-info-popup';
    popup.innerHTML = `
        <div class="tag-info-content">
            <h4>${text}</h4>
            <p>Learn more about ${text} in our comprehensive documentation.</p>
            <button class="close-popup">Close</button>
        </div>
    `;
    
    document.body.appendChild(popup);
    
    // Position popup
    const rect = tag.getBoundingClientRect();
    popup.style.left = rect.left + 'px';
    popup.style.top = (rect.bottom + 10) + 'px';
    
    // Add show animation
    setTimeout(() => popup.classList.add('show'), 10);
    
    // Handle close
    const closeBtn = popup.querySelector('.close-popup');
    closeBtn.addEventListener('click', () => {
        popup.classList.remove('show');
        setTimeout(() => popup.remove(), 300);
    });
    
    // Auto-close after 3 seconds
    setTimeout(() => {
        if (popup.parentElement) {
            popup.classList.remove('show');
            setTimeout(() => popup.remove(), 300);
        }
    }, 3000);
}

// Initialize tooltips
function initializeTooltips() {
    const tooltipElements = document.querySelectorAll('[data-tooltip]');
    
    tooltipElements.forEach(element => {
        element.addEventListener('mouseenter', function() {
            showTooltip(this);
        });
        
        element.addEventListener('mouseleave', function() {
            hideTooltip();
        });
    });
}

// Show tooltip
function showTooltip(element) {
    const tooltipText = element.getAttribute('data-tooltip');
    
    const tooltip = document.createElement('div');
    tooltip.className = 'tooltip';
    tooltip.textContent = tooltipText;
    
    document.body.appendChild(tooltip);
    
    // Position tooltip
    const rect = element.getBoundingClientRect();
    tooltip.style.left = (rect.left + rect.width / 2 - tooltip.offsetWidth / 2) + 'px';
    tooltip.style.top = (rect.top - tooltip.offsetHeight - 10) + 'px';
    
    setTimeout(() => tooltip.classList.add('show'), 10);
}

// Hide tooltip
function hideTooltip() {
    const tooltip = document.querySelector('.tooltip');
    if (tooltip) {
        tooltip.classList.remove('show');
        setTimeout(() => tooltip.remove(), 300);
    }
}

// Form interactions
function initializeFormInteractions() {
    const forms = document.querySelectorAll('form');
    
    forms.forEach(form => {
        form.addEventListener('submit', function(e) {
            e.preventDefault();
            handleFormSubmit(this);
        });
        
        // Add input validation
        const inputs = form.querySelectorAll('input, textarea, select');
        inputs.forEach(input => {
            input.addEventListener('blur', function() {
                validateInput(this);
            });
            
            input.addEventListener('input', function() {
                clearInputError(this);
            });
        });
    });
}

// Form submission
function handleFormSubmit(form) {
    const formData = new FormData(form);
    const data = Object.fromEntries(formData);
    
    trackUserEvent('form_submit', {
        form_id: form.id,
        form_data: data,
        timestamp: new Date().toISOString()
    });
    
    // Show loading state
    const submitButton = form.querySelector('button[type="submit"]');
    const originalText = submitButton.textContent;
    submitButton.textContent = 'Submitting...';
    submitButton.disabled = true;
    
    // Simulate form submission
    setTimeout(() => {
        submitButton.textContent = '✓ Submitted!';
        submitButton.style.background = 'var(--success-color)';
        
        setTimeout(() => {
            submitButton.textContent = originalText;
            submitButton.style.background = '';
            submitButton.disabled = false;
            form.reset();
        }, 2000);
    }, 1500);
}

// Input validation
function validateInput(input) {
    const isValid = input.checkValidity();
    
    if (!isValid) {
        showInputError(input, input.validationMessage);
    }
}

// Show input error
function showInputError(input, message) {
    input.classList.add('error');
    
    const errorElement = document.createElement('div');
    errorElement.className = 'input-error';
    errorElement.textContent = message;
    
    input.parentNode.appendChild(errorElement);
}

// Clear input error
function clearInputError(input) {
    input.classList.remove('error');
    const errorElement = input.parentNode.querySelector('.input-error');
    if (errorElement) {
        errorElement.remove();
    }
}

// Performance optimizations
function initializePerformanceOptimizations() {
    // Lazy load images
    const images = document.querySelectorAll('img[data-src]');
    const imageObserver = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                const img = entry.target;
                img.src = img.dataset.src;
                img.classList.add('loaded');
                imageObserver.unobserve(img);
            }
        });
    });
    
    images.forEach(img => imageObserver.observe(img));
    
    // Preload critical resources
    preloadCriticalResources();
    
    // Implement service worker (if available)
    if ('serviceWorker' in navigator) {
        registerServiceWorker();
    }
}

// Preload critical resources
function preloadCriticalResources() {
    const criticalResources = [
        '/css/enhanced-styles.css',
        '/js/main.js'
    ];
    
    criticalResources.forEach(resource => {
        const link = document.createElement('link');
        link.rel = 'preload';
        link.as = resource.endsWith('.css') ? 'style' : 'script';
        link.href = resource;
        document.head.appendChild(link);
    });
}

// Register service worker
function registerServiceWorker() {
    navigator.serviceWorker.register('/sw.js')
        .then(registration => {
            console.log('✅ Service Worker registered:', registration);
        })
        .catch(error => {
            console.log('❌ Service Worker registration failed:', error);
        });
}

// Analytics and tracking
function initializeAnalytics() {
    // Initialize Google Analytics (or alternative)
    if (typeof gtag !== 'undefined') {
        gtag('config', 'GA_MEASUREMENT_ID', {
            page_title: document.title,
            page_location: window.location.href
        });
    }
    
    // Track page view
    trackPageView();
    
    // Track user engagement
    trackUserEngagement();
}

// Track page view
function trackPageView() {
    trackUserEvent('page_view', {
        page_title: document.title,
        page_location: window.location.href,
        page_referrer: document.referrer,
        user_agent: navigator.userAgent,
        timestamp: new Date().toISOString()
    });
}

// Track user engagement
function trackUserEngagement() {
    let timeOnPage = 0;
    let lastActivity = Date.now();
    
    // Track scroll depth
    window.addEventListener('scroll', () => {
        const scrollDepth = Math.round(
            (window.scrollY / (document.body.scrollHeight - window.innerHeight)) * 100
        );
        
        if (scrollDepth > 25 && scrollDepth <= 50) {
            trackUserEvent('scroll_depth_25', { timestamp: new Date().toISOString() });
        } else if (scrollDepth > 50 && scrollDepth <= 75) {
            trackUserEvent('scroll_depth_50', { timestamp: new Date().toISOString() });
        } else if (scrollDepth > 75) {
            trackUserEvent('scroll_depth_75', { timestamp: new Date().toISOString() });
        }
    });
    
    // Track time on page
    setInterval(() => {
        const currentTime = Date.now();
        if (currentTime - lastActivity < 30000) { // Active if less than 30 seconds of inactivity
            timeOnPage += 30;
            lastActivity = currentTime;
        }
    }, 30000);
    
    // Track time on page before leaving
    window.addEventListener('beforeunload', () => {
        trackUserEvent('time_on_page', {
            time_seconds: timeOnPage,
            timestamp: new Date().toISOString()
        });
    });
    
    // Track tab visibility
    document.addEventListener('visibilitychange', () => {
        if (document.hidden) {
            trackUserEvent('tab_hidden', { timestamp: new Date().toISOString() });
        } else {
            trackUserEvent('tab_visible', { timestamp: new Date().toISOString() });
            lastActivity = Date.now();
        }
    });
}

// Generic user event tracking
function trackUserEvent(eventName, eventData) {
    const event = {
        event: eventName,
        data: eventData,
        timestamp: new Date().toISOString(),
        user_id: getUserId(),
        session_id: getSessionId()
    };
    
    // Send to analytics (placeholder)
    sendToAnalytics(event);
    
    // Log for debugging
    console.log('📊 Event tracked:', event);
}

// Get user ID (placeholder)
function getUserId() {
    let userId = localStorage.getItem('helixcode_user_id');
    if (!userId) {
        userId = 'user_' + Math.random().toString(36).substr(2, 9);
        localStorage.setItem('helixcode_user_id', userId);
    }
    return userId;
}

// Get session ID
function getSessionId() {
    let sessionId = sessionStorage.getItem('helixcode_session_id');
    if (!sessionId) {
        sessionId = 'session_' + Math.random().toString(36).substr(2, 9);
        sessionStorage.setItem('helixcode_session_id', sessionId);
    }
    return sessionId;
}

// Send to analytics (placeholder)
function sendToAnalytics(event) {
    // In a real implementation, this would send to your analytics service
    // For now, we'll store in localStorage for demo purposes
    const events = JSON.parse(localStorage.getItem('helixcode_events') || '[]');
    events.push(event);
    localStorage.setItem('helixcode_events', JSON.stringify(events));
}

// Welcome animation
function showWelcomeAnimation() {
    const welcomeMessage = document.createElement('div');
    welcomeMessage.className = 'welcome-message';
    welcomeMessage.innerHTML = `
        <div class="welcome-content">
            <h2>🚀 Welcome to HelixCode</h2>
            <p>Experience the world's most advanced AI development platform</p>
            <button class="close-welcome">Get Started</button>
        </div>
    `;
    
    document.body.appendChild(welcomeMessage);
    
    setTimeout(() => welcomeMessage.classList.add('show'), 10);
    
    // Auto-hide after 5 seconds
    const hideTimeout = setTimeout(() => {
        hideWelcomeMessage(welcomeMessage);
    }, 5000);
    
    // Handle close button
    const closeBtn = welcomeMessage.querySelector('.close-welcome');
    closeBtn.addEventListener('click', () => {
        clearTimeout(hideTimeout);
        hideWelcomeMessage(welcomeMessage);
    });
    
    // Track welcome shown
    trackUserEvent('welcome_shown', {
        timestamp: new Date().toISOString()
    });
}

// Hide welcome message
function hideWelcomeMessage(welcomeMessage) {
    welcomeMessage.classList.remove('show');
    setTimeout(() => {
        if (welcomeMessage.parentElement) {
            welcomeMessage.remove();
        }
    }, 300);
}

// Keyboard shortcuts
function initializeKeyboardShortcuts() {
    document.addEventListener('keydown', function(e) {
        // Ctrl/Cmd + K for search
        if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
            e.preventDefault();
            openSearch();
        }
        
        // ESC to close modals
        if (e.key === 'Escape') {
            closeAllModals();
        }
        
        // Arrow keys for navigation
        if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
            navigateWithArrows(e.key);
        }
    });
}

// Open search (placeholder)
function openSearch() {
    console.log('Search opened');
    // Implement search modal
}

// Close all modals
function closeAllModals() {
    const modals = document.querySelectorAll('.modal, .tag-info-popup, .tooltip');
    modals.forEach(modal => {
        modal.classList.remove('show');
        setTimeout(() => {
            if (modal.parentElement) {
                modal.remove();
            }
        }, 300);
    });
}

// Arrow key navigation
function navigateWithArrows(key) {
    const navLinks = document.querySelectorAll('.nav-link:not([style*="display: none"])');
    const activeLink = document.querySelector('.nav-link.active');
    const currentIndex = Array.from(navLinks).indexOf(activeLink);
    
    let newIndex;
    if (key === 'ArrowLeft') {
        newIndex = currentIndex > 0 ? currentIndex - 1 : navLinks.length - 1;
    } else {
        newIndex = currentIndex < navLinks.length - 1 ? currentIndex + 1 : 0;
    }
    
    navLinks[newIndex].click();
}

// Initialize keyboard shortcuts
document.addEventListener('DOMContentLoaded', function() {
    setTimeout(initializeKeyboardShortcuts, 1000);
});

// Export functions for external use
window.HelixCodeWebsite = {
    trackUserEvent,
    showWelcomeAnimation,
    initializeWebsite
};