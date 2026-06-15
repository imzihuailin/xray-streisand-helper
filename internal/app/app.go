package app

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/imzihuailin/xray-streisand-helper/internal/config"
	"github.com/imzihuailin/xray-streisand-helper/internal/system"
	"github.com/imzihuailin/xray-streisand-helper/internal/terminalqr"
	"github.com/imzihuailin/xray-streisand-helper/internal/upstream"
	"github.com/imzihuailin/xray-streisand-helper/internal/webui"
)

type App struct {
	In        io.Reader
	Out       io.Writer
	Err       io.Writer
	Version   string
	Commit    string
	BuildDate string
	Runner    system.Runner
	YAMLPath  string
	MetaPath  string
	Upstream  upstream.Manager
}

func New(in io.Reader, out, errOut io.Writer) *App {
	return &App{
		In: in, Out: out, Err: errOut,
		Runner:   system.ExecRunner{},
		YAMLPath: config.DefaultYAML, MetaPath: config.DefaultMetadata,
	}
}

func (a *App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return a.setup(ctx, false)
	}
	switch args[0] {
	case "setup":
		fs := flag.NewFlagSet("setup", flag.ContinueOnError)
		fs.SetOutput(a.Err)
		force := fs.Bool("force", false, "replace an existing valid installation")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return a.setup(ctx, *force)
	case "show":
		return a.show(ctx, a.YAMLPath, true)
	case "link":
		path := a.YAMLPath
		if len(args) > 2 {
			return errors.New("usage: link [yaml]")
		}
		if len(args) == 2 {
			path = args[1]
		}
		return a.printLink(path)
	case "serve":
		path := a.YAMLPath
		if len(args) > 2 {
			return errors.New("usage: serve [yaml]")
		}
		if len(args) == 2 {
			path = args[1]
		}
		link, err := loadLink(path)
		if err != nil {
			return err
		}
		return (webui.Server{}).Run(ctx, link, func(u string) {
			fmt.Fprintf(a.Out, "Open through an SSH tunnel: %s\n", u)
		})
	case "doctor":
		return a.doctor(ctx)
	case "--version", "version":
		fmt.Fprintf(a.Out, "xray-streisand-helper %s\ncommit: %s\nbuild date: %s\n", a.Version, a.Commit, a.BuildDate)
		return nil
	case "--help", "-h", "help":
		a.usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q; use --help", args[0])
	}
}

func (a *App) usage() {
	fmt.Fprintln(a.Out, `xray-streisand-helper

Usage:
  xray-streisand-helper setup [--force]
  xray-streisand-helper show
  xray-streisand-helper link [yaml]
  xray-streisand-helper serve [yaml]
  xray-streisand-helper doctor
  xray-streisand-helper --version`)
}

func (a *App) setup(ctx context.Context, force bool) error {
	if err := system.Platform(); err != nil {
		return err
	}
	if !system.IsRoot() {
		return errors.New("setup must run as root")
	}
	state := a.installState()
	if state == "valid" && !force {
		fmt.Fprintln(a.Out, "Existing valid configuration found; reusing UUID and Reality keys.")
		return a.show(ctx, a.YAMLPath, true)
	}
	if state == "partial" {
		return errors.New("partial Xray installation detected; repair or remove it before continuing")
	}
	if err := system.ExistingXrayKnown(ctx, a.Runner); err != nil {
		return err
	}
	if err := system.Port443Conflict(ctx, a.Runner); err != nil {
		return err
	}
	fmt.Fprint(a.Out, "Domain: ")
	domain, err := system.ReadDomain(bufio.NewReader(a.In))
	if err != nil {
		return err
	}
	bin, err := a.Upstream.Prepare(ctx)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, bin)
	cmd.Stdin = strings.NewReader(domain + "\n")
	cmd.Stdout = a.Out
	cmd.Stderr = a.Err
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("upstream installer failed: %w", err)
	}
	if err := a.doctor(ctx); err != nil {
		return fmt.Errorf("installation completed but validation failed: %w", err)
	}
	return a.show(ctx, a.YAMLPath, false)
}

func (a *App) installState() string {
	_, yerr := os.Stat(a.YAMLPath)
	_, merr := os.Stat(a.MetaPath)
	if yerr != nil && merr != nil {
		return "empty"
	}
	if yerr != nil || merr != nil {
		return "partial"
	}
	p, _, err := config.Load(a.YAMLPath)
	if err != nil {
		return "partial"
	}
	m, err := config.LoadMetadata(a.MetaPath)
	if err != nil || config.MatchMetadata(p, m) != nil {
		return "partial"
	}
	return "valid"
}

func (a *App) doctor(ctx context.Context) error {
	if err := system.Platform(); err != nil {
		return err
	}
	p, _, err := config.Load(a.YAMLPath)
	if err != nil {
		return err
	}
	m, err := config.LoadMetadata(a.MetaPath)
	if err != nil {
		return err
	}
	if err := config.MatchMetadata(p, m); err != nil {
		return err
	}
	for _, path := range []string{a.YAMLPath, a.MetaPath} {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("%s permissions must be private, got %o", path, info.Mode().Perm())
		}
	}
	if err := system.ServiceActive(ctx, a.Runner); err != nil {
		return err
	}
	if err := system.Port443OwnedByXray(ctx, a.Runner); err != nil {
		return err
	}
	if err := system.DNSMatches(ctx, m.Domain, m.PublicIP); err != nil {
		return err
	}
	fmt.Fprintln(a.Out, "doctor: all checks passed")
	return nil
}

func (a *App) show(ctx context.Context, path string, validateSystem bool) error {
	if validateSystem {
		if err := a.doctor(ctx); err != nil {
			return err
		}
	}
	link, err := loadLink(path)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.Out, link)
	width := 0
	if f, ok := a.Out.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil {
			width = w / 2
		}
	}
	if err := terminalqr.Render(a.Out, link, width); err != nil {
		fmt.Fprintf(a.Err, "QR unavailable: %v\n", err)
	}
	return nil
}

func (a *App) printLink(path string) error {
	link, err := loadLink(path)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.Out, link)
	return err
}

func loadLink(path string) (string, error) {
	p, fp, err := config.Load(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	return config.Link(p, fp)
}
