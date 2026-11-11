//go:build unix

package term

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type unixTerm struct {
	tty      *os.File
	ownsTTY  bool
	origStty string
	active   bool
	mu       sync.Mutex
	resizeCh chan struct{}
	sigCh    chan os.Signal
}

// New creates a POSIX-compliant Terminal implementation using stty and ANSI escapes.
func New() Terminal {
	return &unixTerm{
		resizeCh: make(chan struct{}, 4),
	}
}

func (t *unixTerm) openTTY() (*os.File, error) {
	if t.tty != nil {
		return t.tty, nil
	}
	if f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		t.tty = f
		t.ownsTTY = true
		return t.tty, nil
	}
	t.tty = os.Stdin
	t.ownsTTY = false
	return t.tty, nil
}

func (t *unixTerm) runStty(args ...string) (string, error) {
	f, err := t.openTTY()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("stty", args...)
	cmd.Stdin = f
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("stty %v failed: %w (%s)", args, err, strings.TrimSpace(errBuf.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

func (t *unixTerm) EnableRaw() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.active {
		return nil
	}

	orig, err := t.runStty("-g")
	if err != nil {
		return err
	}
	t.origStty = orig

	// POSIX raw-ish mode: disable echo, canonical line buffering, signals (so Ctrl-C
	// is captured cleanly), XON/XOFF flow control, and CR->NL translation.
	if _, err := t.runStty("-echo", "-icanon", "-isig", "-ixon", "-icrnl", "min", "1", "time", "0"); err != nil {
		return err
	}

	t.active = true
	t.sigCh = make(chan os.Signal, 4)
	signal.Notify(t.sigCh, syscall.SIGWINCH)
	go func(ch chan os.Signal) {
		for range ch {
			select {
			case t.resizeCh <- struct{}{}:
			default:
			}
		}
	}(t.sigCh)
	return nil
}

func (t *unixTerm) Restore() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.active {
		return nil
	}
	t.active = false
	if t.sigCh != nil {
		signal.Stop(t.sigCh)
		close(t.sigCh)
		t.sigCh = nil
	}
	var err error
	if t.origStty != "" {
		_, err = t.runStty(t.origStty)
	}
	if t.ownsTTY && t.tty != nil {
		_ = t.tty.Close()
		t.tty = nil
		t.ownsTTY = false
	}
	return err
}

func (t *unixTerm) Size() (int, int, error) {
	out, err := t.runStty("size")
	if err != nil {
		return 80, 24, err
	}
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return 80, 24, fmt.Errorf("unexpected stty size output: %q", out)
	}
	rows, err1 := strconv.Atoi(fields[0])
	cols, err2 := strconv.Atoi(fields[1])
	if err1 != nil || err2 != nil || rows <= 0 || cols <= 0 {
		return 80, 24, fmt.Errorf("invalid stty size: %q", out)
	}
	return cols, rows, nil
}

func (t *unixTerm) Reader() io.Reader {
	if t.tty != nil {
		return t.tty
	}
	return os.Stdin
}

func (t *unixTerm) Writer() io.Writer {
	if t.tty != nil {
		return t.tty
	}
	return os.Stdout
}

func (t *unixTerm) ResizeNotify() <-chan struct{} {
	return t.resizeCh
}
