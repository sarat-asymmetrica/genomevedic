package profiling

import (
	"fmt"
	"strings"
	"time"
)

// FrameProfiler measures performance of individual frame stages
type FrameProfiler struct {
	stages     map[string]time.Duration
	stageCounts map[string]int64
	stageStart time.Time
	currentStage string
	frameStart time.Time
	frameCount int64
}

// NewFrameProfiler creates a new frame profiler
func NewFrameProfiler() *FrameProfiler {
	return &FrameProfiler{
		stages:     make(map[string]time.Duration),
		stageCounts: make(map[string]int64),
		frameCount: 0,
	}
}

// StartFrame begins timing a new frame
func (fp *FrameProfiler) StartFrame() {
	fp.frameStart = time.Now()
	fp.frameCount++
}

// EndFrame completes the current frame timing
func (fp *FrameProfiler) EndFrame() time.Duration {
	return time.Since(fp.frameStart)
}

// StartStage begins timing a frame stage
func (fp *FrameProfiler) StartStage(name string) {
	fp.stageStart = time.Now()
	fp.currentStage = name
}

// EndStage completes timing the current stage
func (fp *FrameProfiler) EndStage() {
	if fp.currentStage == "" {
		return
	}

	duration := time.Since(fp.stageStart)
	fp.stages[fp.currentStage] += duration
	fp.stageCounts[fp.currentStage]++
	fp.currentStage = ""
}

// GetFrameTime returns total frame time
func (fp *FrameProfiler) GetFrameTime() time.Duration {
	total := time.Duration(0)
	for _, duration := range fp.stages {
		total += duration
	}
	return total
}

// GetFPS calculates average FPS
func (fp *FrameProfiler) GetFPS() float64 {
	avgFrameTime := fp.GetAverageFrameTime()
	if avgFrameTime == 0 {
		return 0
	}
	return 1000.0 / avgFrameTime.Seconds() / 1000.0
}

// GetAverageFrameTime returns average frame time across all frames
func (fp *FrameProfiler) GetAverageFrameTime() time.Duration {
	if fp.frameCount == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, duration := range fp.stages {
		total += duration
	}

	return total / time.Duration(fp.frameCount)
}

// GetStageTime returns total time spent in a stage
func (fp *FrameProfiler) GetStageTime(stageName string) time.Duration {
	return fp.stages[stageName]
}

// GetStageAverage returns average time per stage execution
func (fp *FrameProfiler) GetStageAverage(stageName string) time.Duration {
	count := fp.stageCounts[stageName]
	if count == 0 {
		return 0
	}
	return fp.stages[stageName] / time.Duration(count)
}

// Report generates a performance report
func (fp *FrameProfiler) Report() string {
	var sb strings.Builder

	sb.WriteString("Frame Performance Report\n")
	sb.WriteString("========================\n\n")

	avgFrameTime := fp.GetAverageFrameTime()
	fps := fp.GetFPS()

	sb.WriteString(fmt.Sprintf("Frames:          %d\n", fp.frameCount))
	sb.WriteString(fmt.Sprintf("Avg frame time:  %.2fms\n", avgFrameTime.Seconds()*1000))
	sb.WriteString(fmt.Sprintf("FPS:             %.1f\n\n", fps))

	sb.WriteString("Stage Breakdown:\n")
	sb.WriteString("----------------\n")

	// Sort stages by time (descending)
	stages := make([]string, 0, len(fp.stages))
	for stage := range fp.stages {
		stages = append(stages, stage)
	}

	// Simple bubble sort
	for i := 0; i < len(stages); i++ {
		for j := i + 1; j < len(stages); j++ {
			if fp.stages[stages[i]] < fp.stages[stages[j]] {
				stages[i], stages[j] = stages[j], stages[i]
			}
		}
	}

	totalTime := time.Duration(0)
	for _, duration := range fp.stages {
		totalTime += duration
	}

	for _, stage := range stages {
		avgTime := fp.GetStageAverage(stage)
		percentage := float64(fp.stages[stage]) / float64(totalTime) * 100

		sb.WriteString(fmt.Sprintf("  %-20s %.2fms (%.1f%%)\n",
			stage+":", avgTime.Seconds()*1000, percentage))
	}

	return sb.String()
}

// Reset clears all profiling data
func (fp *FrameProfiler) Reset() {
	fp.stages = make(map[string]time.Duration)
	fp.stageCounts = make(map[string]int64)
	fp.frameCount = 0
}
