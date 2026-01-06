/*
Copyright 2025 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// LogSource represents a single source of logs (e.g., a container)
type LogSource struct {
	Name      string
	PodName   string
	Container string
	Color     tcell.Color
	Lines     []string
	mu        sync.RWMutex
}

// AddLine adds a log line to this source
func (ls *LogSource) AddLine(line string) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	// Remove trailing newlines
	line = strings.TrimRight(line, "\n\r")
	if line != "" {
		ls.Lines = append(ls.Lines, line)
		// Keep only last 10000 lines to prevent memory issues
		if len(ls.Lines) > 10000 {
			ls.Lines = ls.Lines[len(ls.Lines)-10000:]
		}
	}
}

// GetLines returns all log lines for this source
func (ls *LogSource) GetLines() []string {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	result := make([]string, len(ls.Lines))
	copy(result, ls.Lines)
	return result
}

// TUILogger manages terminal UI for displaying logs from multiple sources
type TUILogger struct {
	app         *tview.Application
	list        *tview.List
	textView    *tview.TextView
	flex        *tview.Flex
	sources     map[string]*LogSource
	sourceOrder []string
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	started     bool
	colors      []tcell.Color
	colorIndex  int
}

// NewTUILogger creates a new TUI logger
func NewTUILogger(ctx context.Context) *TUILogger {
	app := tview.NewApplication()
	list := tview.NewList()
	textView := tview.NewTextView()

	// Configure list
	list.SetBorder(true).
		SetTitle(" Log Sources (↑↓ to navigate, Enter to select, Tab to switch panes) ").
		SetTitleAlign(tview.AlignLeft)

	// Configure text view
	textView.SetBorder(true).
		SetTitle(" Logs ").
		SetTitleAlign(tview.AlignLeft)
	textView.SetScrollable(true).
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	// Create flex layout
	flex := tview.NewFlex().
		AddItem(list, 0, 1, true).
		AddItem(textView, 0, 3, false)

	tuiCtx, cancel := context.WithCancel(ctx)

	colors := []tcell.Color{
		tcell.ColorGreen,
		tcell.ColorYellow,
		tcell.ColorBlue,
		tcell.ColorDarkMagenta,
		tcell.ColorDarkCyan,
		tcell.ColorOrange,
		tcell.ColorPurple,
		tcell.ColorTeal,
	}

	tui := &TUILogger{
		app:         app,
		list:        list,
		textView:    textView,
		flex:        flex,
		sources:     make(map[string]*LogSource),
		sourceOrder: make([]string, 0),
		ctx:         tuiCtx,
		cancel:      cancel,
		colors:      colors,
		colorIndex:  0,
	}

	// Set up list selection handler
	list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		tui.switchToSource(mainText)
	})

	// Set up input handler
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			tui.Stop()
			return nil
		case tcell.KeyTab:
			if app.GetFocus() == list {
				app.SetFocus(textView)
			} else {
				app.SetFocus(list)
			}
			return nil
		}
		return event
	})

	return tui
}

// Start starts the TUI application
func (t *TUILogger) Start() error {
	t.mu.Lock()
	if t.started {
		t.mu.Unlock()
		return fmt.Errorf("TUI already started")
	}
	t.started = true
	t.mu.Unlock()

	return t.app.SetRoot(t.flex, true).Run()
}

// Stop stops the TUI application
func (t *TUILogger) Stop() {
	t.cancel()
	if t.app != nil {
		t.app.Stop()
	}
}

// GetOrCreateSource gets or creates a log source
func (t *TUILogger) GetOrCreateSource(podName, containerName string) *LogSource {
	sourceName := fmt.Sprintf("%s/%s", podName, containerName)

	t.mu.Lock()
	defer t.mu.Unlock()

	if source, exists := t.sources[sourceName]; exists {
		return source
	}

	// Create new source
	color := t.colors[t.colorIndex%len(t.colors)]
	t.colorIndex++

	source := &LogSource{
		Name:      sourceName,
		PodName:   podName,
		Container: containerName,
		Color:     color,
		Lines:     make([]string, 0),
	}

	t.sources[sourceName] = source
	t.sourceOrder = append(t.sourceOrder, sourceName)

	// Add to list
	t.app.QueueUpdateDraw(func() {
		t.list.AddItem(sourceName, "", 0, nil)
		// Auto-select first source
		if len(t.sourceOrder) == 1 {
			t.list.SetCurrentItem(0)
			t.switchToSource(sourceName)
		}
	})

	return source
}

// WriteLog writes a log line to a specific source
func (t *TUILogger) WriteLog(podName, containerName, line string) {
	source := t.GetOrCreateSource(podName, containerName)
	source.AddLine(line)

	// Update display if this is the currently selected source
	t.mu.RLock()
	currentIndex := t.list.GetCurrentItem()
	if currentIndex >= 0 && currentIndex < len(t.sourceOrder) {
		currentSource := t.sourceOrder[currentIndex]
		if currentSource == source.Name {
			t.mu.RUnlock()
			t.updateDisplay(source)
			return
		}
	}
	t.mu.RUnlock()
}

// switchToSource switches the display to show logs from a specific source
func (t *TUILogger) switchToSource(sourceName string) {
	t.mu.RLock()
	source, exists := t.sources[sourceName]
	t.mu.RUnlock()

	if !exists {
		return
	}

	t.updateDisplay(source)
}

// updateDisplay updates the text view with logs from the given source
func (t *TUILogger) updateDisplay(source *LogSource) {
	lines := source.GetLines()

	t.app.QueueUpdateDraw(func() {
		t.textView.Clear()
		t.textView.SetTitle(fmt.Sprintf(" Logs: %s ", source.Name))

		// Use hex color format for tview
		colorHex := fmt.Sprintf("#%06x", source.Color.Hex())
		for _, line := range lines {
			fmt.Fprintf(t.textView, "[%s]%s[-]\n", colorHex, line)
		}

		// Auto-scroll to bottom
		t.textView.ScrollToEnd()
	})
}

// TUIWriter is an io.Writer that writes to a TUI logger
type TUIWriter struct {
	tui           *TUILogger
	podName       string
	containerName string
}

// NewTUIWriter creates a new TUI writer for a specific pod/container
func NewTUIWriter(tui *TUILogger, podName, containerName string) *TUIWriter {
	return &TUIWriter{
		tui:           tui,
		podName:       podName,
		containerName: containerName,
	}
}

// Write implements io.Writer
func (w *TUIWriter) Write(p []byte) (n int, err error) {
	if w.tui == nil {
		return 0, fmt.Errorf("TUI logger not initialized")
	}

	line := string(p)
	w.tui.WriteLog(w.podName, w.containerName, line)
	return len(p), nil
}

// IsStarted returns whether the TUI has been started
func (t *TUILogger) IsStarted() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.started
}

// WaitForStart waits for the TUI to be started with a timeout
func (t *TUILogger) WaitForStart(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if t.IsStarted() {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("TUI did not start within timeout")
}

var _ io.Writer = (*TUIWriter)(nil)
