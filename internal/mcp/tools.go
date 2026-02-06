package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type FSReadInput struct {
	Path string `json:"path" jsonschema:"relative file path"`
}

type FSReadOutput struct {
	Content string `json:"content" jsonschema:"file content"`
}

type FSWriteInput struct {
	Path    string `json:"path" jsonschema:"relative file path"`
	Content string `json:"content" jsonschema:"file content"`
}

type FSWriteOutput struct {
	OK bool `json:"ok" jsonschema:"write result"`
}

type FSListInput struct {
	Path      string `json:"path" jsonschema:"relative directory path"`
	Recursive bool   `json:"recursive" jsonschema:"recursive listing"`
}

type FSFileEntry struct {
	Path    string    `json:"path" jsonschema:"relative entry path"`
	IsDir   bool      `json:"is_dir" jsonschema:"is directory"`
	Size    int64     `json:"size" jsonschema:"entry size"`
	Mode    uint32    `json:"mode" jsonschema:"file mode"`
	ModTime time.Time `json:"mod_time" jsonschema:"modification time"`
}

type FSListOutput struct {
	Path    string        `json:"path" jsonschema:"listed path"`
	Entries []FSFileEntry `json:"entries" jsonschema:"entries"`
}

type FSEditInput struct {
	Path  string `json:"path" jsonschema:"relative file path"`
	Patch string `json:"patch" jsonschema:"unified diff patch"`
}

type FSEditOutput struct {
	OK bool `json:"ok" jsonschema:"apply result"`
}

type ExecInput struct {
	Command string   `json:"command" jsonschema:"command to run"`
	Args    []string `json:"args" jsonschema:"command arguments"`
}

type ExecOutput struct {
	OK       bool   `json:"ok" jsonschema:"execution success"`
	ExitCode int    `json:"exit_code" jsonschema:"process exit code"`
	Stdout   string `json:"stdout" jsonschema:"standard output"`
	Stderr   string `json:"stderr" jsonschema:"standard error"`
}

func RegisterTools(server *sdkmcp.Server) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "read", Description: "read file content"}, fsReadTool)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "write", Description: "write file content"}, fsWriteTool)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "list", Description: "list directory entries"}, fsListTool)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "edit", Description: "apply unified diff patch"}, fsEditTool)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "exec", Description: "execute command"}, execTool)
}

func fsReadTool(ctx context.Context, req *sdkmcp.CallToolRequest, input FSReadInput) (
	*sdkmcp.CallToolResult,
	FSReadOutput,
	error,
) {
	root := dataRoot()
	target, err := resolvePath(root, input.Path)
	if err != nil {
		return nil, FSReadOutput{}, err
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return nil, FSReadOutput{}, err
	}
	return nil, FSReadOutput{Content: string(data)}, nil
}

func fsWriteTool(ctx context.Context, req *sdkmcp.CallToolRequest, input FSWriteInput) (
	*sdkmcp.CallToolResult,
	FSWriteOutput,
	error,
) {
	root := dataRoot()
	target, err := resolvePath(root, input.Path)
	if err != nil {
		return nil, FSWriteOutput{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return nil, FSWriteOutput{}, err
	}
	if err := os.WriteFile(target, []byte(input.Content), 0o644); err != nil {
		return nil, FSWriteOutput{}, err
	}
	return nil, FSWriteOutput{OK: true}, nil
}

func fsListTool(ctx context.Context, req *sdkmcp.CallToolRequest, input FSListInput) (
	*sdkmcp.CallToolResult,
	FSListOutput,
	error,
) {
	root := dataRoot()
	target, err := resolvePathAllowRoot(root, input.Path)
	if err != nil {
		return nil, FSListOutput{}, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, FSListOutput{}, err
	}
	if !info.IsDir() {
		return nil, FSListOutput{}, fmt.Errorf("path is not a directory")
	}

	entries := []FSFileEntry{}
	if input.Recursive {
		err = filepath.WalkDir(target, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if p == target {
				return nil
			}
			entryInfo, err := d.Info()
			if err != nil {
				return err
			}
			entry, err := entryForPath(root, p, entryInfo)
			if err != nil {
				return err
			}
			entries = append(entries, entry)
			return nil
		})
	} else {
		dirEntries, err := os.ReadDir(target)
		if err != nil {
			return nil, FSListOutput{}, err
		}
		for _, entry := range dirEntries {
			entryInfo, err := entry.Info()
			if err != nil {
				return nil, FSListOutput{}, err
			}
			fullPath := filepath.Join(target, entry.Name())
			fileEntry, err := entryForPath(root, fullPath, entryInfo)
			if err != nil {
				return nil, FSListOutput{}, err
			}
			entries = append(entries, fileEntry)
		}
	}
	if err != nil {
		return nil, FSListOutput{}, err
	}

	listedPath := strings.TrimSpace(input.Path)
	if listedPath == "" {
		listedPath = "."
	}
	return nil, FSListOutput{Path: listedPath, Entries: entries}, nil
}

func fsEditTool(ctx context.Context, req *sdkmcp.CallToolRequest, input FSEditInput) (
	*sdkmcp.CallToolResult,
	FSEditOutput,
	error,
) {
	root := dataRoot()
	target, err := resolvePath(root, input.Path)
	if err != nil {
		return nil, FSEditOutput{}, err
	}
	orig, err := os.ReadFile(target)
	if err != nil {
		return nil, FSEditOutput{}, err
	}
	updated, err := applyUnifiedPatch(string(orig), input.Patch)
	if err != nil {
		return nil, FSEditOutput{}, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, FSEditOutput{}, err
	}
	if err := os.WriteFile(target, []byte(updated), info.Mode().Perm()); err != nil {
		return nil, FSEditOutput{}, err
	}
	return nil, FSEditOutput{OK: true}, nil
}

func execTool(ctx context.Context, req *sdkmcp.CallToolRequest, input ExecInput) (
	*sdkmcp.CallToolResult,
	ExecOutput,
	error,
) {
	if strings.TrimSpace(input.Command) == "" {
		return nil, ExecOutput{}, fmt.Errorf("command is required")
	}
	cmd := exec.CommandContext(ctx, input.Command, input.Args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, ExecOutput{
				OK:       false,
				ExitCode: exitErr.ExitCode(),
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
			}, nil
		}
		return nil, ExecOutput{}, err
	}

	return nil, ExecOutput{
		OK:       true,
		ExitCode: 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

func dataRoot() string {
	root := strings.TrimSpace(os.Getenv("MCP_DATA_DIR"))
	if root == "" {
		root = "/data"
	}
	return root
}

func resolvePathAllowRoot(root, requestPath string) (string, error) {
	if strings.TrimSpace(requestPath) == "" {
		return root, nil
	}
	return resolvePath(root, requestPath)
}

func resolvePath(root, requestPath string) (string, error) {
	clean := filepath.Clean(requestPath)
	if clean == "." || clean == "" {
		return "", os.ErrInvalid
	}
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", os.ErrInvalid
	}
	return filepath.Join(root, clean), nil
}

func entryForPath(root, target string, info os.FileInfo) (FSFileEntry, error) {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return FSFileEntry{}, err
	}
	if strings.HasPrefix(rel, "..") {
		return FSFileEntry{}, os.ErrInvalid
	}
	if rel == "." {
		rel = ""
	}
	return FSFileEntry{
		Path:    filepath.ToSlash(rel),
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		Mode:    uint32(info.Mode().Perm()),
		ModTime: info.ModTime(),
	}, nil
}

func applyUnifiedPatch(original, patch string) (string, error) {
	lines := strings.Split(original, "\n")
	out := make([]string, 0, len(lines))
	index := 0
	patchLines := strings.Split(patch, "\n")
	hunksApplied := 0

	for i := 0; i < len(patchLines); i++ {
		line := patchLines[i]
		if !strings.HasPrefix(line, "@@") {
			continue
		}

		origStart, err := parseUnifiedHunkHeader(line)
		if err != nil {
			return "", err
		}
		origStart--
		if origStart < 0 {
			origStart = 0
		}
		if origStart > len(lines) {
			return "", fmt.Errorf("patch out of range")
		}

		out = append(out, lines[index:origStart]...)
		index = origStart
		hunksApplied++

		for i+1 < len(patchLines) {
			next := patchLines[i+1]
			if strings.HasPrefix(next, "@@") {
				break
			}
			i++

			if next == "" {
				if i == len(patchLines)-1 {
					break
				}
				return "", fmt.Errorf("invalid patch line")
			}
			if next[0] == '\\' {
				continue
			}
			op := next[0]
			text := next[1:]
			switch op {
			case ' ':
				if index >= len(lines) || lines[index] != text {
					return "", fmt.Errorf("patch context mismatch")
				}
				out = append(out, text)
				index++
			case '-':
				if index >= len(lines) || lines[index] != text {
					return "", fmt.Errorf("patch delete mismatch")
				}
				index++
			case '+':
				out = append(out, text)
			default:
				return "", fmt.Errorf("invalid patch operation")
			}
		}
	}
	if hunksApplied == 0 {
		return "", fmt.Errorf("patch contains no hunks")
	}

	out = append(out, lines[index:]...)
	return strings.Join(out, "\n"), nil
}

func parseUnifiedHunkHeader(header string) (int, error) {
	trimmed := strings.TrimPrefix(header, "@@")
	trimmed = strings.TrimSpace(trimmed)
	if !strings.HasPrefix(trimmed, "-") {
		return 0, fmt.Errorf("invalid hunk header")
	}
	parts := strings.SplitN(trimmed, " ", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid hunk header")
	}

	origPart := strings.TrimPrefix(parts[0], "-")
	origFields := strings.SplitN(origPart, ",", 2)
	origStart, err := strconv.Atoi(origFields[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hunk header")
	}
	return origStart, nil
}
