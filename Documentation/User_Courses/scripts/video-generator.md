# Video Generation Workflow

Complete guide for creating professional course videos from the provided scripts and materials.

## Overview

This document describes the end-to-end process for turning course scripts into polished video lessons. The workflow supports both professional studio production and DIY content creation.

## Prerequisites

### Software Requirements

**Essential:**
- Video recording software (OBS Studio, ScreenFlow, Camtasia)
- Video editing software (DaVinci Resolve, Final Cut Pro, Adobe Premiere)
- Audio recording/editing (Audacity, Adobe Audition, Logic Pro)
- Screen recording capability
- Slide presentation software (reveal.js via browser, PowerPoint, Keynote)

**Optional but Recommended:**
- Teleprompter software for script reading
- Color grading tools
- Audio noise reduction (iZotope RX, Krisp)
- Video encoding tools (HandBrake, FFmpeg)

### Hardware Requirements

**Minimum Setup:**
- Webcam (1080p minimum)
- USB microphone (Blue Yeti, Audio-Technica AT2020USB+)
- Decent computer (8GB RAM, multi-core processor)
- Good lighting (window or desk lamp)

**Professional Setup:**
- DSLR or mirrorless camera (4K capable)
- XLR microphone with audio interface (Shure SM7B, Rode NT1)
- Ring light or softbox lighting (3-point lighting setup)
- Green screen (optional, for background replacement)
- High-performance computer (16GB+ RAM, dedicated GPU)

## Production Workflow

### Phase 1: Pre-Production

#### 1.1 Script Preparation

```bash
# Navigate to course directory
cd /Users/milosvasic/Projects/HelixCode/Documentation/User_Courses/

# Review the script for the chapter you're recording
cat Course_01_Introduction/01_Welcome.md
```

**Tasks:**
- Read through the complete script multiple times
- Mark difficult pronunciations or technical terms
- Note where code demos or screen recordings are needed
- Identify sections that need visual aids
- Time yourself reading the script (should match estimated duration ±10%)

#### 1.2 Slide Generation

```bash
# Generate slides from script
cd /Users/milosvasic/Projects/HelixCode/Documentation/User_Courses/scripts/

# Generate reveal.js slides
./generate-slides.sh ../Course_01_Introduction/01_Welcome.md ../Course_01_Introduction/slides/

# Open in browser to review
open ../Course_01_Introduction/slides/01_Welcome_slides.html
```

**Tasks:**
- Review generated slides
- Customize with branding (logos, colors, fonts)
- Add animations or transitions if desired
- Export as PDF backup if needed
- Practice presenting with slides

#### 1.3 Code Example Preparation

For chapters with code demonstrations:

```bash
# Prepare code examples directory
cd Course_01_Introduction/code_examples/

# Create or verify example files
# Ensure syntax highlighting works
# Test all code examples run without errors
```

**Tasks:**
- Write and test all code examples
- Prepare any necessary project scaffolding
- Set up development environment for recording
- Create "before" and "after" states if showing transformations
- Prepare any databases, APIs, or services needed

#### 1.4 Environment Setup

**Recording Space:**
- Clean, quiet room with minimal echo
- Background: blank wall, bookshelf, or green screen
- Remove distractions and potential interruptions
- Ensure stable internet if recording with cloud services

**Technical Setup:**
- Camera at eye level, 3-5 feet away
- Lighting: key light at 45° angle, fill light on opposite side
- Microphone 6-8 inches from mouth, slightly off-axis
- Test recording equipment
- Close unnecessary applications
- Set phone to Do Not Disturb

### Phase 2: Recording

#### 2.1 Audio Recording (Voiceover Method)

If recording audio separately from video:

```bash
# Audio recording checklist:
# - Sample rate: 48kHz
# - Bit depth: 24-bit
# - Format: WAV (lossless)
# - Record in a quiet environment
# - Use pop filter to reduce plosives
```

**Recording Process:**
1. Do a mic check and test recording
2. Record room tone (30 seconds of silence) for noise reduction
3. Record the script in segments (each timestamp section)
4. Leave 2-3 seconds between segments for editing
5. If you make a mistake, pause and restart the sentence
6. Record "pickup" lines for any errors
7. Save with clear naming: `HC-C01-CH01-audio-segment-01.wav`

**Audio Editing:**
```bash
# Noise reduction workflow:
# 1. Capture noise profile from room tone
# 2. Apply noise reduction
# 3. Normalize audio levels to -3dB
# 4. Apply compression (gentle, 3:1 ratio)
# 5. EQ to enhance voice clarity
# 6. Export as WAV or high-quality MP3 (320kbps)
```

#### 2.2 Screen Recording

For demonstrations and code walkthroughs:

**OBS Studio Settings:**
```
Video:
- Base Resolution: 1920x1080
- Output Resolution: 1920x1080
- FPS: 30

Output:
- Encoder: x264 or hardware encoder (NVENC/VideoToolbox)
- Rate Control: CBR
- Bitrate: 5000-8000 Kbps
- Preset: High Quality
- Profile: High
- Keyframe Interval: 2s
```

**Recording Best Practices:**
- Hide desktop clutter and personal information
- Use large, readable fonts (16pt minimum for code)
- Zoom in when showing detailed code
- Use cursor highlighting or screen annotations
- Record at actual speed (don't speed up during recording)
- Leave pauses between actions for easier editing

#### 2.3 Presentation Recording

Recording with slides:

**Setup:**
1. Open slides in browser (fullscreen)
2. Position camera (if doing picture-in-picture)
3. Start recording software
4. Begin presentation

**Teleprompter Option:**
- Use teleprompter software with script
- Adjust scroll speed during practice
- Position near camera for natural eye line
- Practice until comfortable and natural

**Direct Recording Option:**
- Record screen with slides
- Record yourself separately (green screen or webcam)
- Composite in post-production

### Phase 3: Post-Production

#### 3.1 Video Editing Workflow

**Import and Organize:**
```
Project Structure:
├── Raw_Footage/
│   ├── audio_raw/
│   ├── screen_recordings/
│   └── camera_recordings/
├── Assets/
│   ├── intro_outro/
│   ├── lower_thirds/
│   ├── transitions/
│   └── music/
├── Sequences/
│   └── HC-C01-CH01-main.xml
└── Exports/
```

**Editing Steps:**

1. **Assembly Edit:**
   - Import all footage
   - Create sequence timeline
   - Arrange segments in order
   - Cut out mistakes, pauses, and dead air
   - Ensure good pacing (not too fast or slow)

2. **Add Visual Elements:**
   - Intro animation (5-10 seconds)
   - Lower thirds with chapter title
   - Code overlay or picture-in-picture
   - Screen annotations and callouts
   - Transitions between sections
   - Outro with next chapter preview

3. **Audio Sweetening:**
   - Sync audio with video if recorded separately
   - Add background music (subtle, 10-15% volume)
   - Apply audio ducking when speaking
   - Ensure consistent volume levels
   - Add sound effects sparingly (whooshes, clicks)

4. **Color Correction:**
   - Adjust white balance
   - Correct exposure
   - Apply color grade for consistency
   - Match footage from different cameras

5. **Graphics and Text:**
   - Add chapter title card
   - Show learning objectives
   - Display code snippets when mentioned
   - Show URLs or resources
   - Add timestamps for sections

#### 3.2 Subtitle Integration

```bash
# Generate subtitles from script
cd /Users/milosvasic/Projects/HelixCode/Documentation/User_Courses/scripts/

./generate-subtitles.sh ../Course_01_Introduction/01_Welcome.md ../videos/course_01/chapter_01.srt

# Review and adjust timing
# Import SRT into video editor
# Position and style subtitles
```

**Subtitle Styling:**
- Font: Arial or Helvetica, bold
- Size: 48-60pt (readable but not obtrusive)
- Color: White with black outline or semi-transparent background
- Position: Bottom center, above lower third
- Duration: Minimum 1 second per subtitle

#### 3.3 Export Settings

**YouTube/Web Optimal Settings:**

```
Container: MP4
Video Codec: H.264 (x264)
Resolution: 1920x1080
Frame Rate: 30 fps (use source fps)
Bitrate: 8-12 Mbps (VBR, 2-pass)
Audio Codec: AAC
Audio Bitrate: 192 kbps
Sample Rate: 48 kHz
Channels: Stereo

Advanced:
- Profile: High
- Level: 4.2
- Keyframe Interval: 2 seconds
- Color Space: Rec. 709
```

**File Naming Convention:**
```
HC-C{course}-CH{chapter}-{title}-{version}.mp4

Examples:
HC-C01-CH01-Welcome-v1.mp4
HC-C02-CH02-Installation_Steps-v2.mp4
```

**Export Checklist:**
- [ ] Video plays without stuttering
- [ ] Audio is clear and synchronized
- [ ] Subtitles display correctly
- [ ] No copyright issues (music, images)
- [ ] Color and exposure consistent
- [ ] File size reasonable (<500MB for 10min video)
- [ ] Meets technical specifications

### Phase 4: Quality Assurance

#### 4.1 Review Checklist

**Content Accuracy:**
- [ ] All information is correct and up-to-date
- [ ] Code examples work as shown
- [ ] Commands and syntax are accurate
- [ ] No misleading or confusing statements
- [ ] Learning objectives are met

**Technical Quality:**
- [ ] Video resolution is 1080p minimum
- [ ] Audio is clear with no distortion
- [ ] No background noise or echo
- [ ] Proper exposure and color
- [ ] Smooth transitions
- [ ] Professional appearance

**Accessibility:**
- [ ] Subtitles are accurate and well-timed
- [ ] Text is large enough to read
- [ ] Sufficient contrast for readability
- [ ] Narration describes visual content
- [ ] Pacing allows comprehension

**Engagement:**
- [ ] Introduction hooks the viewer
- [ ] Content flows logically
- [ ] Examples are clear and relevant
- [ ] Pacing maintains interest
- [ ] Call to action at end

#### 4.2 Test Viewing

1. Watch entire video start to finish
2. Check on different devices (desktop, tablet, mobile)
3. Test at different playback speeds (1.25x, 1.5x)
4. Verify subtitles on multiple players
5. Get feedback from 2-3 test viewers

### Phase 5: Publishing

#### 5.1 Metadata Preparation

```bash
# Create metadata file for the video
cd /Users/milosvasic/Projects/HelixCode/Documentation/User_Courses/metadata/

# Copy template and fill in details
cp video_metadata_template.json HC-C01-CH01-metadata.json
```

Fill in:
- Video title and description
- Duration and technical specs
- Timestamps for chapters
- Tags and keywords
- Asset file paths

#### 5.2 Upload Process

**YouTube:**
```yaml
Title: "HelixCode Course 1: Welcome to HelixCode"
Description: |
  Welcome to the HelixCode video course series! In this chapter, we introduce
  the course structure, learning objectives, and set expectations for your
  HelixCode journey.

  Timestamps:
  00:00 - Opening
  00:30 - What You'll Learn
  01:15 - Course Structure Overview
  ...

  Resources:
  - Course Materials: [link]
  - HelixCode GitHub: https://github.com/helix-editor/helixcode
  - Documentation: [link]

Tags: helixcode, ai development, coding assistant, tutorial, course
Category: Education
Language: English
Subtitles: Upload .srt file
Thumbnail: Custom thumbnail (1280x720)
Playlist: "HelixCode Course 1: Introduction"
```

**Self-Hosted:**
- Upload to CDN or video hosting service
- Ensure adaptive bitrate streaming if possible
- Test playback from various locations
- Monitor bandwidth usage

#### 5.3 Companion Materials

For each video, provide:

```bash
# Create resources directory
mkdir -p Course_01_Introduction/resources/

# Add downloadable materials:
# - PDF of slides
# - Code examples (zip)
# - Transcript (text file)
# - Exercise worksheet
# - Quick reference guide
```

## Batch Processing

For processing multiple videos efficiently:

### Generate All Slides

```bash
#!/bin/bash
cd /Users/milosvasic/Projects/HelixCode/Documentation/User_Courses/

for course in Course_*/; do
    for script in "$course"*.md; do
        [ -f "$script" ] || continue
        echo "Generating slides for: $script"
        ./scripts/generate-slides.sh "$script" "${course}slides/"
    done
done
```

### Generate All Subtitles

```bash
#!/bin/bash
cd /Users/milosvasic/Projects/HelixCode/Documentation/User_Courses/

for course in Course_*/; do
    course_num=$(echo "$course" | grep -o '[0-9]\+')
    for script in "$course"*.md; do
        [ -f "$script" ] || continue
        chapter_num=$(basename "$script" | grep -o '^[0-9]\+')
        output="videos/course_${course_num}/chapter_${chapter_num}.srt"
        echo "Generating subtitles: $script -> $output"
        ./scripts/generate-subtitles.sh "$script" "$output"
    done
done
```

## Quality Standards

### Video Standards

- **Resolution:** 1920x1080 minimum (1080p)
- **Frame Rate:** 30fps (consistent)
- **Bitrate:** 5-12 Mbps
- **Audio:** 192 kbps AAC, 48kHz stereo
- **Format:** MP4 (H.264)
- **Max File Size:** ~100MB per 10 minutes

### Content Standards

- **Duration Accuracy:** Within 20% of estimated time
- **Audio Clarity:** Clear speech, minimal noise
- **Visual Quality:** Sharp, well-lit, professional
- **Pacing:** Not rushed, allows comprehension
- **Accuracy:** All technical content verified

### Accessibility Standards

- **Subtitles:** Required for all videos
- **Contrast Ratio:** 4.5:1 minimum for text
- **Font Size:** Large enough for mobile viewing
- **Narration:** Describes important visual elements
- **Keyboard Navigation:** All interactive elements accessible

## Tools and Resources

### Recommended Free Tools

- **OBS Studio** - Screen recording and streaming
- **DaVinci Resolve** - Professional video editing
- **Audacity** - Audio editing
- **GIMP** - Image editing for thumbnails
- **Kdenlive** - Alternative video editor (Linux)

### Recommended Paid Tools

- **Camtasia** - Screen recording + editing (beginner-friendly)
- **Adobe Premiere Pro** - Professional video editing
- **Final Cut Pro** - Professional editing (macOS)
- **ScreenFlow** - Screen recording + editing (macOS)
- **Adobe Audition** - Professional audio editing

### Online Resources

- **Pexels/Unsplash** - Free stock images
- **Freesound** - Free sound effects
- **Incompetech** - Royalty-free music
- **Flaticon** - Free icons for graphics
- **Canva** - Thumbnail and graphic design

## Troubleshooting

### Common Issues

**Audio sync drift:**
- Record audio and video at same sample rate (48kHz)
- Use clapperboard or sync point at start
- Convert to constant frame rate before editing

**Large file sizes:**
- Use 2-pass encoding for better compression
- Lower bitrate if quality is acceptable
- Trim unnecessary footage
- Use H.265 (HEVC) for better compression (compatibility trade-off)

**Stuttering playback:**
- Render preview/proxies for complex timelines
- Export with constant frame rate
- Check keyframe interval (2 seconds recommended)

**Subtitle timing issues:**
- Verify timestamps in script are accurate
- Adjust SRT file manually if needed
- Account for intro/outro in timing

## Version Control

Track video versions in metadata:

```json
{
  "versioning": {
    "version": "1.2.0",
    "changelog": [
      {
        "version": "1.0.0",
        "date": "2025-11-06",
        "changes": "Initial release"
      },
      {
        "version": "1.1.0",
        "date": "2025-12-01",
        "changes": "Updated HelixCode version references, fixed typo at 3:45"
      },
      {
        "version": "1.2.0",
        "date": "2026-01-15",
        "changes": "Added new provider information, re-recorded section 4"
      }
    ]
  }
}
```

## Conclusion

This workflow ensures consistent, professional quality across all course videos. Adapt these guidelines to your specific setup and constraints while maintaining the quality standards outlined above.

For questions or issues with the video production process, consult the HelixCode documentation team or open an issue in the GitHub repository.
