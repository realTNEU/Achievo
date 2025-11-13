# LLM-Based Game Discovery Setup

Achievo uses **LLaMA 3.1 8B** to intelligently analyze game directories, classify them as games vs software utilities, and infer save/log paths for automatic rule generation.

## Overview

The LLM classifier analyzes directory structures, file names, and metadata to:
- Determine if a directory contains a **real PC game** (not software/utility)
- Identify the **game root folder**
- Find the **main executable**
- Estimate the **actual game name** (not just folder name)
- Infer **save file paths** (saves/, SaveGames/, AppData/, Documents/)
- Infer **log file paths** (logs/, Logs/, game.log)
- Infer **config file paths** (config/, Config/, settings/)

This provides much better accuracy than rule-based heuristics alone and enables automatic rule generation with accurate paths.

## Requirements

- **Ollama** (recommended) or **LM Studio** (OpenAI-compatible API)
- **LLaMA 3.1 8B** model (~4.7GB download)

## Setup Instructions

### 1. Install Ollama

**Windows:**
```powershell
# Download from https://ollama.com/download
# Or use winget:
winget install Ollama.Ollama
```

**Linux:**
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

**macOS:**
```bash
brew install ollama
```

### 2. Download LLaMA 3.1 8B Model

```bash
# Recommended: LLaMA 3.1 8B (4.7GB, comprehensive analysis)
ollama pull llama3.1:8b

# Alternative smaller models (if 8B is too large):
# ollama pull phi3:mini      # 3.8GB, fast
# ollama pull qwen2.5:0.5b   # 1GB, good balance
```

**Note**: LLaMA 3.1 8B is recommended for best accuracy in game analysis and path inference.

### 3. Configure Achievo

Edit your `config.yaml`:

**For Local Ollama:**
```yaml
detection:
  discovery_enabled: true
  discovery_interval: 24  # hours between scans (0 = only on startup)
  custom_scan_paths:      # optional: custom directories to scan
    - "F:/games"
  llm_enabled: true       # Enable LLM-powered discovery
  llm_api_url: "http://localhost:11434"  # Local Ollama default
  llm_model_name: "llama3.1:8b"         # LLaMA 3.1 8B
```

**For Remote Ollama Server:**
```yaml
detection:
  discovery_enabled: true
  discovery_interval: 24
  custom_scan_paths:
    - "F:/games"
  llm_enabled: true
  llm_api_url: "https://ollama.tneuserver.online"  # Remote Ollama server
  llm_model_name: "llama3.1:8b"
```

**Note**: Remote servers must be accessible via HTTPS and support the Ollama API endpoints.

### 4. Start Ollama (if not running as service)

```bash
ollama serve
```

On Windows, Ollama typically runs as a service automatically after installation.

### 5. Verify Setup

**For Local Ollama:**
```bash
# List installed models
ollama list

# Should show llama3.1:8b

# Test API
curl http://localhost:11434/api/tags
```

**For Remote Ollama Server:**
```bash
# Test remote API
curl -i https://ollama.tneuserver.online/api/tags

# Should return HTTP 200 with list of available models
```

### 6. Run Achievo

Start Achievo and check the logs:

```
[INFO] LLM-based game analysis enabled (API: http://localhost:11434, Model: llama3.1:8b)
[INFO] LLM will analyze game structure, infer paths, and generate rule templates
[INFO] Starting game discovery scan...
[LLM] Classified as GAME: Cyberpunk2077 (confidence: 0.98, reason: "Contains game executable, save directory, and game data files")
[INFO] Using LLM-inferred paths for Cyberpunk2077: saves=[saves/, AppData/Local/CD Projekt Red], logs=[logs/game.log]
[INFO] Generated auto rule file for Cyberpunk2077 at configs\games\auto\game-cyberpunk2077.yaml
```

## How It Works

### 1. Directory Analysis

When scanning, Achievo gathers comprehensive information about each directory:
- Executable files (.exe)
- Subdirectories
- File names and types
- Game metadata files (steam_appid.txt, game.ini, etc.)

### 2. LLM Analysis

The LLM (LLaMA 3.1 8B) analyzes this information and returns structured JSON:
```json
{
  "is_game": true,
  "confidence": 0.95,
  "reason": "Contains game executable, save directory, and game data files",
  "game_name": "Cyberpunk 2077",
  "game_root": ".",
  "main_executable": "Cyberpunk2077.exe",
  "save_paths": ["saves/", "%APPDATA%/Local/CD Projekt Red/Cyberpunk 2077"],
  "log_paths": ["logs/game.log", "r6/logs/"],
  "config_paths": ["config/", "r6/config/"]
}
```

### 3. Rule Generation

Achievo uses the LLM-inferred paths to generate rule files:
- Save paths → File watch rules
- Log paths → Log pattern rules
- Config paths → Config file watch rules

### 4. Decision Making

- **If confidence ≥ 0.6**: LLM result is used
- **If confidence < 0.6**: Falls back to heuristics
- **If LLM unavailable**: Uses heuristics only (no functionality lost)

## Performance

- **Model Size**: LLaMA 3.1 8B (~4.7GB)
- **Speed**: ~1-3 seconds per directory analysis
- **Resource Usage**: 
  - CPU: Moderate (runs on CPU if no GPU)
  - GPU: Recommended (much faster with CUDA/ROCm)
  - RAM: ~8-12GB recommended for smooth operation
- **Caching**: Ollama caches responses for faster subsequent calls

## Fallback Behavior

If the LLM is unavailable or disabled:
- Achievo automatically falls back to rule-based heuristics
- No functionality is lost
- Logs will indicate: `[INFO] LLM-based classification disabled, using heuristics only`

## Troubleshooting

### LLM Not Available

If you see errors like "LLM call failed":
1. **Check Ollama is running**: `ollama list`
2. **Verify model is downloaded**: `ollama list` should show `llama3.1:8b`
3. **Test API**: `curl http://localhost:11434/api/tags`
4. **Check firewall/network settings**: Ensure port 11434 is accessible
5. **Check Ollama logs**: Look for error messages in Ollama output

### Slow Classification

- **Use GPU**: Install CUDA/ROCm drivers for GPU acceleration
- **Reduce scan frequency**: Increase `discovery_interval` in config
- **Use smaller model**: Try `phi3:mini` or `qwen2.5:0.5b` (less accurate but faster)
- **Disable LLM**: Set `llm_enabled: false` to use heuristics only

### High Resource Usage

- **Use GPU**: GPU acceleration significantly reduces CPU usage
- **Reduce model size**: Use smaller models if 8B is too resource-intensive
- **Adjust scan interval**: Increase `discovery_interval` to scan less frequently
- **Disable for initial scan**: Can enable LLM only for periodic scans

### Model Not Found

If you see "model not found" errors:
```bash
# Pull the model explicitly
ollama pull llama3.1:8b

# Verify it's installed
ollama list
```

## Advanced Configuration

### Using LM Studio

If you prefer LM Studio over Ollama:

1. Install [LM Studio](https://lmstudio.ai/)
2. Download LLaMA 3.1 8B model in LM Studio
3. Start local server in LM Studio
4. Configure Achievo:

```yaml
detection:
  llm_enabled: true
  llm_api_url: "http://localhost:1234"  # LM Studio default port
  llm_model_name: "llama-3.1-8b"        # Model name in LM Studio
```

**Note**: LM Studio uses OpenAI-compatible API, so it should work with Achievo's Ollama-compatible client.

### Custom API Endpoint

If using a different LLM service (must support Ollama-compatible `/api/generate` endpoint):

```yaml
detection:
  llm_api_url: "https://api.example.com/v1"
  llm_model_name: "llama3.1:8b"
```

## Model Recommendations

| Model | Size | Speed | Accuracy | Use Case |
|-------|------|-------|----------|----------|
| `llama3.1:8b` | 4.7GB | Medium | **Highest** | **Recommended** - Best game analysis |
| `phi3:mini` | 3.8GB | Fast | High | Good balance, faster than 8B |
| `qwen2.5:0.5b` | 1GB | Very Fast | Medium | Low-resource systems |
| `tinyllama` | 637MB | Very Fast | Medium | Minimal resource usage |

**Recommendation**: Use `llama3.1:8b` for best accuracy in game classification and path inference.

## Example Output

```
[INFO] Starting game discovery scan...
[INFO] LLM-based game analysis enabled (API: http://localhost:11434, Model: llama3.1:8b)
[LLM] Classified as GAME: Cyberpunk2077 (confidence: 0.98, reason: "Contains game executable, save directory, and game data files")
[INFO] Using LLM-inferred paths for Cyberpunk2077: saves=[saves/, AppData/Local/CD Projekt Red], logs=[logs/game.log]
[LLM] Classified as SOFTWARE: VMware Workstation (confidence: 0.95, reason: "Virtualization software, not a game")
[LLM] Classified as GAME: The Witcher 3 (confidence: 0.92, reason: "Game executable and Steam App ID detected")
[INFO] Discovery summary: 2 new games saved, 2 auto-generated rule files created
```

## Next Steps

After setting up LLM-powered discovery:

1. **Run discovery scan**: Start Achievo to see LLM classifications
2. **Review generated rules**: Check `configs/games/auto/` for rule files with LLM-inferred paths
3. **Verify paths**: Confirm save/log paths are correct (LLM is usually accurate)
4. **Customize rules**: Edit rule files to add specific achievement detection logic
5. **Move on to achievement detection**: Once games are discovered and rules are set up, Achievo will track achievements

## Benefits of LLM-Powered Discovery

- ✅ **Accurate Classification**: Distinguishes games from software utilities
- ✅ **Path Inference**: Automatically finds save/log/config paths
- ✅ **Game Name Detection**: Identifies actual game names (not folder names)
- ✅ **Reduced False Positives**: Filters out drivers, utilities, development tools
- ✅ **Better Rule Generation**: Creates rule files with accurate paths
- ✅ **Intelligent Analysis**: Understands game directory structures

## Without LLM

If you don't set up LLM:
- Achievo still works with heuristic-based detection
- Less accurate classification (may detect some utilities as games)
- Path inference is less reliable (uses common patterns)
- Still functional, but LLM provides significant improvements
