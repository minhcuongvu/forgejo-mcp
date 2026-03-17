# Forgejo MCP Server

Forgejo/Gitea MCP server for self-hosted git (repos, issues, PRs).

Custom Go MCP server coming soon...

Forgejo MCP server setup for self-hosted git (OpenCode, Claude Code, etc).

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) installed and running
- [GitHub Personal Access Token](https://github.com/settings/tokens) with appropriate scopes
- [OpenCode](https://opencode.ai) installed (for Option 1)

## Setup

### Option 1: Local MCP via OpenCode (Recommended)

Spawns Docker container per-request via stdio. No persistent server needed.

1. Copy the example config:
   ```bash
   # Create config directory if needed
   mkdir -p ~/.config/opencode
   
   # Copy and edit the example
   cp opencode.json.example ~/.config/opencode/opencode.json
   ```

2. Edit `~/.config/opencode/opencode.json` and replace `your_token_here` with your GitHub token

3. Verify:
   ```bash
   opencode mcp list
   ```

### Option 2: HTTP Server (Docker Compose)

Runs persistent HTTP server. Useful for sharing across multiple clients or web apps.

```bash
cp .env.example .env  # Add your GitHub token
docker-compose up -d  # Server at http://localhost:8766/mcp
```

Requires Bearer token header on each request.

## Toolsets

`repos`, `issues`, `pull_requests`, `actions`, `code_security`, `discussions`, `gists`, `notifications`, `orgs`, `projects`, `users`

Flags: `--read-only`, `--toolsets`

## Verified Commits (SSH Signing)

```bash
# 1. Generate key
ssh-keygen -t ed25519 -C "your_email@example.com" -f ~/.ssh/id_ed25519_signing -N ""

# 2. Configure git
git config --global user.name "Your Name"
git config --global user.email "your_email@example.com"
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519_signing
git config --global commit.gpgsign true

# 3. Add to GitHub
gh auth refresh -h github.com -s admin:ssh_signing_key
gh ssh-key add ~/.ssh/id_ed25519_signing.pub --type signing --title "Signing key"
```

Or manually: https://github.com/settings/keys > "New SSH Key" > Key type: "Signing Key"

## Rules for AI Assistants

**CRITICAL**: 
1. **Know your identity** - Before committing, verify which model you are
2. **Never change git identity** - Use human's global config + co-author trailer
3. **Take responsibility** - Use your actual model name in co-author

```bash
# WRONG - breaks verification
git config --local user.name "Grok"
git config --local user.email "grok@x.ai"
git commit -m "..."  # Unverified!

# WRONG - taking credit for another AI's work
Co-authored-by: Grok <grok@x.ai>  # But you're Claude!

# CORRECT - use YOUR actual model
git commit -S -m "Message

Co-authored-by: Claude (claude-opus-4-5) <noreply@anthropic.com>"
# or
Co-authored-by: opencode (grok-4-1-fast) <grok@x.ai>
```

**Why**: SSH key belongs to human. GitHub verifies signature matches committer email. If committer = AI, signature won't match = unverified. `Co-authored-by` gives AI credit without breaking verification.

**History cleanup** (remove bad commits):
```bash
git reset --hard <good-commit>
git push --force
```

## Usage

Prompts: "List issues using github", "Review PR #123 with github mcp"

AGENTS.md: `For PR reviews, use github tools.`

## AI Skills

This repo includes a skill file for AI assistants. Install to your environment:

```bash
# For OpenCode
cp -r skills/git-ai-commits ~/.config/opencode/skills/

# For Claude Code
cp -r skills/git-ai-commits ~/.claude/skills/
```

The skill teaches AI models proper git commit attribution rules.

## References

- [GitHub MCP Server](https://github.com/github/github-mcp-server)
- [OpenCode MCP Docs](https://opencode.ai/docs/mcp-servers/)
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [GitHub: Signing Commits](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits)
