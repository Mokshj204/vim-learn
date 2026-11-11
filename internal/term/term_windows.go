//go:build windows

package term

import (
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	enableProcessedInput        = 0x0001
	enableLineInput             = 0x0002
	enableEchoInput             = 0x0004
	enableWindowInput           = 0x0008
	enableVirtualTerminalInput  = 0x0200
	enableProcessedOutput       = 0x0001
	enableVirtualTerminalOutput = 0x0004
	disableNewlineAutoReturn    = 0x0008
	cpUTF8                      = 65001
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procGetConsoleOutputCP         = kernel32.NewProc("GetConsoleOutputCP")
	procSetConsoleOutputCP         = kernel32.NewProc("SetConsoleOutputCP")
	procGetConsoleCP               = kernel32.NewProc("GetConsoleCP")
	procSetConsoleCP               = kernel32.NewProc("SetConsoleCP")
)

type coord struct {
	X int16
	Y int16
}

type smallRect struct {
	Left   int16
	Top    int16
	Right  int16
	Bottom int16
}

type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

type windowsTerm struct {
	inHandle  syscall.Handle
	outHandle syscall.Handle
	origIn    uint32
	origOut   uint32
	origOutCP uint32
	origInCP  uint32
	active    bool
	mu        sync.Mutex
	resizeCh  chan struct{}
	stopCh    chan struct{}
}

// New creates a Terminal implementation for Windows PowerShell / Console.
func New() Terminal {
	return &windowsTerm{
		inHandle:  syscall.Handle(os.Stdin.Fd()),
		outHandle: syscall.Handle(os.Stdout.Fd()),
		resizeCh:  make(chan struct{}, 4),
	}
}

func getConsoleMode(h syscall.Handle) (uint32, error) {
	var mode uint32
	r1, _, err := procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	if r1 == 0 {
		return 0, err
	}
	return mode, nil
}

func setConsoleMode(h syscall.Handle, mode uint32) error {
	r1, _, err := procSetConsoleMode.Call(uintptr(h), uintptr(mode))
	if r1 == 0 {
		return err
	}
	return nil
}

func (t *windowsTerm) EnableRaw() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.active {
		return nil
	}

	inMode, err := getConsoleMode(t.inHandle)
	if err != nil {
		return fmt.Errorf("stdin is not an interactive Windows console (if using Git Bash mintty, run via PowerShell or winpty): %w", err)
	}
	outMode, err := getConsoleMode(t.outHandle)
	if err != nil {
		return fmt.Errorf("stdout is not an interactive Windows console: %w", err)
	}

	t.origIn = inMode
	t.origOut = outMode

	rOutCP, _, _ := procGetConsoleOutputCP.Call()
	rInCP, _, _ := procGetConsoleCP.Call()
	t.origOutCP = uint32(rOutCP)
	t.origInCP = uint32(rInCP)

	procSetConsoleOutputCP.Call(uintptr(cpUTF8))
	procSetConsoleCP.Call(uintptr(cpUTF8))

	newIn := (inMode &^ (enableProcessedInput | enableLineInput | enableEchoInput)) | enableVirtualTerminalInput | enableWindowInput
	if err := setConsoleMode(t.inHandle, newIn); err != nil {
		return fmt.Errorf("failed to enable raw stdin mode: %w", err)
	}

	newOut := outMode | enableProcessedOutput | enableVirtualTerminalOutput | disableNewlineAutoReturn
	if err := setConsoleMode(t.outHandle, newOut); err != nil {
		_ = setConsoleMode(t.inHandle, t.origIn)
		return fmt.Errorf("failed to enable ANSI virtual terminal output: %w", err)
	}

	t.active = true
	t.stopCh = make(chan struct{})
	go t.watchResize()
	return nil
}

func (t *windowsTerm) Restore() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.active {
		return nil
	}
	close(t.stopCh)
	t.active = false

	_ = setConsoleMode(t.inHandle, t.origIn)
	_ = setConsoleMode(t.outHandle, t.origOut)
	if t.origOutCP != 0 {
		procSetConsoleOutputCP.Call(uintptr(t.origOutCP))
	}
	if t.origInCP != 0 {
		procSetConsoleCP.Call(uintptr(t.origInCP))
	}
	return nil
}

func (t *windowsTerm) Size() (int, int, error) {
	var info consoleScreenBufferInfo
	r1, _, err := procGetConsoleScreenBufferInfo.Call(uintptr(t.outHandle), uintptr(unsafe.Pointer(&info)))
	if r1 == 0 {
		return 80, 24, err
	}
	cols := int(info.Window.Right - info.Window.Left + 1)
	rows := int(info.Window.Bottom - info.Window.Top + 1)
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	return cols, rows, nil
}

func (t *windowsTerm) watchResize() {
	lastCols, lastRows, _ := t.Size()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			cols, rows, err := t.Size()
			if err == nil && (cols != lastCols || rows != lastRows) {
				lastCols, lastRows = cols, rows
				select {
				case t.resizeCh <- struct{}{}:
				default:
				}
			}
		}
	}
}

func (t *windowsTerm) Reader() io.Reader {
	return os.Stdin
}

func (t *windowsTerm) Writer() io.Writer {
	return os.Stdout
}

func (t *windowsTerm) ResizeNotify() <-chan struct{} {
	return t.resizeCh
}
