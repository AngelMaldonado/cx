package project

import (
	"fmt"
	"os"
	"path/filepath"
)

type ScaffoldResult struct {
	Created []string
	Skipped []string
}

func ScaffoldDocs(rootDir string) (*ScaffoldResult, error) {
	result := &ScaffoldResult{}

	dirs := []string{
		"docs",
		"docs/architecture",
		"docs/architecture/diagrams",
		"docs/specs",
		"docs/memory",
		"docs/memory/observations",
		"docs/memory/decisions",
		"docs/memory/sessions",
		"docs/changes",
		"docs/masterfiles",
		"docs/archive",
	}

	for _, dir := range dirs {
		full := filepath.Join(rootDir, dir)
		if err := os.MkdirAll(full, 0o755); err != nil {
			return nil, fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	templates := map[string]func() string{
		"docs/overview.md":          OverviewTemplate,
		"docs/architecture/index.md": ArchitectureTemplate,
		"docs/specs/index.md":       SpecsIndexTemplate,
	}

	for relPath, tmplFn := range templates {
		full := filepath.Join(rootDir, relPath)
		if _, err := os.Stat(full); err == nil {
			result.Skipped = append(result.Skipped, relPath)
			continue
		}
		if err := atomicWrite(full, []byte(tmplFn())); err != nil {
			return nil, fmt.Errorf("writing %s: %w", relPath, err)
		}
		result.Created = append(result.Created, relPath)
	}

	return result, nil
}

func ScaffoldCXCache(rootDir string) (bool, error) {
	cxDir := filepath.Join(rootDir, ".cx")
	created := false

	if _, err := os.Stat(cxDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cxDir, 0o755); err != nil {
			return false, fmt.Errorf("creating .cx/: %w", err)
		}
		created = true
	}

	gitignore := filepath.Join(cxDir, ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		if err := atomicWrite(gitignore, []byte("*\n")); err != nil {
			return created, fmt.Errorf("writing .cx/.gitignore: %w", err)
		}
	}

	return created, nil
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func OverviewTemplate() string {
	return `# Project Overview

## What is this project?
<!-- Describe the project in 1-2 paragraphs -->

## Key decisions
<!-- Link to important decisions in docs/memory/decisions/ -->

## Quick links
- [Architecture](architecture/index.md)
- [Specs](specs/index.md)
`
}

func ArchitectureTemplate() string {
	return `# Architecture

## System overview
<!-- High-level architecture description -->

## Components
<!-- List major components and their responsibilities -->

## Diagrams
<!-- Reference diagrams in diagrams/ subdirectory -->
`
}

func SpecsIndexTemplate() string {
	return `# Specifications

## Active specs
<!-- List active specification documents -->

## Draft specs
<!-- List work-in-progress specs -->
`
}
