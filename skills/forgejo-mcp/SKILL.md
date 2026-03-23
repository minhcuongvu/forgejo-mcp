---
name: forgejo-mcp
description: Guide for AI assistants on using forgejo-mcp server to interact with a self-hosted Forgejo instance
license: MIT
compatibility: opencode, claude-code
metadata:
  category: git
  audience: ai-assistants
---

## What I do

Enable AI assistants to interact with a self-hosted Forgejo instance via the forgejo-mcp server, providing access to repositories, branches, commits, issues, pull requests, and file operations.

## Environment Setup

The forgejo-mcp server requires these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `FORGEJO_URL` | Base URL of your Forgejo instance | `http://localhost:3000` |
| `FORGEJO_TOKEN` | API token for authentication | (none - public access only) |

### Getting a Forgejo API Token

1. Go to your Forgejo instance Settings > Applications
2. Generate a token with these scopes:
   - `read:repository` - list repos and read files
   - `read:issue` - list issues
   - `write:issue` - create issues
   - `write:repository` - create PRs and merge

## Available Tools

### Repository Tools (Read)

| Tool | Purpose |
|------|---------|
| `list_repos` | List/search repositories |
| `get_repo` | Get repository details |
| `get_file` | Read file contents + SHA |

### Branch Tools (Read)

| Tool | Purpose |
|------|---------|
| `list_branches` | List branches in a repo |
| `get_branch` | Get branch details |

### Commit Tools (Read)

| Tool | Purpose |
|------|---------|
| `list_commits` | List commit history |
| `get_commit` | Get specific commit details |

### Issue Tools (Read/Write)

| Tool | Purpose |
|------|---------|
| `list_issues` | List issues (open/closed/all) |
| `get_issue` | Get issue details |
| `create_issue` | Create a new issue |

### Pull Request Tools (Read/Write)

| Tool | Purpose |
|------|---------|
| `list_pull_requests` | List PRs |
| `get_pull_request` | Get PR details |
| `get_pull_request_diff` | Get PR diff |
| `create_pull_request` | Create a new PR |
| `merge_pull_request` | Merge a PR |

### File Tools (Write)

| Tool | Purpose |
|------|---------|
| `create_branch` | Create a new branch |
| `create_or_update_file` | Commit a file (create/update) |
| `delete_file` | Delete a file |

## Common Workflows

### Read a File

```
Tool: get_file
  owner: "michael"
  repo: "backend1"
  path: "README.md"
  ref: "master"
```

### List Open Issues

```
Tool: list_issues
  owner: "michael"
  repo: "backend1"
  state: "open"
```

### Create an Issue

```
Tool: create_issue
  owner: "michael"
  repo: "backend1"
  title: "Bug: Login fails with 500 error"
  body: "When attempting to log in with valid credentials, the server returns HTTP 500."
```

### Create a PR Workflow

**1. Create branch:**
```
Tool: create_branch
  owner: "michael"
  repo: "backend1"
  new_branch_name: "feat/add-health-check"
  old_branch_name: "master"
```

**2. Get file SHA (for updates):**
```
Tool: get_file
  owner: "michael"
  repo: "backend1"
  path: "internal/handler/health.go"
  ref: "master"
```

**3. Update file:**
```
Tool: create_or_update_file
  owner: "michael"
  repo: "backend1"
  path: "internal/handler/health.go"
  content: "<full content>"
  message: "Add detailed health check endpoint

Co-authored-by: <Model> (<model-id>) <email>"
  branch: "feat/add-health-check"
  sha: "<sha from step 2>"
```

**4. Create PR:**
```
Tool: create_pull_request
  owner: "michael"
  repo: "backend1"
  title: "Add detailed health check endpoint"
  body: "## Summary\n- Extends /health endpoint\n- Adds component status\n\nCo-authored-by: <Model> (<model-id>) <email>"
  head: "feat/add-health-check"
  base: "master"
```

**5. Review diff:**
```
Tool: get_pull_request_diff
  owner: "michael"
  repo: "backend1"
  index: <pr_number>
```

**6. Merge (ask human first):**
```
Tool: merge_pull_request
  owner: "michael"
  repo: "backend1"
  index: <pr_number>
  merge_style: "squash"
  delete_branch: true
```

## AI Attribution Rules

Always include `Co-authored-by` in commit and PR messages:

```
Co-authored-by: <Name> (<model-id>) <email>
```

| Model | Attribution Line |
|-------|------------------|
| Claude Opus 4.5 | `Co-authored-by: Claude (claude-opus-4-5) <noreply@anthropic.com>` |
| Claude Opus 4.6 | `Co-authored-by: Claude Opus 4.6 <noreply@anthropic.com>` |
| Claude Sonnet 4 | `Co-authored-by: Claude (claude-sonnet-4) <noreply@anthropic.com>` |
| Grok | `Co-authored-by: opencode (grok-4-1-fast) <grok@x.ai>` |
| GPT-4 | `Co-authored-by: GPT (gpt-4) <noreply@openai.com>` |
| Kimi K2.5 | `Co-authored-by: Kimi (kimi-k2.5) <noreply@moonshot.cn>` |

### Rules

1. **Include Co-authored-by in every commit**
2. **Use your real model identity** — never impersonate
3. **Never set git user.name/email locally** — the API commits as the token owner
4. **Include attribution in PR body** — visible in PR description

## Branch Naming Conventions

| Prefix | Use Case |
|--------|----------|
| `feat/` | New features |
| `fix/` | Bug fixes |
| `docs/` | Documentation |
| `refactor/` | Code restructuring |
| `chore/` | Maintenance/config |

Examples: `feat/add-user-auth`, `fix/null-pointer-handler`

## Error Handling

| Problem | Solution |
|---------|----------|
| SHA mismatch (409) | Re-read file with `get_file` and retry |
| Branch exists | Use `get_branch` to check, commit to existing branch |
| PR merge conflict | Create new commit resolving the conflict |
| Auth error | Verify `FORGEJO_TOKEN` has required scopes |

## References

- [Forgejo MCP Server Repository](https://github.com/your-org/forgejo-mcp)
- [Forgejo API Docs](https://forgejo.org/docs/latest/dev/api-usage/)
- [Gitea API Docs](https://docs.gitea.com/api/1.22/)
