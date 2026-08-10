package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Request struct {
	CommandPath  string         `json:"commandPath"`
	Args         map[string]any `json:"args,omitempty"`
	Format       string         `json:"format,omitempty"`
	Confirm      bool           `json:"confirm,omitempty"`
	ConfirmToken string         `json:"confirmToken,omitempty"`
}

type Result struct {
	CommandPath string `json:"commandPath"`
	Title       string `json:"title"`
	ReadOnly    bool   `json:"readOnly"`
	Output      string `json:"output"`
	Data        any    `json:"data,omitempty"`
	GeneratedAt string `json:"generatedAt"`
}

type Handler func(context.Context, map[string]any) (string, error)

type HandlerResult struct {
	Output string
	Data   any
}

type StructuredHandler func(context.Context, map[string]any) (HandlerResult, error)

type Command struct {
	Path              string
	Title             string
	Description       string
	ReadOnly          bool
	Handler           Handler
	StructuredHandler StructuredHandler
}

type Runner struct {
	commands map[string]Command
}

func NewRunner() (*Runner, error) {
	r := &Runner{commands: map[string]Command{}}
	for _, cmd := range commandDefinitions() {
		if err := r.register(cmd); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func MustNewRunner() *Runner {
	r, err := NewRunner()
	if err != nil {
		panic(err)
	}
	return r
}

func (r *Runner) Run(ctx context.Context, req Request) (Result, error) {
	path := NormalizeCommandPath(req.CommandPath)
	if path == "" || path == "help" {
		out := RenderHelp()
		return formatResult(req, Command{Path: "help", Title: "go-stock CLI 功能树", ReadOnly: true}, out), nil
	}

	cmd, ok := r.commands[path]
	if !ok {
		if strings.HasPrefix(path, "tool ") {
			if cmd, ok = r.resolveRawToolCommand(path); !ok {
				return Result{}, fmt.Errorf("unknown raw tool command %q; run `tool list` first", req.CommandPath)
			}
		} else {
			return Result{}, fmt.Errorf("unknown command path %q; run `help` to inspect supported commands", req.CommandPath)
		}
	}
	if cmd.Handler == nil && cmd.StructuredHandler == nil {
		return Result{}, fmt.Errorf("command %q is documented but not executable yet", cmd.Path)
	}

	args := cloneArgs(req.Args)
	if req.Confirm {
		args["confirm"] = true
	}
	if strings.TrimSpace(req.ConfirmToken) != "" {
		args["confirmToken"] = strings.TrimSpace(req.ConfirmToken)
	}

	if cmd.StructuredHandler != nil {
		handled, err := cmd.StructuredHandler(ctx, args)
		if err != nil {
			return Result{}, err
		}
		result := formatResult(req, cmd, handled.Output)
		result.Data = handled.Data
		return result, nil
	}
	out, err := cmd.Handler(ctx, args)
	if err != nil {
		return Result{}, err
	}
	return formatResult(req, cmd, out), nil
}

func (r *Runner) resolveRawToolCommand(path string) (Command, bool) {
	name := strings.TrimSpace(strings.TrimPrefix(path, "tool "))
	if name == "" || name == "list" || name == "info" {
		return Command{}, false
	}
	toolRunner := newToolRunner()
	entry, ok := toolRunner.findRawTool(name)
	if !ok || isBlockedRawTool(entry.Name) {
		return Command{}, false
	}
	return Command{
		Path:        "tool " + entry.Name,
		Title:       "归档工具：" + entry.Name,
		Description: entry.Desc,
		ReadOnly:    true,
		Handler:     toolRunner.call(entry.Name, nil),
	}, true
}

func (r *Runner) RunText(ctx context.Context, req Request) (string, error) {
	result, err := r.Run(ctx, req)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(strings.TrimSpace(req.Format), "json") {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return result.Output, nil
}

func (r *Runner) Commands() []Command {
	commands := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		commands = append(commands, cmd)
	}
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Path < commands[j].Path
	})
	return commands
}

func (r *Runner) register(cmd Command) error {
	cmd.Path = NormalizeCommandPath(cmd.Path)
	if cmd.Path == "" {
		return fmt.Errorf("command path is empty")
	}
	if cmd.Title == "" {
		cmd.Title = cmd.Path
	}
	if _, exists := r.commands[cmd.Path]; exists {
		return fmt.Errorf("duplicate command path %q", cmd.Path)
	}
	r.commands[cmd.Path] = cmd
	return nil
}

func NormalizeCommandPath(path string) string {
	path = strings.ToLower(strings.TrimSpace(path))
	path = strings.ReplaceAll(path, "/", " ")
	path = strings.ReplaceAll(path, "\\", " ")
	path = strings.ReplaceAll(path, "_", "-")
	return strings.Join(strings.Fields(path), " ")
}

func formatResult(req Request, cmd Command, out string) Result {
	return Result{
		CommandPath: cmd.Path,
		Title:       cmd.Title,
		ReadOnly:    cmd.ReadOnly,
		Output:      out,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
}

func cloneArgs(args map[string]any) map[string]any {
	cloned := map[string]any{}
	for k, v := range args {
		cloned[k] = v
	}
	return cloned
}
