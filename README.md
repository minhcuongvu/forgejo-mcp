# Forgejo MCP Server

MCP (Model Context Protocol) server for Forgejo/Gitea self-hosted git forges. Enables AI assistants (OpenCode, Claude Code, etc.) to interact with your Forgejo instance.

## Features

- **Repository Management**: List, search, and get detailed information about repositories
- **Branch Operations**: List branches and get branch details
- **Commit History**: List commits and get detailed commit information
- **Issue Tracking**: List, view, and create issues
- **Pull Request Management**: List, view, create, and merge pull requests
- **File Access**: Retrieve file contents from repositories

## Prerequisites

- [Go 1.23+](https://go.dev/dl/) (for building)
- [Docker](https://docs.docker.com/get-docker/) (optional, for containerized deployment)
- Forgejo/Gitea instance with API access
- Forgejo API token (for private repos and write operations)

## Quick Start

### Option 1: Build and Run Directly

```bash
# Build
go build -o forgejo-mcp-server

# Run (with env vars)
FORGEJO_URL=http://your-forgejo:3000 FORGEJO_TOKEN=your_token ./forgejo-mcp-server
```

### Option 2: Docker

```bash
# Build image
docker build -t forgejo-mcp-server .

# Run
docker run -i --rm \
  -e FORGEJO_URL=http://your-forgejo:3000 \
  -e FORGEJO_TOKEN=your_token \
  forgejo-mcp-server
```

### Option 3: OpenCode Integration

1. Copy example config:
   ```bash
   mkdir -p ~/.config/opencode
   cp opencode.json.example ~/.config/opencode/opencode.json
   ```

2. Edit `~/.config/opencode/opencode.json`:
   - Set `FORGEJO_URL` to your Forgejo instance
   - Set `FORGEJO_TOKEN` to your API token (optional for public repos)

3. Verify:
   ```bash
   opencode mcp list
   ```

## Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `FORGEJO_URL` | Base URL of your Forgejo instance | `http://localhost:3000` |
| `FORGEJO_TOKEN` | API token for authentication | (none - public access only) |

### Getting a Forgejo API Token

1. Go to your Forgejo instance
2. Navigate to Settings > Applications
3. Generate new token with appropriate scopes:
   - `read:repository` - for listing repos and files
   - `read:issue` - for listing issues
   - `write:issue` - for creating issues
   - `write:repository` - for creating PRs and merging

## Available Tools

### Repository Tools

#### `list_repos`
List repositories from Forgejo.

**Parameters:**
- `query` (optional): Search query to filter repositories
- `owner` (optional): Filter by repository owner username
- `limit` (optional): Maximum number of results (default: 50)

#### `get_repo`
Get details of a specific repository.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name

#### `get_file`
Get contents of a file from a repository.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `path` (required): Path to the file
- `ref` (optional): Branch, tag, or commit (default: default branch)

### Branch Tools

#### `list_branches`
List branches in a repository.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name

#### `get_branch`
Get details of a specific branch.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `branch` (required): Branch name

### Commit Tools

#### `list_commits`
List commits in a repository.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `sha` (optional): Branch name or commit SHA (default: default branch)
- `limit` (optional): Maximum number of commits (default: 30)

#### `get_commit`
Get details of a specific commit.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `sha` (required): Commit SHA

### Issue Tools

#### `list_issues`
List issues in a repository.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `state` (optional): Filter by state: `open`, `closed`, `all` (default: `open`)

#### `get_issue`
Get details of a specific issue.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `index` (required): Issue number

#### `create_issue`
Create a new issue in a repository.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `title` (required): Issue title
- `body` (optional): Issue body/description

### Pull Request Tools

#### `list_pull_requests`
List pull requests in a repository.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `state` (optional): Filter by state: `open`, `closed`, `all` (default: `open`)

#### `get_pull_request`
Get details of a specific pull request.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `index` (required): Pull request number

#### `get_pull_request_diff`
Get the diff of a pull request.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `index` (required): Pull request number

#### `create_pull_request`
Create a new pull request.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `title` (required): Pull request title
- `head` (required): Source branch name
- `base` (required): Target branch name
- `body` (optional): Pull request description

#### `merge_pull_request`
Merge a pull request.

**Parameters:**
- `owner` (required): Repository owner username
- `repo` (required): Repository name
- `index` (required): Pull request number
- `merge_style` (optional): Merge style: `merge`, `rebase`, `squash` (default: `merge`)
- `title` (optional): Merge commit title
- `message` (optional): Merge commit message
- `delete_branch` (optional): Delete source branch after merge (default: false)

## Usage Examples

In your AI assistant:
- "List all repos on forgejo"
- "Show me the README from michael/myproject"
- "List open issues in michael/myproject"
- "Get the contents of main.go from michael/myproject"
- "Show me the last 10 commits in michael/myproject"
- "List all branches in michael/myproject"
- "Show me pull request #5 in michael/myproject"
- "Create a new issue in michael/myproject about the login bug"
- "Merge pull request #3 in michael/myproject"

## Development

```bash
# Run tests
go test ./...

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o forgejo-mcp-server-linux
GOOS=windows GOARCH=amd64 go build -o forgejo-mcp-server.exe
GOOS=darwin GOARCH=arm64 go build -o forgejo-mcp-server-darwin
```

## Protocol

This server implements the [Model Context Protocol](https://modelcontextprotocol.io/) specification, using JSON-RPC 2.0 over stdio.

## References

- [Forgejo API Documentation](https://forgejo.org/docs/latest/dev/api-usage/)
- [Gitea API Documentation](https://docs.gitea.com/api/1.22/)
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [MCP Go SDK](https://github.com/modelcontextprotocol/sdk-go)
