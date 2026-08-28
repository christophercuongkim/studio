package cli

import "testing"

func TestRunExitCodes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{"no args", nil, 2},
		{"help long", []string{"--help"}, 0},
		{"help word", []string{"help"}, 0},
		{"version", []string{"--version"}, 0},
		{"unknown command", []string{"bogus"}, 2},
		{"declared but unbuilt", []string{"ingest"}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Run("test", tt.args); got != tt.want {
				t.Errorf("Run(%q) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}

// A real command routes to its Run and maps a returned error to exit code 1.
func TestRunDispatchesToCommand(t *testing.T) {
	orig := commands
	t.Cleanup(func() { commands = orig })

	var gotArgs []string
	commands = []*Command{{
		Name: "demo",
		Run: func(args []string) error {
			gotArgs = args
			return nil
		},
	}}

	if got := Run("test", []string{"demo", "a", "b"}); got != 0 {
		t.Fatalf("Run(demo) = %d, want 0", got)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "a" || gotArgs[1] != "b" {
		t.Errorf("command received args %q, want [a b]", gotArgs)
	}
}
