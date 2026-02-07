# Autogit

A robust, cross-platform CLI tool that automates Git version control using AI-generated commit messages.

## Features

- 🤖 **AI-Powered Commit Messages**: Supports multiple AI providers (Gemini, OpenAI, Anthropic, OpenRouter)
- 🔄 **Background Daemon**: Monitors your repository and automatically commits and pushes changes
- 🎨 **Interactive TUI**: Beautiful terminal UI with dashboard, logs, and settings
- 🔔 **Desktop Notifications**: Get notified of commits and errors
- 🛡️ **Error Handling**: Gracefully handles merge conflicts and network errors
- 🌍 **Cross-Platform**: Works on Linux, Windows, and macOS

## Installation

### Build from Source

```bash
git clone <repository-url>
cd git-autobot
go build -o autogit ./cmd/autogit
```

### Make Executable Available System-Wide

After building, add the `autogit` executable to your PATH so you can run it from any directory.

#### **Ubuntu/Linux:**

1. Create a local bin directory (if it doesn't exist):
   ```bash
   mkdir -p ~/bin
   ```

2. Move or copy the `autogit` binary to `~/bin`:
   ```bash
   cp autogit ~/bin/autogit
   # or if building in a different location:
   # cp /path/to/autogit ~/bin/autogit
   ```

3. Add to PATH by editing `~/.bashrc` or `~/.zshrc`:
   ```bash
   echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
   source ~/.bashrc
   ```
   
   For Zsh:
   ```bash
   echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
   source ~/.zshrc
   ```

4. Verify installation:
   ```bash
   autogit --version
   ```

#### **macOS:**

1. Create a local bin directory (if it doesn't exist):
   ```bash
   mkdir -p ~/bin
   ```

2. Move or copy the `autogit` binary to `~/bin`:
   ```bash
   cp autogit ~/bin/autogit
   ```

3. Add to PATH by editing `~/.zshrc` (or `~/.bash_profile` for older macOS):
   ```bash
   echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
   source ~/.zshrc
   ```

4. Verify installation:
   ```bash
   autogit --version
   ```

#### **Windows:**

1. Create a local bin directory (e.g., `C:\Users\YourUsername\bin`):
   ```cmd
   mkdir %USERPROFILE%\bin
   ```

2. Move or copy the `autogit.exe` binary to that directory:
   ```cmd
   copy autogit.exe %USERPROFILE%\bin\autogit.exe
   ```

3. Add to PATH:
   - Open System Properties → Advanced → Environment Variables
   - Under "User variables", select "Path" and click "Edit"
   - Click "New" and add: `%USERPROFILE%\bin`
   - Click "OK" on all dialogs
   - **Restart your terminal/PowerShell**

   Or using PowerShell (run as Administrator):
   ```powershell
   [Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\bin", "User")
   ```

4. Verify installation (in a new terminal):
   ```cmd
   autogit --version
   ```

### Alternative: Install via Go

If you have Go installed and the repository is available:
```bash
go install github.com/aadityansha/autogit/cmd/autogit@latest
```

This will install to `$GOPATH/bin` or `$HOME/go/bin`, which should already be in your PATH if Go is properly configured.

## Quick Start

1. **Initialize autogit in your repository:**
   ```bash
   cd /path/to/your/git/repo
   autogit init
   ```

2. **Configure your AI provider:**
   ```bash
   autogit --menu
   ```
   Navigate to Settings tab and configure:
   - AI Provider (Gemini, OpenAI, OpenRouter, Anthropic)
   - API Key
   - Base URL (for OpenRouter or custom endpoints)
   - Check Interval (in minutes)

3. **View status:**
   ```bash
   autogit status
   ```

4. **Open interactive dashboard:**
   ```bash
   autogit --menu
   # or
   autogit menu
   ```

5. **Pause the daemon:**
   ```bash
   autogit pause
   ```

## Configuration

Configuration is stored in `~/.config/autogit/config.json` (or `%APPDATA%\autogit\config.json` on Windows).

You can also set environment variables:
- `AUTOGIT_AI_PROVIDER`: AI provider name
- `AUTOGIT_API_KEY`: API key
- `AUTOGIT_BASE_URL`: Base URL for API
- `AUTOGIT_CHECK_INTERVAL_MINUTES`: Check interval in minutes

## AI Providers

### Google Gemini
- Provider: `gemini`
- Requires: API key from Google AI Studio

### OpenAI / Compatible
- Provider: `openai` or `openrouter`
- Requires: API key
- Base URL: Optional (defaults to OpenAI or OpenRouter endpoints)

### Anthropic (Claude)
- Provider: `anthropic` or `claude`
- Requires: API key from Anthropic

## Architecture

```
/cmd/autogit/main.go        # CLI entry point
/internal/
  ├── config/               # Configuration management
  ├── daemon/               # Background daemon logic
  ├── git/                  # Git command wrappers
  ├── ai/                   # AI provider adapters
  ├── tui/                  # Bubble Tea TUI
  └── notify/                # Desktop notifications
```

## How It Works

1. **Git Root Detection**: Automatically detects the Git root directory using `git rev-parse --show-toplevel`
2. **Background Monitoring**: Daemon runs in the background, checking for changes at configured intervals
3. **Change Detection**: Uses `git status --porcelain` to detect uncommitted changes
4. **AI Generation**: Sends code diff to AI provider to generate Conventional Commit messages
5. **Auto Commit & Push**: Stages, commits, and pushes changes automatically
6. **Error Handling**: On push failure (merge conflict, network error), daemon pauses and notifies user

## Commands

- `autogit --version` / `autogit -v` - Show version information
- `autogit init` - Initialize daemon for current repository
- `autogit --menu` / `autogit menu` - Open interactive TUI
- `autogit pause` - Stop the daemon
- `autogit status` - Show daemon status

## License

MIT

