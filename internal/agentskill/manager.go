package agentskill

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/UsingCoding/rocketchat-cli/skills"
)

type Target struct {
	Name        string
	GlobalRoot  string
	ProjectRoot string
}

type Status struct {
	Target string `json:"target"`
	Path   string `json:"path"`
	Exists bool   `json:"installed"`
}

func Targets() ([]Target, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return []Target{
		{"claude", filepath.Join(home, ".claude", "skills"), filepath.Join(".claude", "skills")},
		{"codex", filepath.Join(home, ".codex", "skills"), filepath.Join(".codex", "skills")},
		{"cursor", filepath.Join(home, ".cursor", "skills"), filepath.Join(".cursor", "skills")},
		{"opencode", filepath.Join(home, ".config", "opencode", "skills"), filepath.Join(".opencode", "skills")},
		{"agents", filepath.Join(home, ".agents", "skills"), filepath.Join(".agents", "skills")},
	}, nil
}

func List(project bool) ([]Status, error) {
	targets, err := scopedTargets(project)
	if err != nil {
		return nil, err
	}
	statuses := make([]Status, 0, len(targets))
	for _, target := range targets {
		path := filepath.Join(target.GlobalRoot, "rocketchat-cli")
		_, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		statuses = append(statuses, Status{Target: target.Name, Path: path, Exists: err == nil})
	}
	return statuses, nil
}

func Install(project bool, selected string) ([]string, error) {
	targets, err := scopedTargets(project)
	if err != nil {
		return nil, err
	}
	chosen := make([]Target, 0, len(targets))
	for _, target := range targets {
		if selected != "" {
			if target.Name == selected {
				chosen = append(chosen, target)
			}
			continue
		}
		_, err := os.Stat(filepath.Dir(target.GlobalRoot))
		if err == nil {
			chosen = append(chosen, target)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if len(chosen) == 0 {
		if selected != "" {
			return nil, fmt.Errorf("unknown skill target %q", selected)
		}
		chosen = append(chosen, targets[len(targets)-1])
	}
	paths := make([]string, 0, len(chosen))
	for _, target := range chosen {
		destination := filepath.Join(target.GlobalRoot, "rocketchat-cli")
		if err := os.RemoveAll(destination); err != nil {
			return nil, err
		}
		if err := copySkill(destination); err != nil {
			return nil, err
		}
		paths = append(paths, destination)
	}
	sort.Strings(paths)
	return paths, nil
}

func Remove(project bool, selected string) ([]string, error) {
	statuses, err := List(project)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if !status.Exists || (selected != "" && selected != status.Target) {
			continue
		}
		if err := os.RemoveAll(status.Path); err != nil {
			return nil, err
		}
		paths = append(paths, status.Path)
	}
	return paths, nil
}

func scopedTargets(project bool) ([]Target, error) {
	targets, err := Targets()
	if err != nil {
		return nil, err
	}
	if project {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		for i := range targets {
			targets[i].GlobalRoot = filepath.Join(cwd, targets[i].ProjectRoot)
		}
	}
	return targets, nil
}

func copySkill(destination string) error {
	source, err := fs.Sub(skills.FS, "rocketchat-cli")
	if err != nil {
		return err
	}
	return fs.WalkDir(source, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		target := filepath.Join(destination, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
