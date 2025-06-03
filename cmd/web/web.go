package web

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"time"

	"github.com/devflowinc/uzi/pkg/state"
	"github.com/peterbourgon/ff/v3/ffcli"
)

var (
	fs     = flag.NewFlagSet("uzi web", flag.ExitOnError)
	port   = fs.Int("port", 8080, "port for the web server")
	CmdWeb = &ffcli.Command{
		Name:       "web",
		ShortUsage: "uzi web",
		ShortHelp:  "Run a simple web UI for uzi",
		FlagSet:    fs,
		Exec:       run,
	}
)

//go:embed templates/index.html
var indexHTML string

var indexTmpl = template.Must(template.New("index.html").Parse(indexHTML))

type pageData struct {
	LsOutput string
}

func run(ctx context.Context, args []string) error {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/prompt", promptHandler)
	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{Addr: addr}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()
	fmt.Printf("Starting uzi web on %s\n", addr)
	return srv.ListenAndServe()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	lsOutput := runUziLs()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	indexTmpl.Execute(w, pageData{LsOutput: lsOutput})
}

func promptHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	prompt := r.FormValue("prompt")
	claude := r.FormValue("claude")
	codex := r.FormValue("codex")
	if claude == "" {
		claude = "3"
	}
	if codex == "" {
		codex = "5"
	}
	go func() {
		runPrompt(prompt, claude, codex)
	}()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func runPrompt(prompt, claude, codex string) {
	agents := fmt.Sprintf("claude:%s,codex:%s", claude, codex)
	cmd := exec.Command("./uzi", "prompt", "--agents", agents, prompt)
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Run()
	runAutoUntilDone()
}

func runAutoUntilDone() {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "./uzi", "auto")
	if err := cmd.Start(); err != nil {
		return
	}
	sm := state.NewStateManager()
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		sessions, _ := sm.GetActiveSessionsForRepo()
		if len(sessions) == 0 {
			cancel()
			ticker.Stop()
			cmd.Wait()
			return
		}
	}
}

func runUziLs() string {
	out, err := exec.Command("./uzi", "ls").CombinedOutput()
	if err != nil {
		return err.Error()
	}
	return string(out)
}
