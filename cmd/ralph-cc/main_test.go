package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if version == "" {
		t.Error("version should not be empty")
	}
}

func TestDebugFlagsExist(t *testing.T) {
	var buf bytes.Buffer
	cmd := newRootCmd(&buf)

	expectedFlags := []string{"dparse", "dc", "dasm", "dclight", "dcminor", "drtl", "dltl", "dmach"}
	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag --%s to exist", flagName)
		}
	}
}

func TestDebugFlagsWarnAndExit(t *testing.T) {
	testCases := []struct {
		flagName string
		wantMsg  string
	}{
		{"dparse", "dparse"},
		{"dc", "dc"},
		{"dasm", "dasm"},
		{"dclight", "dclight"},
		{"dcminor", "dcminor"},
		{"drtl", "drtl"},
		{"dltl", "dltl"},
		{"dmach", "dmach"},
	}

	for _, tc := range testCases {
		t.Run(tc.flagName, func(t *testing.T) {
			// Reset all flags before each test
			resetDebugFlags()

			var buf bytes.Buffer
			cmd := newRootCmd(&buf)
			cmd.SetArgs([]string{"--" + tc.flagName, "test.c"})
			err := cmd.Execute()

			// Should return an error
			if err == nil {
				t.Errorf("expected error for flag --%s, got nil", tc.flagName)
			}
			if !errors.Is(err, ErrNotImplemented) {
				t.Errorf("expected ErrNotImplemented, got %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tc.wantMsg) {
				t.Errorf("expected output to contain %q, got %q", tc.wantMsg, output)
			}
			if !strings.Contains(output, "not yet implemented") {
				t.Errorf("expected output to contain 'not yet implemented', got %q", output)
			}
		})
	}
}

func TestNoDebugFlagsNoError(t *testing.T) {
	resetDebugFlags()

	var buf bytes.Buffer
	cmd := newRootCmd(&buf)
	cmd.SetArgs([]string{"test.c"})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("expected no error without debug flags, got %v", err)
	}
}

func resetDebugFlags() {
	dParse = false
	dC = false
	dAsm = false
	dClight = false
	dCminor = false
	dRTL = false
	dLTL = false
	dMach = false
}
