# HelixCode Course System - Quick Start Guide

Get started creating video course content in 5 minutes.

## For Course Creators

### 1. Choose Your Chapter

Navigate to the course directory:
```bash
cd /Users/milosvasic/Projects/HelixCode/Documentation/User_Courses/
ls Course_*/
```

### 2. Review the Script

Open any chapter script to understand the format:
```bash
cat Course_01_Introduction/01_Welcome.md
```

**Key sections in every script:**
- Duration and learning objectives
- Video script with timestamps
- Slide outline (6-9 slides)
- Chapter exercises (3-4 activities)
- Code examples
- Additional resources

### 3. Generate Slides

Use the automated script:
```bash
cd scripts/
./generate-slides.sh ../Course_01_Introduction/01_Welcome.md ../Course_01_Introduction/slides/

# Open the generated HTML in your browser
open ../Course_01_Introduction/slides/01_Welcome_slides.html
```

**What this does:**
- Extracts slide content from the script
- Creates a standalone HTML presentation
- Uses reveal.js for professional appearance
- Ready for presentation or recording

### 4. Generate Subtitles

Create SRT subtitles from timestamps:
```bash
./generate-subtitles.sh ../Course_01_Introduction/01_Welcome.md ../videos/course_01/chapter_01.srt

# Check the output
cat ../videos/course_01/chapter_01.srt
```

**Output includes:**
- Properly formatted SRT file
- WebVTT file for web players
- Validated timing
- Line length optimization

### 5. Record Your Video

Follow the comprehensive guide:
```bash
cat scripts/video-generator.md
```

**Quick recording steps:**
1. Set up your recording environment
2. Use the script for narration
3. Show the generated slides
4. Demonstrate code examples
5. Record in segments (easier editing)

### 6. Edit and Export

**Recommended settings:**
- Resolution: 1920x1080
- Format: MP4 (H.264)
- Frame rate: 30 fps
- Audio: AAC, 192 kbps

### 7. Add Metadata

Copy the template and fill in details:
```bash
cd metadata/
cp video_metadata_template.json HC-C01-CH01-metadata.json
# Edit with your video details
```

## For Content Writers

### Creating a New Chapter

1. **Copy an existing chapter as a template:**
   ```bash
   cp Course_01_Introduction/01_Welcome.md Course_04_AI_Providers/01_Provider_Overview.md
   ```

2. **Edit the header:**
   ```markdown
   # Chapter 1: Provider Overview

   **Duration:** X minutes
   **Learning Objectives:**
   - Objective 1
   - Objective 2
   ```

3. **Write the video script:**
   - Use timestamp format: `[MM:SS - MM:SS]`
   - Break into 1-2 minute segments
   - Write conversationally
   - Include demos and examples

4. **Create slide outline:**
   - 6-9 slides per chapter
   - Clear titles and bullet points
   - Use **bold** for emphasis

5. **Add exercises:**
   - 3-4 practical exercises
   - Varied difficulty
   - Reinforce learning objectives

## For Learners

### How to Use This Course System

1. **Start with Course 1:**
   ```bash
   cat Course_01_Introduction/README.md
   ```

2. **Read the script or watch the video:**
   - Scripts contain full narration
   - Videos bring concepts to life
   - Both are valuable resources

3. **Review the slides:**
   - Open HTML slides in browser
   - Use as reference while coding
   - Present/fullscreen mode with F key

4. **Complete exercises:**
   - Found at the end of each chapter
   - Hands-on practice
   - Builds real skills

5. **Try the code examples:**
   ```bash
   cd Course_XX_Name/code_examples/
   # Follow the README in each project
   ```

## Batch Operations

### Generate All Slides for a Course

```bash
#!/bin/bash
course_dir="Course_01_Introduction"
slides_dir="$course_dir/slides"
mkdir -p "$slides_dir"

for script in $course_dir/*.md; do
    [ -f "$script" ] || continue
    [ "$(basename "$script")" = "README.md" ] && continue
    echo "Processing: $script"
    ./scripts/generate-slides.sh "$script" "$slides_dir/"
done
```

### Generate All Subtitles for a Course

```bash
#!/bin/bash
course_num="01"
course_dir="Course_${course_num}_Introduction"
videos_dir="videos/course_${course_num}"
mkdir -p "$videos_dir"

chapter_num=1
for script in $course_dir/[0-9]*.md; do
    [ -f "$script" ] || continue
    echo "Processing: $script"
    output="${videos_dir}/chapter_$(printf %02d $chapter_num).srt"
    ./scripts/generate-subtitles.sh "$script" "$output"
    ((chapter_num++))
done
```

### Validate All Scripts

```bash
#!/bin/bash
# Check that all scripts have required sections
for script in Course_*/*.md; do
    [ -f "$script" ] || continue
    [ "$(basename "$script")" = "README.md" ] && continue

    echo "Checking: $script"

    grep -q "^## Video Script" "$script" || echo "  ⚠ Missing Video Script section"
    grep -q "^## Slide Outline" "$script" || echo "  ⚠ Missing Slide Outline section"
    grep -q "^## Chapter Exercises" "$script" || echo "  ⚠ Missing Exercises section"
    grep -q "\[.*:.*\]" "$script" || echo "  ⚠ Missing timestamps"
done
```

## Common Tasks

### Update Course Duration

Edit `metadata/course_index.json`:
```json
{
  "course_id": "HC-C01",
  "duration_minutes": 30,
  "chapters": [
    {
      "chapter_id": "HC-C01-CH01",
      "duration_minutes": 5
    }
  ]
}
```

### Add a New Course

1. Create directory structure:
   ```bash
   mkdir -p Course_09_New_Topic/{slides,code_examples,audio_scripts}
   ```

2. Add to `metadata/course_index.json`:
   ```json
   {
     "course_id": "HC-C09",
     "title": "New Topic",
     "description": "...",
     "chapters": []
   }
   ```

3. Create chapter scripts following the template

### Preview Your Work

**Slides:**
```bash
open Course_01_Introduction/slides/01_Welcome_slides.html
```

**Subtitles:**
```bash
cat videos/course_01/chapter_01.srt | less
```

**Metadata:**
```bash
cat metadata/course_index.json | python3 -m json.tool | less
```

## Troubleshooting

### "Permission denied" when running scripts

```bash
chmod +x scripts/*.sh
```

### "pandoc not found" error

```bash
# macOS
brew install pandoc

# Linux (Ubuntu/Debian)
sudo apt-get install pandoc

# Linux (Fedora)
sudo dnf install pandoc
```

### Subtitle timing seems off

1. Check that timestamps in script use `[MM:SS - MM:SS]` format
2. Verify no overlapping time ranges
3. Manually adjust SRT file if needed

### Slides not rendering correctly

1. Ensure reveal.js CDN is accessible
2. Check browser console for errors
3. Try a different browser
4. Validate Markdown syntax in slide outline

## Resources

### Templates
- Script template: `Course_01_Introduction/01_Welcome.md`
- Metadata template: `metadata/video_metadata_template.json`
- Project template: `Course_08_Real_World_Projects/code_examples/project_1_rest_api/`

### Documentation
- Full README: `README.md`
- Video production guide: `scripts/video-generator.md`
- System summary: `COURSE_SYSTEM_SUMMARY.md`

### Tools
- Slide generator: `scripts/generate-slides.sh`
- Subtitle generator: `scripts/generate-subtitles.sh`
- Course catalog: `metadata/course_index.json`

## Getting Help

### Course Content Questions
- Review existing chapters for examples
- Check the course catalog metadata
- Consult HelixCode documentation

### Technical Issues
- Read `scripts/video-generator.md`
- Check script comments for usage
- Review error messages carefully

### Contributing
- Follow existing patterns
- Maintain consistent formatting
- Test generated output
- Update metadata as needed

## Quick Reference

### File Naming
- Scripts: `##_Chapter_Name.md` (e.g., `01_Welcome.md`)
- Slides: `##_Chapter_Name_slides.html`
- Subtitles: `chapter_##.srt`
- Metadata: `HC-C##-CH##-metadata.json`

### Directory Structure
```
Course_##_Name/
├── ##_Chapter.md           (script)
├── slides/
│   └── ##_Chapter_slides.html
├── code_examples/
└── audio_scripts/
```

### Command Cheat Sheet
```bash
# Generate slides
./scripts/generate-slides.sh <script.md> <output_dir>

# Generate subtitles
./scripts/generate-subtitles.sh <script.md> <output.srt>

# List all courses
ls -d Course_*/

# Count chapters in a course
ls Course_01_Introduction/[0-9]*.md | wc -l

# View course metadata
cat metadata/course_index.json | python3 -m json.tool
```

## Next Steps

1. **Familiarize yourself** with the existing content
2. **Choose a task:**
   - Complete existing courses
   - Create new chapters
   - Improve documentation
   - Test the tools
3. **Follow the workflow** outlined above
4. **Contribute back** improvements and feedback

---

**Ready to create?** Start with Course 1 and work your way through!

**Need help?** Consult the full documentation in `README.md` and `scripts/video-generator.md`.

**Found a bug?** Open an issue in the HelixCode repository.
