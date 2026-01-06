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
	"testing"
	"time"
)

func TestNewTUILogger(t *testing.T) {
	t.Skip("Skipping TUI tests - requires terminal")
	ctx := context.Background()
	tui := NewTUILogger(ctx)

	if tui == nil {
		t.Fatal("NewTUILogger returned nil")
	}

	if tui.app == nil {
		t.Error("TUI app not initialized")
	}

	if tui.sources == nil {
		t.Error("TUI sources map not initialized")
	}
}

func TestGetOrCreateSource(t *testing.T) {
	t.Skip("Skipping TUI tests - requires terminal")
	ctx := context.Background()
	tui := NewTUILogger(ctx)

	podName := "test-pod"
	containerName := "test-container"

	source := tui.GetOrCreateSource(podName, containerName)
	if source == nil {
		t.Fatal("GetOrCreateSource returned nil")
	}

	if source.PodName != podName {
		t.Errorf("Expected pod name %s, got %s", podName, source.PodName)
	}

	if source.Container != containerName {
		t.Errorf("Expected container name %s, got %s", containerName, source.Container)
	}

	// Test that getting the same source returns the same object
	source2 := tui.GetOrCreateSource(podName, containerName)
	if source != source2 {
		t.Error("GetOrCreateSource should return the same source for the same pod/container")
	}
}

func TestWriteLog(t *testing.T) {
	t.Skip("Skipping TUI tests - requires terminal")
	ctx := context.Background()
	tui := NewTUILogger(ctx)

	podName := "test-pod"
	containerName := "test-container"
	logLine := "Test log line"

	tui.WriteLog(podName, containerName, logLine)

	// Give some time for goroutines to process
	time.Sleep(100 * time.Millisecond)

	source := tui.GetOrCreateSource(podName, containerName)
	lines := source.GetLines()

	if len(lines) != 1 {
		t.Fatalf("Expected 1 log line, got %d", len(lines))
	}

	if lines[0] != logLine {
		t.Errorf("Expected log line %q, got %q", logLine, lines[0])
	}
}

func TestLogSourceAddLine(t *testing.T) {
	source := &LogSource{
		Name:      "test-source",
		PodName:   "test-pod",
		Container: "test-container",
		Lines:     make([]string, 0),
	}

	// Test adding single line
	source.AddLine("Line 1")
	lines := source.GetLines()
	if len(lines) != 1 || lines[0] != "Line 1" {
		t.Error("Failed to add single line")
	}

	// Test adding multiple lines
	source.AddLine("Line 2")
	source.AddLine("Line 3")
	lines = source.GetLines()
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Test empty line is ignored
	source.AddLine("")
	lines = source.GetLines()
	if len(lines) != 3 {
		t.Error("Empty line should not be added")
	}

	// Test line with only newline is ignored
	source.AddLine("\n")
	lines = source.GetLines()
	if len(lines) != 3 {
		t.Error("Line with only newline should not be added")
	}
}

func TestTUIWriter(t *testing.T) {
	t.Skip("Skipping TUI tests - requires terminal")
	ctx := context.Background()
	tui := NewTUILogger(ctx)

	podName := "test-pod"
	containerName := "test-container"

	writer := NewTUIWriter(tui, podName, containerName)
	if writer == nil {
		t.Fatal("NewTUIWriter returned nil")
	}

	testData := []byte("Test log message\n")
	n, err := writer.Write(testData)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Give some time for processing
	time.Sleep(100 * time.Millisecond)

	source := tui.GetOrCreateSource(podName, containerName)
	lines := source.GetLines()

	if len(lines) != 1 {
		t.Fatalf("Expected 1 log line, got %d", len(lines))
	}

	if lines[0] != "Test log message" {
		t.Errorf("Expected %q, got %q", "Test log message", lines[0])
	}
}

func TestMultipleSources(t *testing.T) {
	t.Skip("Skipping TUI tests - requires terminal")
	ctx := context.Background()
	tui := NewTUILogger(ctx)

	// Create multiple sources
	pods := []string{"pod1", "pod2", "pod3"}
	for _, pod := range pods {
		tui.WriteLog(pod, "container", "Log from "+pod)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify all sources were created
	if len(tui.sources) != 3 {
		t.Errorf("Expected 3 sources, got %d", len(tui.sources))
	}

	// Verify each source has its own log
	for _, pod := range pods {
		source := tui.GetOrCreateSource(pod, "container")
		lines := source.GetLines()
		if len(lines) != 1 {
			t.Errorf("Expected 1 line for %s, got %d", pod, len(lines))
		}
	}
}
