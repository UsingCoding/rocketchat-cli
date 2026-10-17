package agentskill

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/UsingCoding/rocketchat-cli/skills"
)

func TestTargets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	got, err := Targets()
	if err != nil {
		t.Fatal(err)
	}
	want := []Target{
		{"claude", filepath.Join(home, ".claude/skills"), ".claude/skills"},
		{"codex", filepath.Join(home, ".codex/skills"), ".codex/skills"},
		{"cursor", filepath.Join(home, ".cursor/skills"), ".cursor/skills"},
		{"opencode", filepath.Join(home, ".config/opencode/skills"), ".opencode/skills"},
		{"agents", filepath.Join(home, ".agents/skills"), ".agents/skills"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("targets = %#v, want %#v", got, want)
	}
}

func TestProjectInstallReplaceRemove(t *testing.T) {
	project := skillProject(t)
	destination := filepath.Join(project, ".codex/skills/rocketchat-cli")
	paths, err := Install(true, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths, []string{destination}) {
		t.Fatalf("paths = %v", paths)
	}
	for _, asset := range []string{"SKILL.md", "references/commands.md", "references/workflows.md"} {
		want, err := skills.FS.ReadFile("rocketchat-cli/" + asset)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(destination, asset))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("asset differs: %s", asset)
		}
	}
	stale := filepath.Join(destination, "stale.txt")
	sibling := filepath.Join(project, ".codex/skills/other.txt")
	for _, path := range []string{stale, sibling} {
		if err := os.WriteFile(path, []byte("keep"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Install(true, "codex"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale file survived: %v", err)
	}
	if data, err := os.ReadFile(sibling); err != nil || string(data) != "keep" {
		t.Fatalf("sibling changed: %q %v", data, err)
	}
	paths, err = Remove(true, "codex")
	if err != nil || !reflect.DeepEqual(paths, []string{destination}) {
		t.Fatalf("remove = %v, %v", paths, err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("destination survived: %v", err)
	}
	paths, err = Remove(true, "codex")
	if err != nil || len(paths) != 0 {
		t.Fatalf("second remove = %v, %v", paths, err)
	}
}

func TestProjectSelection(t *testing.T) {
	project := skillProject(t)
	if _, err := Install(true, "unknown"); err == nil {
		t.Fatal("unknown target accepted")
	}
	paths, err := Install(true, "")
	if err != nil || !reflect.DeepEqual(paths, []string{filepath.Join(project, ".agents/skills/rocketchat-cli")}) {
		t.Fatalf("fallback = %v, %v", paths, err)
	}
	paths, err = Remove(true, "unknown")
	if err != nil || len(paths) != 0 {
		t.Fatalf("unknown removal = %v, %v", paths, err)
	}
	for _, root := range []string{".claude", ".codex"} {
		if err := os.Mkdir(filepath.Join(project, root), 0755); err != nil {
			t.Fatal(err)
		}
	}
	paths, err = Install(true, "")
	want := []string{filepath.Join(project, ".agents/skills/rocketchat-cli"), filepath.Join(project, ".claude/skills/rocketchat-cli"), filepath.Join(project, ".codex/skills/rocketchat-cli")}
	if err != nil || !reflect.DeepEqual(paths, want) {
		t.Fatalf("detected = %v, %v", paths, err)
	}
}

func skillProject(t *testing.T) string {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	})
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return current
}
