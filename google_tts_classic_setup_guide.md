# Google Classic TTS Setup Guide for StoryNest

## Overview

Google Classic TTS provides high-quality, natural-sounding text-to-speech with voices specifically optimized for children's content. This guide walks you through setting up Chirp TTS in StoryNest.

## Prerequisites

### 1. Google Cloud Account Setup

1. **Create a Google Cloud Project:**
   ```bash
   # Install Google Cloud CLI if not already installed
   curl https://sdk.cloud.google.com | bash
   exec -l $SHELL
   gcloud init
   
   # Create a new project
   gcloud projects create storynest-tts --name="StoryNest TTS"
   gcloud config set project storynest-tts
   ```

2. **Enable the Text-to-Speech API:**
   ```bash
   gcloud services enable texttospeech.googleapis.com
   ```

3. **Create a Service Account:**
   ```bash
   gcloud iam service-accounts create storynest-tts \
     --description="Service account for StoryNest TTS" \
     --display-name="StoryNest TTS"
   
   # Download the key file
   gcloud iam service-accounts keys create ~/.config/storynest/google-credentials.json \
     --iam-account=storynest-tts@storynest-tts.iam.gserviceaccount.com
   ```

4. **Grant necessary permissions:**
   ```bash
   gcloud projects add-iam-policy-binding storynest-tts \
     --member="serviceAccount:storynest-tts@storynest-tts.iam.gserviceaccount.com" \
     --role="roles/cloudtts.user"
   ```

### 2. Environment Setup

Set the environment variable to point to your credentials:

```bash
export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.config/storynest/google-credentials.json"
```

Add this to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.) to make it permanent:

```bash
echo 'export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.config/storynest/google-credentials.json"' >> ~/.bashrc
```

### 3. Go Dependencies

Add these dependencies to your `go.mod`:

```bash
go get cloud.google.com/go/texttospeech/apiv1
go get cloud.google.com/go/texttospeech/apiv1/texttospeechpb
go get google.golang.org/api/option
```

### 4. Audio Player Requirements

StoryNest requires an audio player to play the generated MP3 files:

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install mpg123 ffmpeg
```

**macOS:**
```bash
# afplay is built-in, but you can also install mpg123
brew install mpg123 ffmpeg
```

**Windows:**
```powershell
# Install chocolatey if not already installed
# Then install ffmpeg
choco install ffmpeg
```

## Configuration

### 1. Update your StoryNest configuration

Create or update `~/.storynest/config.yaml`:

```yaml
tts:
  type: "chirp"
  voice: "en-US-Journey-F"  # Great for children's stories
  speed: 1.0
  volume: 0.8
  cache_enabled: true
  cache_max_size_mb: 500
```

### 2. Test the setup

```bash
# Test TTS engine
storynest tts test "Hello from Google Chirp TTS!"

# Check TTS status
storynest tts status

# Configure voices and settings
storynest tts configure
```

## Recommended Voices for Children

### Journey Voices (Best for Children)
- `en-US-Journey-F` - Female, warm and engaging
- `en-US-Journey-D` - Male, friendly and clear  
- `en-US-Journey-O` - Male, storyteller-like

### Neural Voices (High Quality)
- `en-US-Neural2-C` - Female, natural sounding
- `en-US-Neural2-F` - Female, expressive
- `en-US-Neural2-H` - Female, gentle

### UK English Voices (For Variety)
- `en-GB-Standard-A` - Female, British accent
- `en-GB-Standard-B` - Male, British accent

## Features

### 🎯 **Local Caching**
- Generated audio files are cached locally to avoid repeated API calls
- Cache location: `~/.cache/storynest/tts_audio/` (or platform equivalent)
- Automatic cache management with configurable size limits

### 🎛️ **Voice Customization**
- 20+ high-quality voices available
- Speed control (0.25x to 4.0x)
- Volume control with decibel precision
- Optimized settings for children's content

### 📱 **Smart Text Processing**
- Automatic text chunking for long stories
- Sentence boundary detection
- Special character handling
- Audio file concatenation for seamless playback

### 💰 **Cost Optimization**
- Local caching prevents duplicate API calls
- Efficient chunking reduces API usage
- Only generates new audio when text/settings change

## Usage Examples

### Basic Usage
```bash
# Use Chirp TTS for a random story
storynest random

# Configure voice settings
storynest tts configure

# Test with custom text
storynest tts test "Once upon a time, in a magical forest..."
```

### Advanced Configuration
```bash
# Check available voices
storynest tts status

# Clear cache to free space
storynest tts clear-cache

# Read a specific story with Chirp
storynest read goldilocks
```

## Troubleshooting

### Common Issues

1. **Authentication Error:**
   ```
   Error: failed to create Google TTS client
   ```
   **Solution:** Check that `GOOGLE_APPLICATION_CREDENTIALS` is set and the file exists.

2. **API Not Enabled:**
   ```
   Error: Cloud Text-to-Speech API has not been used
   ```
   **Solution:** Enable the API: `gcloud services enable texttospeech.googleapis.com`

3. **Permission Denied:**
   ```
   Error: The caller does not have permission
   ```
   **Solution:** Add the `roles/cloudtts.user` role to your service account.

4. **Audio Player Not Found:**
   ```
   Error: no audio player found
   ```
   **Solution:** Install `mpg123`, `ffplay`, or ensure system audio players are available.

### Debug Commands

```bash
# Check Google Cloud configuration
gcloud auth list
gcloud config list

# Verify credentials file
ls -la ~/.config/storynest/google-credentials.json

# Test API access directly
gcloud auth activate-service-account --key-file=~/.config/storynest/google-credentials.json
```

## Cost Considerations

Google Cloud Text-to-Speech pricing (as of 2024):
- Standard voices: $4.00 per 1 million characters
- Neural voices: $16.00 per 1 million characters  
- Journey voices: $16.00 per 1 million characters

**Cost-saving tips:**
- Use local caching (enabled by default)
- Choose shorter stories for testing
- Consider using standard voices for development

## Security Best Practices

1. **Credential Protection:**
   - Never commit credential files to version control
   - Use environment variables in production
   - Rotate service account keys regularly

2. **Permissions:**
   - Use principle of least privilege
   - Only grant `cloudtts.user` role, not broader permissions

3. **Cache Management:**
   - Regularly clean cache directory
   - Set appropriate cache size limits
   - Monitor disk usage

## Next Steps

Once Chirp TTS is configured:

1. **Explore Voices:** Try different voices to find the best fit for your children
2. **Optimize Settings:** Adjust speed and volume for comfortable listening  
3. **Manage Cache:** Set up automatic cache cleanup if needed
4. **Monitor Usage:** Keep track of API usage in Google Cloud Console

Enjoy high-quality, natural storytelling with Google Chirp TTS! 🌟