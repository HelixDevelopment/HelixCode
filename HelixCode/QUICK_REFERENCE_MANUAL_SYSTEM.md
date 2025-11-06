# HelixCode Manual System - Quick Reference

## Quick Start

### Generate and Sync in One Go
```bash
cd HelixCode
./scripts/sync-manual.sh
```

## Files Overview

| File | Purpose | Size |
|------|---------|------|
| `Documentation/User_Manual/manual.html` | Main HTML manual | 45KB |
| `scripts/sync-manual.sh` | Sync to GitHub Pages | 12KB |
| `scripts/generate-manual.sh` | Generate from README | 1.6KB |
| `scripts/md-to-html` | Go converter binary | 3.0MB |
| `scripts/md-to-html.go` | Go converter source | 8.7KB |

## Common Commands

### Sync Manual to Website
```bash
cd HelixCode
./scripts/sync-manual.sh
```

### Generate Manual from README
```bash
cd HelixCode
./scripts/generate-manual.sh
```

### Use Converter Directly
```bash
cd HelixCode/scripts
./md-to-html -input=../README.md -output=output.html
```

### Build Converter
```bash
cd HelixCode/scripts
go build -o md-to-html md-to-html.go
```

### View Log
```bash
cat scripts/sync-manual.log
```

## Features at a Glance

### HTML Manual
- ✅ Responsive design
- ✅ Search functionality
- ✅ Dark/Light themes
- ✅ Sidebar navigation
- ✅ Syntax highlighting
- ✅ Mobile-friendly

### Sync Script
- ✅ Copies HTML to GitHub Pages
- ✅ Syncs images
- ✅ Generates PDFs
- ✅ Updates timestamps
- ✅ Logs operations
- ✅ Detects git changes

### Converter
- ✅ Markdown to HTML
- ✅ Auto TOC generation
- ✅ Code highlighting
- ✅ Custom styling

## File Locations

### Source
- Manual: `HelixCode/Documentation/User_Manual/manual.html`
- Images: `HelixCode/Documentation/User_Manual/images/`

### Destination (after sync)
- Manual: `Github-Pages-Website/docs/manual/index.html`
- Images: `Github-Pages-Website/docs/manual/images/`
- PDF: `Github-Pages-Website/docs/manual/HelixCode_User_Manual_Latest.pdf`

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Permission denied | `chmod +x scripts/*.sh` |
| Manual not found | File exists at `Documentation/User_Manual/manual.html` |
| No GitHub Pages dir | Clone at `../Github-Pages-Website` |
| PDF generation fails | Install `wkhtmltopdf` or skip PDF |

## Installation Requirements

### Required
- Bash shell
- Go 1.16+ (for building converter)

### Optional
- wkhtmltopdf (for PDF generation)
- Git (for version control)

## Customization

### Update Colors
Edit CSS variables in `manual.html`:
```css
:root {
    --primary-500: #0ea5e9;
    --accent-1: #10b981;
    /* ... */
}
```

### Add Content
Edit the `<main>` section in `manual.html`.

### Add Images
1. Place in `Documentation/User_Manual/images/`
2. Run `./scripts/sync-manual.sh`

## Support

- Full Guide: `Documentation/User_Manual/README.md`
- Summary: `DOCUMENTATION_SYSTEM_SUMMARY.md`
- Website: https://docs.helixcode.dev

---

**Version**: 1.0
**Last Updated**: 2025-11-06
