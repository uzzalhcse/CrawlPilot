package plugin

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// BuildStatus represents the status of a build
type BuildStatus string

const (
	BuildStatusQueued   BuildStatus = "queued"
	BuildStatusBuilding BuildStatus = "building"
	BuildStatusSuccess  BuildStatus = "success"
	BuildStatusFailed   BuildStatus = "failed"
)

// BuildJob represents a build job
type BuildJob struct {
	ID          string      `json:"id"`
	PluginSlug  string      `json:"plugin_slug"`
	Status      BuildStatus `json:"status"`
	Log         string      `json:"log"`
	Artifact    string      `json:"artifact"`
	CreatedAt   time.Time   `json:"created_at"`
	CompletedAt *time.Time  `json:"completed_at"`
}

// Builder handles plugin building
type Builder struct {
	SourcePath string
	OutputPath string
	jobs       map[string]*BuildJob
	mu         sync.RWMutex
}

// NewBuilder creates a new Builder
func NewBuilder(sourcePath, outputPath string) *Builder {
	return &Builder{
		SourcePath: sourcePath,
		OutputPath: outputPath,
		jobs:       make(map[string]*BuildJob),
	}
}

// Build triggers a build for a plugin
func (b *Builder) Build(pluginSlug string) (string, error) {
	jobID := uuid.New().String()
	job := &BuildJob{
		ID:         jobID,
		PluginSlug: pluginSlug,
		Status:     BuildStatusQueued,
		CreatedAt:  time.Now(),
	}

	b.mu.Lock()
	b.jobs[jobID] = job
	b.mu.Unlock()

	go b.runBuild(job)

	return jobID, nil
}

// GetBuildJob returns a build job by ID
func (b *Builder) GetBuildJob(jobID string) (*BuildJob, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	job, ok := b.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("build job not found: %s", jobID)
	}
	return job, nil
}

func (b *Builder) runBuild(job *BuildJob) {
	// Update status to building
	b.mu.Lock()
	job.Status = BuildStatusBuilding
	b.mu.Unlock()

	pluginDir := filepath.Join(b.SourcePath, job.PluginSlug)

	// Ensure output directory exists
	if err := exec.Command("mkdir", "-p", b.OutputPath).Run(); err != nil {
		b.updateJobStatus(job, BuildStatusFailed, fmt.Sprintf("Failed to create output directory: %v", err))
		return
	}

	// Run go mod tidy first to ensure dependencies are downloaded
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = pluginDir

	var tidyStdout, tidyStderr bytes.Buffer
	tidyCmd.Stdout = &tidyStdout
	tidyCmd.Stderr = &tidyStderr

	if err := tidyCmd.Run(); err != nil {
		errorMsg := fmt.Sprintf("Failed to run go mod tidy: %v\nStdout: %s\nStderr: %s",
			err, tidyStdout.String(), tidyStderr.String())
		b.updateJobStatus(job, BuildStatusFailed, errorMsg)
		return
	}

	outputFile, err := filepath.Abs(filepath.Join(b.OutputPath, job.PluginSlug+".so"))
	if err != nil {
		b.updateJobStatus(job, BuildStatusFailed, fmt.Sprintf("Failed to resolve output path: %v", err))
		return
	}

	// Construct command: go build -buildmode=plugin -o <output> .
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", outputFile, ".")
	cmd.Dir = pluginDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	logOutput := stdout.String() + "\n" + stderr.String()

	if err != nil {
		b.updateJobStatus(job, BuildStatusFailed, logOutput+fmt.Sprintf("\nBuild failed: %v", err))
	} else {
		// Update job with success and artifact path
		b.mu.Lock()
		now := time.Now()
		job.Status = BuildStatusSuccess
		job.CompletedAt = &now
		job.Log = logOutput
		job.Artifact = outputFile
		b.mu.Unlock()
	}
}

func (b *Builder) updateJobStatus(job *BuildJob, status BuildStatus, log string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	job.Status = status
	job.CompletedAt = &now
	job.Log = log
}
