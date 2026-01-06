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

package logger

import (
	"context"
	"fmt"
	"io"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log/stream"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log/tui"
	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

// StartTUI initializes and starts the TUI logger
func (a *LogAggregator) StartTUI(ctx context.Context) error {
	tuiLogger := tui.NewTUILogger(ctx)
	a.tuiLogger = tuiLogger

	// Start TUI in a goroutine
	go func() {
		if err := tuiLogger.Start(); err != nil {
			olog.Entry(ctx).Warnf("TUI error: %v", err)
		}
	}()

	return nil
}

// streamContainerLogsTUI streams logs for a container to the TUI
func (a *LogAggregator) streamContainerLogsTUI(ctx context.Context, pod *v1.Pod, container v1.ContainerStatus) {
	olog.Entry(ctx).Infof("Streaming logs from pod: %s container: %s", pod.Name, container.Name)

	sinceSeconds := fmt.Sprintf("--since=%ds", sinceSeconds(time.Since(a.sinceTime)))

	tr, tw := io.Pipe()
	go func() {
		if err := a.kubectlcli.Run(ctx, nil, tw, "logs", sinceSeconds, "-f", pod.Name, "-c", container.Name, "--namespace", pod.Namespace); err != nil {
			if ctx.Err() != context.Canceled {
				olog.Entry(ctx).Warn(err)
			}
		}
		_ = tw.Close()
	}()

	// Get TUI logger
	tuiLogger, ok := a.tuiLogger.(*tui.TUILogger)
	if !ok || tuiLogger == nil {
		olog.Entry(ctx).Error("TUI logger not initialized")
		return
	}

	// Create TUI writer for this container
	tuiWriter := tui.NewTUIWriter(tuiLogger, pod.Name, container.Name)

	if err := stream.StreamRequest(ctx, tuiWriter, a.formatter(*pod, container, a.IsMuted), tr); err != nil {
		olog.Entry(ctx).Errorf("streaming request %s", err)
	}
}
