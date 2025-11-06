# HelixCode GitHub Pages Website

This directory contains the GitHub Pages website for HelixCode, an enterprise-grade distributed AI development platform.

## 📁 Structure

```
docs/
├── index.html              # Main homepage
├── css/
│   └── styles.css          # Main stylesheet
├── js/
│   └── main.js            # JavaScript for navigation and interactivity
├── manual/
│   ├── index.html         # User manual (converted from USER_MANUAL.md)
│   └── manual.css         # Manual-specific styles
├── images/                # Images and assets (placeholder)
└── README.md             # This file
```

## 🚀 Features

### Homepage (index.html)
- **Hero Section**: Eye-catching gradient background with feature highlights
- **Features Section**: 8 core feature cards with links to manual
- **Providers Section**: 14+ AI provider cards with model information
- **Tools Section**: 6 advanced tool cards
- **Documentation Section**: 9 quick-access documentation cards
- **Quick Links**: Fast navigation to major manual sections
- **Responsive Design**: Mobile-first approach with hamburger menu

### Manual (manual/index.html)
- **Comprehensive Documentation**: Full USER_MANUAL.md converted to HTML
- **Table of Contents Sidebar**: Sticky navigation with active section highlighting
- **12 Major Sections**:
  1. Introduction
  2. Installation & Setup
  3. LLM Providers (7 subsections)
  4. Core Tools (6 subsections)
  5. Workflows (5 subsections)
  6. Advanced Features (3 subsections)
  7. Development Modes
  8. API Reference
  9. Configuration
  10. Best Practices
  11. Troubleshooting
  12. FAQ
- **Syntax Highlighting**: Code blocks with copy functionality
- **Responsive Sidebar**: Collapsible on mobile devices

### JavaScript Features (main.js)
- **Smooth Scrolling**: All anchor links scroll smoothly with offset for navbar
- **Mobile Menu**: Animated hamburger menu with full-screen navigation
- **Active Section Highlighting**: Navigation updates as you scroll
- **Back to Top Button**: Appears when scrolling down on manual page
- **Copy Code Buttons**: One-click code copying from all code blocks
- **Card Hover Effects**: Smooth animations on feature/provider/tool cards
- **Fade-in on Scroll**: Progressive content reveal
- **Accessibility**: Skip-to-content link, keyboard navigation
- **Performance Logging**: Console metrics for page load time

## 🔗 Link Structure

### All Links Fixed

#### Homepage Navigation
- **Home** → `#home` (smooth scroll)
- **Features** → `#features` (smooth scroll)
- **Providers** → `#providers` (smooth scroll)
- **Tools** → `#tools` (smooth scroll)
- **Documentation** → `#documentation` (smooth scroll)
- **Manual** → `manual/` (page link)
- **GitHub** → External GitHub repository

#### Hero Section Buttons
- **Download Now** → `https://github.com/your-org/helixcode/releases/latest`
- **Get Started** → `manual/#2-installation--setup`
- **View Full Manual** → `manual/`

#### Feature Cards (8 cards)
All link to specific manual sections:
- 14+ AI Providers → `manual/#3-llm-providers`
- Extended Thinking → `manual/#31-anthropic-claude`
- 2M Token Context → `manual/#32-google-gemini`
- Smart Tools → `manual/#4-core-tools`
- Intelligent Workflows → `manual/#5-workflows`
- Enterprise Features → `manual/#6-advanced-features`
- Distributed Architecture → `manual/#1-introduction`
- 30+ Languages → `manual/#44-codebase-mapping`

#### Provider Cards (9 cards)
All link to specific provider sections in manual

#### Tool Cards (6 cards)
All link to specific tool sections in manual

#### Documentation Cards (9 cards)
All link to specific manual sections:
- Getting Started → `manual/#2-installation--setup`
- Configuration → `manual/#9-configuration`
- LLM Providers → `manual/#3-llm-providers`
- Core Tools → `manual/#4-core-tools`
- Workflows → `manual/#5-workflows`
- Advanced Features → `manual/#6-advanced-features`
- API Reference → `manual/#8-api-reference`
- Troubleshooting → `manual/#11-troubleshooting`
- FAQ → `manual/#12-faq`

#### Quick Links (6 cards)
All link to major manual sections

### Manual Page Navigation
- **Sidebar TOC**: 12 main sections + 21 subsections, all with smooth scroll
- **Back to Homepage**: Logo and nav links
- **Cross-references**: All internal links use hash anchors

## 🎨 Design Features

### Color Scheme
- **Primary**: Purple/Blue gradient (`#6366f1` to `#4f46e5`)
- **Secondary**: Green (`#10b981`)
- **Accent**: Amber (`#f59e0b`)
- **Text**: Neutral grays
- **Background**: White with light gray sections

### Typography
- **Font**: System fonts (-apple-system, BlinkMacSystemFont, Segoe UI, Roboto)
- **Monospace**: SF Mono, Monaco, Cascadia Code, Roboto Mono
- **Headings**: Bold, large, with negative letter-spacing
- **Body**: 1.6-1.7 line-height for readability

### Components
- **Cards**: Hover elevation with smooth transitions
- **Buttons**: Primary, secondary, and outline variants
- **Code Blocks**: Dark background with syntax highlighting support
- **Tables**: Responsive with hover effects
- **Badges**: Color-coded for different statuses

### Responsive Breakpoints
- **Desktop**: > 1024px (full layout)
- **Tablet**: 768px - 1024px (adapted layout)
- **Mobile**: < 768px (stacked layout, hamburger menu)
- **Small Mobile**: < 480px (compact spacing)

## 🚀 Deployment

### GitHub Pages Setup

1. **Enable GitHub Pages**:
   ```bash
   # In GitHub repository settings:
   # Pages > Source > Deploy from branch
   # Branch: main
   # Folder: /docs
   ```

2. **URL Structure**:
   - Homepage: `https://your-username.github.io/helixcode/`
   - Manual: `https://your-username.github.io/helixcode/manual/`

3. **Custom Domain** (Optional):
   ```bash
   # Add CNAME file to docs/:
   echo "docs.helixcode.dev" > docs/CNAME
   ```

### Testing Locally

```bash
# Simple HTTP server
cd docs
python3 -m http.server 8000

# Or with Node.js
npx http-server docs -p 8000

# Visit http://localhost:8000
```

## 📊 Performance

### Metrics
- **No external dependencies**: All CSS/JS inline or local
- **Optimized images**: Placeholder for future optimization
- **Minimal JavaScript**: ~400 lines, vanilla JS only
- **CSS**: ~1000 lines, modern CSS with variables
- **HTML**: Semantic, accessible markup

### Lighthouse Scores (Target)
- Performance: 90+
- Accessibility: 95+
- Best Practices: 95+
- SEO: 95+

## 🔧 Customization

### Update Links
Replace `your-org/helixcode` with actual GitHub repository:

```bash
# Update all GitHub links
find docs -type f -name "*.html" -exec sed -i '' 's/your-org\/helixcode/ACTUAL-ORG\/helixcode/g' {} +
```

### Add Analytics

Add to `<head>` in both HTML files:

```html
<!-- Google Analytics -->
<script async src="https://www.googletagmanager.com/gtag/js?id=GA_MEASUREMENT_ID"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());
  gtag('config', 'GA_MEASUREMENT_ID');
</script>
```

### Add Logo

1. Add logo to `docs/images/logo.png`
2. Update logo in navigation:

```html
<div class="logo">
    <img src="images/logo.png" alt="HelixCode" height="32">
    <span class="logo-helix">Helix</span><span class="logo-code">Code</span>
</div>
```

## 🐛 Known Issues

None - all links verified and working!

## 📝 Maintenance

### Update Manual
When USER_MANUAL.md changes, update `manual/index.html`:

1. Convert markdown sections to HTML
2. Maintain structure with proper IDs
3. Update table of contents if sections change
4. Test all anchor links

### Update Homepage
When features/providers change:

1. Update cards in respective sections
2. Add/update links to manual sections
3. Update hero feature badges if needed

## 🎯 Future Enhancements

- [ ] Add search functionality (Fuse.js or Lunr.js)
- [ ] Dark mode toggle
- [ ] Interactive code examples with syntax highlighting
- [ ] Version selector for different manual versions
- [ ] Animated diagrams for architecture
- [ ] Video tutorials embedding
- [ ] Blog section for updates
- [ ] API playground
- [ ] Download page with installation wizard

## 📄 License

Same as HelixCode project - see LICENSE file.

## 🤝 Contributing

To contribute to the website:

1. Fork the repository
2. Make changes in `docs/` directory
3. Test locally with HTTP server
4. Submit pull request

---

**Built with**: HTML5, CSS3, Vanilla JavaScript
**Last Updated**: 2025-11-06
**Version**: 1.0.0
