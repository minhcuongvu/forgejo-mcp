package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// MCP Protocol types
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Forgejo API types
type ForgejoRepo struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	Private       bool   `json:"private"`
	Fork          bool   `json:"fork"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
	Stars         int    `json:"stars_count"`
	Forks         int    `json:"forks_count"`
	Watchers      int    `json:"watchers_count"`
	Size          int    `json:"size"`
	DefaultBranch string `json:"default_branch"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type ForgejoSearchResult struct {
	OK   bool          `json:"ok"`
	Data []ForgejoRepo `json:"data"`
}

type ForgejoUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type ForgejoIssue struct {
	ID        int         `json:"id"`
	Number    int         `json:"number"`
	Title     string      `json:"title"`
	Body      string      `json:"body"`
	State     string      `json:"state"`
	HTMLURL   string      `json:"html_url"`
	User      ForgejoUser `json:"user"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
}

type ForgejoCommit struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
	Author    *ForgejoUser `json:"author"`
	Committer *ForgejoUser `json:"committer"`
}

type ForgejoBranch struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
	Protected bool `json:"protected"`
}

type ForgejoPullRequest struct {
	ID      int         `json:"id"`
	Number  int         `json:"number"`
	Title   string      `json:"title"`
	Body    string      `json:"body"`
	State   string      `json:"state"`
	HTMLURL string      `json:"html_url"`
	DiffURL string      `json:"diff_url"`
	User    ForgejoUser `json:"user"`
	Head    struct {
		Ref  string      `json:"ref"`
		SHA  string      `json:"sha"`
		Repo ForgejoRepo `json:"repo"`
	} `json:"head"`
	Base struct {
		Ref  string      `json:"ref"`
		SHA  string      `json:"sha"`
		Repo ForgejoRepo `json:"repo"`
	} `json:"base"`
	Merged    bool   `json:"merged"`
	Mergeable bool   `json:"mergeable"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	MergedAt  string `json:"merged_at"`
}

// Server state
var (
	forgejoURL   string
	forgejoToken string
)

func main() {
	forgejoURL = os.Getenv("FORGEJO_URL")
	if forgejoURL == "" {
		fmt.Fprintf(os.Stderr, "Warning: FORGEJO_URL not set, using localhost:3000\n")
		forgejoURL = "http://localhost:3000"
	}
	forgejoURL = strings.TrimSuffix(forgejoURL, "/")

	forgejoToken = os.Getenv("FORGEJO_TOKEN")

	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
			continue
		}

		resp := handleRequest(req)
		respBytes, _ := json.Marshal(resp)
		fmt.Println(string(respBytes))
	}
}

func handleRequest(req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: InitializeResult{
				ProtocolVersion: "2024-11-05",
				Capabilities: Capabilities{
					Tools: &ToolsCapability{},
				},
				ServerInfo: ServerInfo{
					Name:    "forgejo-mcp-server",
					Version: "2.0.0",
				},
			},
		}

	case "notifications/initialized":
		return JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: nil}

	case "tools/list":
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: ToolsListResult{
				Tools: getTools(),
			},
		}

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, -32602, "Invalid params")
		}
		result := callTool(params)
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}

	default:
		return errorResponse(req.ID, -32601, "Method not found: "+req.Method)
	}
}

func errorResponse(id interface{}, code int, message string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
}

func getTools() []Tool {
	return []Tool{
		// Repository tools
		{
			Name:        "list_repos",
			Description: "List repositories from Forgejo. Returns all accessible repos or search by query.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "Optional search query to filter repositories"},
					"owner": {Type: "string", Description: "Filter by repository owner username"},
					"limit": {Type: "integer", Description: "Maximum number of results (default 50)"},
				},
			},
		},
		{
			Name:        "get_repo",
			Description: "Get details of a specific repository",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
				},
				Required: []string{"owner", "repo"},
			},
		},
		{
			Name:        "get_file",
			Description: "Get contents of a file from a repository",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
					"path":  {Type: "string", Description: "Path to the file"},
					"ref":   {Type: "string", Description: "Branch, tag, or commit (default: default branch)"},
				},
				Required: []string{"owner", "repo", "path"},
			},
		},
		// Branch tools
		{
			Name:        "list_branches",
			Description: "List branches in a repository",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
				},
				Required: []string{"owner", "repo"},
			},
		},
		{
			Name:        "get_branch",
			Description: "Get details of a specific branch",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner":  {Type: "string", Description: "Repository owner username"},
					"repo":   {Type: "string", Description: "Repository name"},
					"branch": {Type: "string", Description: "Branch name"},
				},
				Required: []string{"owner", "repo", "branch"},
			},
		},
		// Commit tools
		{
			Name:        "list_commits",
			Description: "List commits in a repository",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
					"sha":   {Type: "string", Description: "Branch name or commit SHA (default: default branch)"},
					"limit": {Type: "integer", Description: "Maximum number of commits (default 30)"},
				},
				Required: []string{"owner", "repo"},
			},
		},
		{
			Name:        "get_commit",
			Description: "Get details of a specific commit",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
					"sha":   {Type: "string", Description: "Commit SHA"},
				},
				Required: []string{"owner", "repo", "sha"},
			},
		},
		// Issue tools
		{
			Name:        "list_issues",
			Description: "List issues in a repository",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
					"state": {Type: "string", Description: "Filter by state: open, closed, all (default: open)"},
				},
				Required: []string{"owner", "repo"},
			},
		},
		{
			Name:        "get_issue",
			Description: "Get details of a specific issue",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
					"index": {Type: "integer", Description: "Issue number"},
				},
				Required: []string{"owner", "repo", "index"},
			},
		},
		{
			Name:        "create_issue",
			Description: "Create a new issue in a repository",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
					"title": {Type: "string", Description: "Issue title"},
					"body":  {Type: "string", Description: "Issue body/description"},
				},
				Required: []string{"owner", "repo", "title"},
			},
		},
		// Pull Request tools
		{
			Name:        "list_pull_requests",
			Description: "List pull requests in a repository",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
					"state": {Type: "string", Description: "Filter by state: open, closed, all (default: open)"},
				},
				Required: []string{"owner", "repo"},
			},
		},
		{
			Name:        "get_pull_request",
			Description: "Get details of a specific pull request",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
					"index": {Type: "integer", Description: "Pull request number"},
				},
				Required: []string{"owner", "repo", "index"},
			},
		},
		{
			Name:        "get_pull_request_diff",
			Description: "Get the diff of a pull request",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
					"index": {Type: "integer", Description: "Pull request number"},
				},
				Required: []string{"owner", "repo", "index"},
			},
		},
		{
			Name:        "create_pull_request",
			Description: "Create a new pull request",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner": {Type: "string", Description: "Repository owner username"},
					"repo":  {Type: "string", Description: "Repository name"},
					"title": {Type: "string", Description: "Pull request title"},
					"body":  {Type: "string", Description: "Pull request description"},
					"head":  {Type: "string", Description: "Source branch name"},
					"base":  {Type: "string", Description: "Target branch name"},
				},
				Required: []string{"owner", "repo", "title", "head", "base"},
			},
		},
		{
			Name:        "merge_pull_request",
			Description: "Merge a pull request",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner":         {Type: "string", Description: "Repository owner username"},
					"repo":          {Type: "string", Description: "Repository name"},
					"index":         {Type: "integer", Description: "Pull request number"},
					"merge_style":   {Type: "string", Description: "Merge style: merge, rebase, squash (default: merge)"},
					"title":         {Type: "string", Description: "Merge commit title (optional)"},
					"message":       {Type: "string", Description: "Merge commit message (optional)"},
					"delete_branch": {Type: "boolean", Description: "Delete source branch after merge (default: false)"},
				},
				Required: []string{"owner", "repo", "index"},
			},
		},
		// Branch creation
		{
			Name:        "create_branch",
			Description: "Create a new branch in a repository",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner":           {Type: "string", Description: "Repository owner username"},
					"repo":            {Type: "string", Description: "Repository name"},
					"new_branch_name": {Type: "string", Description: "Name for the new branch"},
					"old_branch_name": {Type: "string", Description: "Source branch to create from (default: default branch)"},
				},
				Required: []string{"owner", "repo", "new_branch_name"},
			},
		},
		// File write tools
		{
			Name:        "create_or_update_file",
			Description: "Create or update a file in a repository. This commits the change directly to the specified branch. To update an existing file you must provide its current SHA (use get_file to obtain it).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner":   {Type: "string", Description: "Repository owner username"},
					"repo":    {Type: "string", Description: "Repository name"},
					"path":    {Type: "string", Description: "Path to the file (e.g. src/main.go)"},
					"content": {Type: "string", Description: "File content (plain text, will be base64-encoded automatically)"},
					"message": {Type: "string", Description: "Commit message"},
					"branch":  {Type: "string", Description: "Branch to commit to (default: default branch)"},
					"sha":     {Type: "string", Description: "SHA of the file being replaced (required for updates, omit for new files)"},
				},
				Required: []string{"owner", "repo", "path", "content", "message"},
			},
		},
		{
			Name:        "delete_file",
			Description: "Delete a file from a repository. This commits the deletion directly to the specified branch.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"owner":   {Type: "string", Description: "Repository owner username"},
					"repo":    {Type: "string", Description: "Repository name"},
					"path":    {Type: "string", Description: "Path to the file to delete"},
					"message": {Type: "string", Description: "Commit message"},
					"branch":  {Type: "string", Description: "Branch to commit to (default: default branch)"},
					"sha":     {Type: "string", Description: "SHA of the file being deleted (required, use get_file to obtain it)"},
				},
				Required: []string{"owner", "repo", "path", "message", "sha"},
			},
		},
	}
}

func callTool(params CallToolParams) CallToolResult {
	switch params.Name {
	// Repos
	case "list_repos":
		return listRepos(params.Arguments)
	case "get_repo":
		return getRepo(params.Arguments)
	case "get_file":
		return getFile(params.Arguments)
	// Branches
	case "list_branches":
		return listBranches(params.Arguments)
	case "get_branch":
		return getBranch(params.Arguments)
	// Commits
	case "list_commits":
		return listCommits(params.Arguments)
	case "get_commit":
		return getCommit(params.Arguments)
	// Issues
	case "list_issues":
		return listIssues(params.Arguments)
	case "get_issue":
		return getIssue(params.Arguments)
	case "create_issue":
		return createIssue(params.Arguments)
	// Pull Requests
	case "list_pull_requests":
		return listPullRequests(params.Arguments)
	case "get_pull_request":
		return getPullRequest(params.Arguments)
	case "get_pull_request_diff":
		return getPullRequestDiff(params.Arguments)
	case "create_pull_request":
		return createPullRequest(params.Arguments)
	case "merge_pull_request":
		return mergePullRequest(params.Arguments)
	// Branch creation
	case "create_branch":
		return createBranch(params.Arguments)
	// File write
	case "create_or_update_file":
		return createOrUpdateFile(params.Arguments)
	case "delete_file":
		return deleteFile(params.Arguments)
	default:
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Unknown tool: " + params.Name}},
			IsError: true,
		}
	}
}

// HTTP helpers
func forgejoRequest(method, endpoint string, body interface{}) ([]byte, error) {
	url := forgejoURL + "/api/v1" + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	if forgejoToken != "" {
		req.Header.Set("Authorization", "token "+forgejoToken)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func forgejoGet(endpoint string) ([]byte, error) {
	return forgejoRequest("GET", endpoint, nil)
}

func forgejoPost(endpoint string, body interface{}) ([]byte, error) {
	return forgejoRequest("POST", endpoint, body)
}

func forgejoPut(endpoint string, body interface{}) ([]byte, error) {
	return forgejoRequest("PUT", endpoint, body)
}

func forgejoDeleteWithBody(endpoint string, body interface{}) ([]byte, error) {
	return forgejoRequest("DELETE", endpoint, body)
}

func forgejoGetRaw(endpoint string) ([]byte, error) {
	url := forgejoURL + "/api/v1" + endpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if forgejoToken != "" {
		req.Header.Set("Authorization", "token "+forgejoToken)
	}
	req.Header.Set("Accept", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Tool implementations

func listRepos(args map[string]interface{}) CallToolResult {
	query := ""
	if q, ok := args["query"].(string); ok && q != "" {
		query = q
	}

	owner := ""
	if o, ok := args["owner"].(string); ok && o != "" {
		owner = o
	}

	limit := 50
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var endpoint string
	if owner != "" {
		endpoint = fmt.Sprintf("/users/%s/repos?limit=%d", owner, limit)
	} else if forgejoToken != "" && query == "" {
		endpoint = fmt.Sprintf("/user/repos?limit=%d", limit)
	} else if query != "" {
		endpoint = fmt.Sprintf("/repos/search?q=%s&limit=%d", query, limit)
	} else {
		endpoint = fmt.Sprintf("/repos/search?limit=%d", limit)
	}

	body, err := forgejoGet(endpoint)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var searchResult ForgejoSearchResult
	if err := json.Unmarshal(body, &searchResult); err == nil && searchResult.OK {
		return formatRepoList(searchResult.Data)
	}

	var repos []ForgejoRepo
	if err := json.Unmarshal(body, &repos); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	return formatRepoList(repos)
}

func formatRepoList(repos []ForgejoRepo) CallToolResult {
	if len(repos) == 0 {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "No repositories found."}},
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d repositories:\n\n", len(repos)))

	for _, repo := range repos {
		visibility := "public"
		if repo.Private {
			visibility = "private"
		}
		sb.WriteString(fmt.Sprintf("## %s\n", repo.FullName))
		if repo.Description != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", repo.Description))
		}
		sb.WriteString(fmt.Sprintf("  URL: %s\n", repo.HTMLURL))
		sb.WriteString(fmt.Sprintf("  Visibility: %s | Stars: %d | Forks: %d\n", visibility, repo.Stars, repo.Forks))
		sb.WriteString(fmt.Sprintf("  Default branch: %s\n\n", repo.DefaultBranch))
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: sb.String()}},
	}
}

func getRepo(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)

	if owner == "" || repo == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner and repo are required"}},
			IsError: true,
		}
	}

	body, err := forgejoGet(fmt.Sprintf("/repos/%s/%s", owner, repo))
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var r ForgejoRepo
	if err := json.Unmarshal(body, &r); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	visibility := "public"
	if r.Private {
		visibility = "private"
	}

	text := fmt.Sprintf(`# %s

%s

- **URL:** %s
- **Clone URL:** %s
- **SSH URL:** %s
- **Visibility:** %s
- **Stars:** %d | **Forks:** %d | **Watchers:** %d
- **Size:** %d KB
- **Default branch:** %s
- **Created:** %s
- **Updated:** %s
`, r.FullName, r.Description, r.HTMLURL, r.CloneURL, r.SSHURL, visibility, r.Stars, r.Forks, r.Watchers, r.Size, r.DefaultBranch, r.CreatedAt, r.UpdatedAt)

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func getFile(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	path, _ := args["path"].(string)
	ref, _ := args["ref"].(string)

	if owner == "" || repo == "" || path == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, and path are required"}},
			IsError: true,
		}
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
	if ref != "" {
		endpoint += "?ref=" + ref
	}

	body, err := forgejoGet(endpoint)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var fileContent struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		Name     string `json:"name"`
		Path     string `json:"path"`
		Size     int    `json:"size"`
		SHA      string `json:"sha"`
	}

	if err := json.Unmarshal(body, &fileContent); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	content := fileContent.Content
	if fileContent.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
		if err != nil {
			return CallToolResult{
				Content: []ContentBlock{{Type: "text", Text: "Error decoding file: " + err.Error()}},
				IsError: true,
			}
		}
		content = string(decoded)
	}

	text := fmt.Sprintf("# %s\n\nPath: %s\nSize: %d bytes\nSHA: %s\n\n```\n%s\n```", fileContent.Name, fileContent.Path, fileContent.Size, fileContent.SHA, content)

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func listBranches(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)

	if owner == "" || repo == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner and repo are required"}},
			IsError: true,
		}
	}

	body, err := forgejoGet(fmt.Sprintf("/repos/%s/%s/branches", owner, repo))
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var branches []ForgejoBranch
	if err := json.Unmarshal(body, &branches); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	if len(branches) == 0 {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "No branches found."}},
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d branches in %s/%s:\n\n", len(branches), owner, repo))

	for _, b := range branches {
		protected := ""
		if b.Protected {
			protected = " [protected]"
		}
		sb.WriteString(fmt.Sprintf("- **%s**%s (commit: %s)\n", b.Name, protected, b.Commit.SHA[:7]))
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: sb.String()}},
	}
}

func getBranch(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	branch, _ := args["branch"].(string)

	if owner == "" || repo == "" || branch == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, and branch are required"}},
			IsError: true,
		}
	}

	body, err := forgejoGet(fmt.Sprintf("/repos/%s/%s/branches/%s", owner, repo, branch))
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var b ForgejoBranch
	if err := json.Unmarshal(body, &b); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	protected := "No"
	if b.Protected {
		protected = "Yes"
	}

	text := fmt.Sprintf(`# Branch: %s

- **Repository:** %s/%s
- **Latest commit:** %s
- **Protected:** %s
`, b.Name, owner, repo, b.Commit.SHA, protected)

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func listCommits(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	sha, _ := args["sha"].(string)

	limit := 30
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	if owner == "" || repo == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner and repo are required"}},
			IsError: true,
		}
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/commits?limit=%d", owner, repo, limit)
	if sha != "" {
		endpoint += "&sha=" + sha
	}

	body, err := forgejoGet(endpoint)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var commits []ForgejoCommit
	if err := json.Unmarshal(body, &commits); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	if len(commits) == 0 {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "No commits found."}},
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d commits in %s/%s:\n\n", len(commits), owner, repo))

	for _, c := range commits {
		message := strings.Split(c.Commit.Message, "\n")[0]
		if len(message) > 72 {
			message = message[:72] + "..."
		}
		author := c.Commit.Author.Name
		date := c.Commit.Author.Date
		if len(date) > 10 {
			date = date[:10]
		}
		sb.WriteString(fmt.Sprintf("- `%s` %s (%s, %s)\n", c.SHA[:7], message, author, date))
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: sb.String()}},
	}
}

func getCommit(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	sha, _ := args["sha"].(string)

	if owner == "" || repo == "" || sha == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, and sha are required"}},
			IsError: true,
		}
	}

	body, err := forgejoGet(fmt.Sprintf("/repos/%s/%s/git/commits/%s", owner, repo, sha))
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var c ForgejoCommit
	if err := json.Unmarshal(body, &c); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	text := fmt.Sprintf(`# Commit %s

**Message:**
%s

- **Author:** %s <%s>
- **Date:** %s
- **Committer:** %s <%s>
- **URL:** %s
`, c.SHA, c.Commit.Message, c.Commit.Author.Name, c.Commit.Author.Email, c.Commit.Author.Date, c.Commit.Committer.Name, c.Commit.Committer.Email, c.HTMLURL)

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func listIssues(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	state := "open"
	if s, ok := args["state"].(string); ok && s != "" {
		state = s
	}

	if owner == "" || repo == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner and repo are required"}},
			IsError: true,
		}
	}

	body, err := forgejoGet(fmt.Sprintf("/repos/%s/%s/issues?state=%s&type=issues", owner, repo, state))
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var issues []ForgejoIssue
	if err := json.Unmarshal(body, &issues); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	if len(issues) == 0 {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("No %s issues found in %s/%s.", state, owner, repo)}},
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d %s issues in %s/%s:\n\n", len(issues), state, owner, repo))

	for _, issue := range issues {
		sb.WriteString(fmt.Sprintf("## #%d: %s\n", issue.Number, issue.Title))
		sb.WriteString(fmt.Sprintf("  State: %s | Author: %s\n", issue.State, issue.User.Login))
		sb.WriteString(fmt.Sprintf("  URL: %s\n", issue.HTMLURL))
		if issue.Body != "" {
			body := issue.Body
			if len(body) > 200 {
				body = body[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("  %s\n", body))
		}
		sb.WriteString("\n")
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: sb.String()}},
	}
}

func getIssue(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	index := 0
	if i, ok := args["index"].(float64); ok {
		index = int(i)
	}

	if owner == "" || repo == "" || index == 0 {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, and index are required"}},
			IsError: true,
		}
	}

	body, err := forgejoGet(fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, index))
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var issue ForgejoIssue
	if err := json.Unmarshal(body, &issue); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	text := fmt.Sprintf(`# #%d: %s

**State:** %s
**Author:** %s
**Created:** %s
**Updated:** %s
**URL:** %s

## Description

%s
`, issue.Number, issue.Title, issue.State, issue.User.Login, issue.CreatedAt, issue.UpdatedAt, issue.HTMLURL, issue.Body)

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func createIssue(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	title, _ := args["title"].(string)
	body, _ := args["body"].(string)

	if owner == "" || repo == "" || title == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, and title are required"}},
			IsError: true,
		}
	}

	payload := map[string]string{
		"title": title,
		"body":  body,
	}

	respBody, err := forgejoPost(fmt.Sprintf("/repos/%s/%s/issues", owner, repo), payload)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var issue ForgejoIssue
	if err := json.Unmarshal(respBody, &issue); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	text := fmt.Sprintf("Created issue #%d: %s\nURL: %s", issue.Number, issue.Title, issue.HTMLURL)

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func listPullRequests(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	state := "open"
	if s, ok := args["state"].(string); ok && s != "" {
		state = s
	}

	if owner == "" || repo == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner and repo are required"}},
			IsError: true,
		}
	}

	body, err := forgejoGet(fmt.Sprintf("/repos/%s/%s/pulls?state=%s", owner, repo, state))
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var prs []ForgejoPullRequest
	if err := json.Unmarshal(body, &prs); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	if len(prs) == 0 {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("No %s pull requests found in %s/%s.", state, owner, repo)}},
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d %s pull requests in %s/%s:\n\n", len(prs), state, owner, repo))

	for _, pr := range prs {
		sb.WriteString(fmt.Sprintf("## #%d: %s\n", pr.Number, pr.Title))
		sb.WriteString(fmt.Sprintf("  %s -> %s | State: %s | Author: %s\n", pr.Head.Ref, pr.Base.Ref, pr.State, pr.User.Login))
		sb.WriteString(fmt.Sprintf("  URL: %s\n", pr.HTMLURL))
		if pr.Body != "" {
			body := pr.Body
			if len(body) > 200 {
				body = body[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("  %s\n", body))
		}
		sb.WriteString("\n")
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: sb.String()}},
	}
}

func getPullRequest(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	index := 0
	if i, ok := args["index"].(float64); ok {
		index = int(i)
	}

	if owner == "" || repo == "" || index == 0 {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, and index are required"}},
			IsError: true,
		}
	}

	body, err := forgejoGet(fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, index))
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var pr ForgejoPullRequest
	if err := json.Unmarshal(body, &pr); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	merged := "No"
	if pr.Merged {
		merged = fmt.Sprintf("Yes (at %s)", pr.MergedAt)
	}

	mergeable := "Unknown"
	if pr.Mergeable {
		mergeable = "Yes"
	}

	text := fmt.Sprintf(`# PR #%d: %s

**State:** %s
**Author:** %s
**Branch:** %s -> %s
**Merged:** %s
**Mergeable:** %s
**Created:** %s
**Updated:** %s
**URL:** %s

## Description

%s
`, pr.Number, pr.Title, pr.State, pr.User.Login, pr.Head.Ref, pr.Base.Ref, merged, mergeable, pr.CreatedAt, pr.UpdatedAt, pr.HTMLURL, pr.Body)

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func getPullRequestDiff(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	index := 0
	if i, ok := args["index"].(float64); ok {
		index = int(i)
	}

	if owner == "" || repo == "" || index == 0 {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, and index are required"}},
			IsError: true,
		}
	}

	body, err := forgejoGetRaw(fmt.Sprintf("/repos/%s/%s/pulls/%d.diff", owner, repo, index))
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	text := fmt.Sprintf("# Diff for PR #%d\n\n```diff\n%s\n```", index, string(body))

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func createPullRequest(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	title, _ := args["title"].(string)
	body, _ := args["body"].(string)
	head, _ := args["head"].(string)
	base, _ := args["base"].(string)

	if owner == "" || repo == "" || title == "" || head == "" || base == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, title, head, and base are required"}},
			IsError: true,
		}
	}

	payload := map[string]string{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
	}

	respBody, err := forgejoPost(fmt.Sprintf("/repos/%s/%s/pulls", owner, repo), payload)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var pr ForgejoPullRequest
	if err := json.Unmarshal(respBody, &pr); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	text := fmt.Sprintf("Created PR #%d: %s\nBranch: %s -> %s\nURL: %s", pr.Number, pr.Title, pr.Head.Ref, pr.Base.Ref, pr.HTMLURL)

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func mergePullRequest(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	index := 0
	if i, ok := args["index"].(float64); ok {
		index = int(i)
	}
	mergeStyle := "merge"
	if s, ok := args["merge_style"].(string); ok && s != "" {
		mergeStyle = s
	}
	title, _ := args["title"].(string)
	message, _ := args["message"].(string)
	deleteBranch := false
	if d, ok := args["delete_branch"].(bool); ok {
		deleteBranch = d
	}

	if owner == "" || repo == "" || index == 0 {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, and index are required"}},
			IsError: true,
		}
	}

	payload := map[string]interface{}{
		"Do":                        mergeStyle,
		"delete_branch_after_merge": deleteBranch,
	}
	if title != "" {
		payload["MergeTitleField"] = title
	}
	if message != "" {
		payload["MergeMessageField"] = message
	}

	_, err := forgejoPost(fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", owner, repo, index), payload)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	text := fmt.Sprintf("Successfully merged PR #%d using %s strategy.", index, mergeStyle)
	if deleteBranch {
		text += " Source branch deleted."
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func createBranch(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	newBranch, _ := args["new_branch_name"].(string)
	oldBranch, _ := args["old_branch_name"].(string)

	if owner == "" || repo == "" || newBranch == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, and new_branch_name are required"}},
			IsError: true,
		}
	}

	payload := map[string]string{
		"new_branch_name": newBranch,
	}
	if oldBranch != "" {
		payload["old_branch_name"] = oldBranch
	}

	respBody, err := forgejoPost(fmt.Sprintf("/repos/%s/%s/branches", owner, repo), payload)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var b ForgejoBranch
	if err := json.Unmarshal(respBody, &b); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	text := fmt.Sprintf("Created branch '%s' in %s/%s (commit: %s)", b.Name, owner, repo, b.Commit.SHA[:7])

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func createOrUpdateFile(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	message, _ := args["message"].(string)
	branch, _ := args["branch"].(string)
	sha, _ := args["sha"].(string)

	if owner == "" || repo == "" || path == "" || content == "" || message == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, path, content, and message are required"}},
			IsError: true,
		}
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(content))

	payload := map[string]interface{}{
		"content": encoded,
		"message": message,
	}
	if branch != "" {
		payload["branch"] = branch
	}
	if sha != "" {
		payload["sha"] = sha
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)

	var respBody []byte
	var err error
	if sha != "" {
		respBody, err = forgejoPut(endpoint, payload)
	} else {
		respBody, err = forgejoPost(endpoint, payload)
	}
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	var result struct {
		Content struct {
			Name string `json:"name"`
			Path string `json:"path"`
			SHA  string `json:"sha"`
		} `json:"content"`
		Commit struct {
			SHA     string `json:"sha"`
			Message string `json:"message"`
			HTMLURL string `json:"html_url"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error parsing response: " + err.Error()}},
			IsError: true,
		}
	}

	action := "Created"
	if sha != "" {
		action = "Updated"
	}
	text := fmt.Sprintf("%s file '%s' in %s/%s\nCommit: %s\nMessage: %s", action, result.Content.Path, owner, repo, result.Commit.SHA[:7], result.Commit.Message)
	if result.Commit.HTMLURL != "" {
		text += fmt.Sprintf("\nURL: %s", result.Commit.HTMLURL)
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func deleteFile(args map[string]interface{}) CallToolResult {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	path, _ := args["path"].(string)
	message, _ := args["message"].(string)
	branch, _ := args["branch"].(string)
	sha, _ := args["sha"].(string)

	if owner == "" || repo == "" || path == "" || message == "" || sha == "" {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: owner, repo, path, message, and sha are required"}},
			IsError: true,
		}
	}

	payload := map[string]interface{}{
		"message": message,
		"sha":     sha,
	}
	if branch != "" {
		payload["branch"] = branch
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)

	_, err := forgejoDeleteWithBody(endpoint, payload)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
			IsError: true,
		}
	}

	text := fmt.Sprintf("Deleted file '%s' from %s/%s\nCommit message: %s", path, owner, repo, message)

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}
