package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/UsingCoding/rocketchat-cli/internal/agentskill"
)

func TestSkillProjectLifecycle(t *testing.T) {
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
	project, err = os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(project, ".agents/skills/rocketchat-cli")
	run := func(args ...string) string {
		t.Helper()
		// Override the helper's synthetic credentials; skill operations need none.
		args = append([]string{"--user-id=", "--token="}, args...)
		out, err := executeCommand(t, "", "", args...)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	checkStatus := func(installed bool) {
		t.Helper()
		var statuses []agentskill.Status
		if err := json.Unmarshal([]byte(run("skill", "list", "--project", "--json")), &statuses); err != nil {
			t.Fatal(err)
		}
		if len(statuses) != 5 {
			t.Fatalf("statuses = %#v", statuses)
		}
		for _, status := range statuses {
			if status.Target == "agents" {
				if status.Path != destination || status.Exists != installed {
					t.Fatalf("status = %#v", status)
				}
				return
			}
		}
		t.Fatal("agents status missing")
	}
	checkStatus(false)
	for _, step := range []struct{ verb, action string }{{"install", "Installed"}, {"update", "Updated"}, {"remove", "Removed"}} {
		out := run("skill", step.verb, "--project", "--target", "agents")
		if out != step.action+" rocketchat-cli skill: "+destination+"\n" {
			t.Fatalf("output = %q", out)
		}
		checkStatus(step.verb != "remove")
	}
	out, err := executeCommand(t, "", "", "skill", "list", "--project", "--json", "--raw")
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("conflicting output flags: %q, %v", out, err)
	}
}
