# HelixCode User Manual Documentation System

This directory contains the HelixCode user manual and related documentation tools.

## Overview

The HelixCode documentation system provides:

- **Styled HTML Manual**: Professional, responsive HTML version with search and navigation
- **Automatic Sync**: Scripts to sync manual to GitHub Pages Website
- **Markdown to HTML Conversion**: Go-based converter for custom Markdown processing
- **PDF Generation**: Optional PDF export using wkhtmltopdf

## Files and Directories

```
Documentation/User_Manual/
├── manual.html              # Main HTML manual (styled and interactive)
├── images/                  # Supporting images and assets
└── README.md               # This file
```

## Scripts

### 1. Generate Manual (`scripts/generate-manual.sh`)

Generates the HTML manual from README.md using the Go-based converter.

```bash
cd HelixCode
./scripts/generate-manual.sh
```

**Features:**
- Converts Markdown to beautifully styled HTML
- Generates table of contents automatically
- Adds syntax highlighting
- Creates responsive layout

### 2. Sync Manual (`scripts/sync-manual.sh`)

Syncs the generated manual to the GitHub Pages Website.

```bash
cd HelixCode
./scripts/sync-manual.sh
```

**Features:**
- Copies `manual.html` to `GitHub-Pages-Website/docs/manual/index.html`
- Copies all images to the destination
- Generates PDF version (if wkhtmltopdf is installed)
- Creates README and metadata files
- Logs all operations to `scripts/sync-manual.log`
- Checks for git changes and provides commit commands

### 3. Markdown to HTML Converter (`scripts/md-to-html`)

Go program that converts Markdown to styled HTML.

```bash
cd HelixCode/scripts
./md-to-html -input=../README.md -output=../Documentation/User_Manual/manual.html
```

**Options:**
- `-input`: Input Markdown file (required)
- `-output`: Output HTML file (required)
- `-title`: HTML document title (default: "HelixCode User Manual")
- `-theme`: Theme (light/dark, default: "light")

## Manual Features

The generated HTML manual includes:

### Navigation
- Fixed header with logo and search
- Sidebar table of contents with jump links
- Smooth scrolling to sections
- Active section highlighting

### Styling
- Professional CSS matching GitHub Pages website theme
- Responsive design (mobile-friendly)
- Light/Dark theme toggle
- Syntax highlighting for code blocks (highlight.js)
- Custom alert boxes (info, warning, danger, success)

### Functionality
- Full-text search across documentation
- Mobile menu toggle
- Automatic anchor links for headings
- Interactive TOC with active state
- Timestamp generation

## Workflow

### Complete Documentation Update Workflow

1. **Update Content**: Edit `README.md` or create content in `Documentation/User_Manual/`

2. **Generate Manual** (optional if using pre-built `manual.html`):
   ```bash
   ./scripts/generate-manual.sh
   ```

3. **Sync to GitHub Pages**:
   ```bash
   ./scripts/sync-manual.sh
   ```

4. **Review Changes**:
   - Check the log file: `scripts/sync-manual.log`
   - Review generated files in `Github-Pages-Website/docs/manual/`

5. **Commit and Push**:
   ```bash
   cd ../Github-Pages-Website
   git add docs/manual/
   git commit -m "Update user manual - $(date)"
   git push
   ```

## PDF Generation

To generate PDF versions, install wkhtmltopdf:

### macOS
```bash
brew install --cask wkhtmltopdf
```

### Linux
```bash
sudo apt-get install wkhtmltopdf
```

The sync script will automatically generate PDFs if wkhtmltopdf is available.

## Customization

### Updating Styles

The HTML manual includes embedded CSS. To update styles:

1. Edit the `<style>` section in `manual.html`
2. Re-run the sync script to deploy changes

### Adding Images

1. Place images in `Documentation/User_Manual/images/`
2. Reference them in the manual with relative paths
3. The sync script will automatically copy them to GitHub Pages

### Custom Themes

Edit CSS variables in `manual.html`:

```css
:root {
    --primary-500: #0ea5e9;    /* Primary color */
    --accent-1: #10b981;       /* Success color */
    --accent-2: #f59e0b;       /* Warning color */
    --accent-3: #ef4444;       /* Error color */
    /* ... */
}
```

## Troubleshooting

### Issue: "manual.html not found"
**Solution**: Run `./scripts/generate-manual.sh` first

### Issue: "GitHub Pages directory does not exist"
**Solution**: Ensure `Github-Pages-Website` is cloned at the same level as `HelixCode`

### Issue: "PDF generation failed"
**Solution**: Install wkhtmltopdf or skip PDF generation (manual will still sync)

### Issue: "Permission denied"
**Solution**: Make scripts executable:
```bash
chmod +x scripts/*.sh
```

## Integration with Website

The manual is automatically integrated with the GitHub Pages website:

- **URL**: `https://your-username.github.io/Github-Pages-Website/docs/manual/`
- **Location**: `docs/manual/index.html`
- **Assets**: `docs/manual/images/`
- **PDF**: `docs/manual/HelixCode_User_Manual_Latest.pdf`

The manual inherits the website's design language and theme.

## Version Control

The manual tracks:
- Generation timestamp (automatically added)
- Sync metadata (`.sync-metadata.json`)
- Version number (from README.md or manual metadata)

## Future Enhancements

Potential improvements:
- [ ] Watch mode for automatic regeneration
- [ ] Multi-language support
- [ ] Version history and changelog
- [ ] Interactive examples and demos
- [ ] API documentation integration
- [ ] Search index generation

## Support

For issues or questions:
- GitHub Issues: https://github.com/your-org/helixcode/issues
- Documentation: https://docs.helixcode.dev

---

**Last Updated**: 2025-11-06
**Version**: 1.0
