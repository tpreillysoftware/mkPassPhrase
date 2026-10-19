/******************************************************************************
Copyright (c) 2026, Timothy Reilly
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of mkPassPhrase nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*******************************************************************************/

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "mkPassPhrase-integration-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	binaryPath = filepath.Join(tempDir, "mkPassPhrase")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build mkPassPhrase binary: %v\nOutput:\n%s\n", err, string(out))
		os.Exit(1)
	}

	exitCode := m.Run()
	os.Exit(exitCode)
}

type runResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func runBinary(t *testing.T, args ...string) runResult {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("unexpected execution error running binary with args %v: %v", args, err)
		}
	}

	return runResult{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: exitCode,
	}
}

func TestBinary_DefaultWordMap(t *testing.T) {
	tests := []struct {
		name         string
		diceKey      string
		expectedWord string
	}{
		{
			name:         "README example key 53728",
			diceKey:      "53728",
			expectedWord: "dividend",
		},
		{
			name:         "first word key 11111",
			diceKey:      "11111",
			expectedWord: "abed",
		},
		{
			name:         "second word key 11112",
			diceKey:      "11112",
			expectedWord: "abel",
		},
		{
			name:         "middle boundary key 11128",
			diceKey:      "11128",
			expectedWord: "adze",
		},
		{
			name:         "last word key 88888",
			diceKey:      "88888",
			expectedWord: "experimentation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := runBinary(t, "-d", tt.diceKey)

			if res.exitCode != 0 {
				t.Fatalf("expected exit code 0, got %d; stderr: %q", res.exitCode, res.stderr)
			}
			expectedOutput := tt.expectedWord + "\n"
			if res.stdout != expectedOutput {
				t.Errorf("stdout = %q, want %q", res.stdout, expectedOutput)
			}
			if res.stderr != "" {
				t.Errorf("expected empty stderr, got %q", res.stderr)
			}
		})
	}
}

func TestBinary_CustomWordList(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("valid custom word list", func(t *testing.T) {
		customListPath := filepath.Join(tempDir, "custom_words.txt")
		content := "alpha\nbravo\ncharlie\ndelta\necho\n"
		if err := os.WriteFile(customListPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write custom wordlist: %v", err)
		}

		tests := []struct {
			diceKey      string
			expectedWord string
		}{
			{diceKey: "11111", expectedWord: "alpha"},
			{diceKey: "11112", expectedWord: "bravo"},
			{diceKey: "11113", expectedWord: "charlie"},
			{diceKey: "11114", expectedWord: "delta"},
			{diceKey: "11115", expectedWord: "echo"},
		}

		for _, tt := range tests {
			res := runBinary(t, "-w", customListPath, "-d", tt.diceKey)
			if res.exitCode != 0 {
				t.Fatalf("expected exit code 0 for key %s, got %d; stderr: %q", tt.diceKey, res.exitCode, res.stderr)
			}
			expectedOutput := tt.expectedWord + "\n"
			if res.stdout != expectedOutput {
				t.Errorf("stdout = %q, want %q for key %s", res.stdout, expectedOutput, tt.diceKey)
			}
		}
	})

	t.Run("custom word list full capacity 32768 entries", func(t *testing.T) {
		fullListPath := filepath.Join(tempDir, "full_words.txt")
		var builder strings.Builder
		for i := 0; i < wordCount; i++ {
			fmt.Fprintf(&builder, "word_%d\n", i)
		}
		if err := os.WriteFile(fullListPath, []byte(builder.String()), 0644); err != nil {
			t.Fatalf("failed to write full wordlist: %v", err)
		}

		// First entry (index 0 -> key "11111")
		resFirst := runBinary(t, "-w", fullListPath, "-d", "11111")
		if resFirst.exitCode != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %q", resFirst.exitCode, resFirst.stderr)
		}
		if resFirst.stdout != "word_0\n" {
			t.Errorf("stdout = %q, want %q", resFirst.stdout, "word_0\n")
		}

		// Last entry (index 32767 -> key "88888")
		resLast := runBinary(t, "-w", fullListPath, "-d", "88888")
		if resLast.exitCode != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %q", resLast.exitCode, resLast.stderr)
		}
		if resLast.stdout != "word_32767\n" {
			t.Errorf("stdout = %q, want %q", resLast.stdout, "word_32767\n")
		}
	})

	t.Run("key not found in custom word list", func(t *testing.T) {
		shortListPath := filepath.Join(tempDir, "short_words.txt")
		content := "only_one_word\n"
		if err := os.WriteFile(shortListPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write short wordlist: %v", err)
		}

		// 11112 corresponds to index 1 which doesn't exist
		res := runBinary(t, "-w", shortListPath, "-d", "11112")
		if res.exitCode != 1 {
			t.Fatalf("expected exit code 1, got %d; stderr: %q", res.exitCode, res.stderr)
		}
		if !strings.Contains(res.stderr, `no word found for dice key "11112"`) {
			t.Errorf("stderr does not contain expected error message; stderr = %q", res.stderr)
		}
	})

	t.Run("non-existent custom word list file", func(t *testing.T) {
		res := runBinary(t, "-w", filepath.Join(tempDir, "does_not_exist.txt"), "-d", "11111")
		if res.exitCode != 1 {
			t.Fatalf("expected exit code 1, got %d; stderr: %q", res.exitCode, res.stderr)
		}
		if !strings.Contains(res.stderr, "failed to import word list:") {
			t.Errorf("stderr does not contain 'failed to import word list:'; stderr = %q", res.stderr)
		}
	})

	t.Run("custom word list exceeding 32768 entries", func(t *testing.T) {
		overCapacityPath := filepath.Join(tempDir, "over_capacity_words.txt")
		var builder strings.Builder
		for i := 0; i < wordCount+1; i++ {
			fmt.Fprintf(&builder, "extra_word_%d\n", i)
		}
		if err := os.WriteFile(overCapacityPath, []byte(builder.String()), 0644); err != nil {
			t.Fatalf("failed to write over capacity wordlist: %v", err)
		}

		res := runBinary(t, "-w", overCapacityPath, "-d", "11111")
		if res.exitCode != 1 {
			t.Fatalf("expected exit code 1, got %d; stderr: %q", res.exitCode, res.stderr)
		}
		if !strings.Contains(res.stderr, "word list contains more than 32,768 entries") {
			t.Errorf("stderr does not contain overflow error message; stderr = %q", res.stderr)
		}
	})
}

func TestBinary_UsageAndFlagErrors(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedExitCode  int
		expectedStderrSub string
	}{
		{
			name:              "no arguments provided",
			args:              []string{},
			expectedExitCode:  2,
			expectedStderrSub: "-d must be exactly 5 digits, each in the range 1-8",
		},
		{
			name:              "empty dice key flag",
			args:              []string{"-d", ""},
			expectedExitCode:  2,
			expectedStderrSub: "-d must be exactly 5 digits, each in the range 1-8",
		},
		{
			name:              "dice key too short (4 digits)",
			args:              []string{"-d", "1234"},
			expectedExitCode:  2,
			expectedStderrSub: "-d must be exactly 5 digits, each in the range 1-8",
		},
		{
			name:              "dice key too long (6 digits)",
			args:              []string{"-d", "123456"},
			expectedExitCode:  2,
			expectedStderrSub: "-d must be exactly 5 digits, each in the range 1-8",
		},
		{
			name:              "dice key contains digit 0",
			args:              []string{"-d", "01234"},
			expectedExitCode:  2,
			expectedStderrSub: "-d must be exactly 5 digits, each in the range 1-8",
		},
		{
			name:              "dice key contains digit 9",
			args:              []string{"-d", "12349"},
			expectedExitCode:  2,
			expectedStderrSub: "-d must be exactly 5 digits, each in the range 1-8",
		},
		{
			name:              "dice key contains letters",
			args:              []string{"-d", "abcde"},
			expectedExitCode:  2,
			expectedStderrSub: "-d must be exactly 5 digits, each in the range 1-8",
		},
		{
			name:              "dice key contains special symbols",
			args:              []string{"-d", "1234!"},
			expectedExitCode:  2,
			expectedStderrSub: "-d must be exactly 5 digits, each in the range 1-8",
		},
		{
			name:              "unexpected positional arguments",
			args:              []string{"-d", "11111", "extra_arg"},
			expectedExitCode:  2,
			expectedStderrSub: "unexpected positional argument(s)",
		},
		{
			name:              "unknown flag",
			args:              []string{"-unknownFlag"},
			expectedExitCode:  2,
			expectedStderrSub: "flag provided but not defined: -unknownFlag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := runBinary(t, tt.args...)

			if res.exitCode != tt.expectedExitCode {
				t.Fatalf("expected exit code %d, got %d; stderr: %q", tt.expectedExitCode, res.exitCode, res.stderr)
			}
			if !strings.Contains(res.stderr, tt.expectedStderrSub) {
				t.Errorf("stderr does not contain expected substring %q; got stderr: %q", tt.expectedStderrSub, res.stderr)
			}
			if !strings.Contains(res.stderr, "Usage:") {
				t.Errorf("stderr does not contain usage information; got stderr: %q", res.stderr)
			}
		})
	}
}

func TestBinary_HelpFlag(t *testing.T) {
	for _, flagArg := range []string{"-h", "-help", "--help"} {
		t.Run(flagArg, func(t *testing.T) {
			res := runBinary(t, flagArg)

			if res.exitCode != 0 {
				t.Fatalf("expected exit code 0 for %s, got %d", flagArg, res.exitCode)
			}
			if !strings.Contains(res.stderr, "Usage:") {
				t.Errorf("expected stderr to contain 'Usage:'; got: %q", res.stderr)
			}
			if !strings.Contains(res.stderr, "-w") || !strings.Contains(res.stderr, "-d") {
				t.Errorf("expected stderr to contain flag definitions; got: %q", res.stderr)
			}
		})
	}
}

func TestUnit_ValidDiceKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"11111", true},
		{"88888", true},
		{"53728", true},
		{"12345", true},
		{"", false},
		{"1234", false},
		{"123456", false},
		{"01234", false},
		{"12349", false},
		{"abcde", false},
		{"1234a", false},
		{" 1234", false},
		{"1234 ", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("key_%q", tt.key), func(t *testing.T) {
			got := validDiceKey(tt.key)
			if got != tt.valid {
				t.Errorf("validDiceKey(%q) = %v; want %v", tt.key, got, tt.valid)
			}
		})
	}
}

func TestUnit_KeyForIndex(t *testing.T) {
	tests := []struct {
		index       int
		expectedKey string
		expectErr   bool
	}{
		{index: 0, expectedKey: "11111", expectErr: false},
		{index: 1, expectedKey: "11112", expectErr: false},
		{index: 7, expectedKey: "11118", expectErr: false},
		{index: 8, expectedKey: "11121", expectErr: false},
		{index: wordCount - 1, expectedKey: "88888", expectErr: false},
		{index: -1, expectedKey: "", expectErr: true},
		{index: wordCount, expectedKey: "", expectErr: true},
		{index: wordCount + 100, expectedKey: "", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("index_%d", tt.index), func(t *testing.T) {
			key, err := keyForIndex(tt.index)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("keyForIndex(%d) expected error, got key %q", tt.index, key)
				}
			} else {
				if err != nil {
					t.Fatalf("keyForIndex(%d) unexpected error: %v", tt.index, err)
				}
				if key != tt.expectedKey {
					t.Errorf("keyForIndex(%d) = %q, want %q", tt.index, key, tt.expectedKey)
				}
			}
		})
	}
}

func TestUnit_ImportWordList(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("valid list", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "words.txt")
		content := "word1\nword2\nword3\n"
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		words, err := importWordList(filePath)
		if err != nil {
			t.Fatalf("unexpected error importing word list: %v", err)
		}
		if len(words) != 3 {
			t.Fatalf("expected 3 words, got %d", len(words))
		}
		if words["11111"] != "word1" || words["11112"] != "word2" || words["11113"] != "word3" {
			t.Errorf("unexpected words map: %v", words)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := importWordList(filepath.Join(tempDir, "nonexistent.txt"))
		if err == nil {
			t.Fatal("expected error importing nonexistent file, got nil")
		}
	})
}
