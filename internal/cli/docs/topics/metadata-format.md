# Metadata Directory Format

## Directory Structure

The metadata directory follows a locale-based structure compatible with Fastlane:

```
metadata/
  en-US/
    title.txt
    short_description.txt
    full_description.txt
    video_url.txt
    images/
      phoneScreenshots/
        1.png
        2.png
      sevenInchScreenshots/
        1.png
      tenInchScreenshots/
        1.png
      tvScreenshots/
      wearScreenshots/
  ja-JP/
    title.txt
    short_description.txt
    full_description.txt
  fr-FR/
    title.txt
    short_description.txt
    full_description.txt
```

## File Descriptions

| File | Max Length | Required | Description |
|------|-----------|----------|-------------|
| `title.txt` | 30 chars | Yes | App title displayed on Play Store |
| `short_description.txt` | 80 chars | Yes | Brief description shown in search results |
| `full_description.txt` | 4000 chars | Yes | Full app description on the store page |
| `video_url.txt` | - | No | YouTube video URL for the store listing |

## Character Limits

Google Play enforces strict character limits:

- **Title**: Maximum 30 characters
- **Short description**: Maximum 80 characters
- **Full description**: Maximum 4000 characters
- **Release notes**: Maximum 500 characters per locale

## Screenshot Types

| Directory | Device Type | Max Count |
|-----------|------------|-----------|
| `phoneScreenshots/` | Phone | 8 |
| `sevenInchScreenshots/` | 7-inch tablet | 8 |
| `tenInchScreenshots/` | 10-inch tablet | 8 |
| `tvScreenshots/` | TV | 8 |
| `wearScreenshots/` | Wear OS | 8 |

Minimum 2 screenshots are recommended per device type.

## Supported Image Formats

- PNG
- JPEG/JPG

## Images Sync Layout

The `gplay images plan`, `gplay images pull`, and `gplay images sync`
commands use the same locale layout under each locale directory:

```
metadata/
  en-US/
    images/
      phoneScreenshots/
        1.png
      featureGraphic.png
      icon.png
      promoGraphic.png
      tvBanner.png
```

Screenshots stay grouped by device type. Single assets stay as one file per
locale so deterministic sync can compare by SHA-256 hash.

## Migrating from Fastlane

If you have existing Fastlane metadata, use the migration command:

```bash
gplay migrate fastlane --source ./fastlane/metadata/android --dest ./metadata
```
