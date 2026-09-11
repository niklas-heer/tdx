package util

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestClipboardPlatforms(t *testing.T) {
	for _, tt := range []struct{ platform, display, copy, paste string }{{"darwin", "", "pbcopy", "pbpaste"}, {"linux", "WAYLAND_DISPLAY", "wl-copy", "wl-paste"}, {"linux", "DISPLAY", "xclip", "xclip"}, {"windows", "", "powershell.exe", "powershell.exe"}} {
		t.Run(tt.platform+tt.display, func(t *testing.T) {
			var calls []string
			var inputs []string
			c := SystemClipboard{Platform: tt.platform, Getenv: func(key string) string {
				if key == tt.display {
					return "yes"
				}
				return ""
			}, LookPath: func(s string) (string, error) { return s, nil }, Run: func(_ context.Context, name string, args []string, input string) (string, error) {
				calls = append(calls, name)
				inputs = append(inputs, input)
				return "café\n", nil
			}}
			if err := c.Copy("世界"); err != nil {
				t.Fatal(err)
			}
			text, err := c.Paste()
			if err != nil || text != "café\n" {
				t.Fatal(text, err)
			}
			if !reflect.DeepEqual(calls, []string{tt.copy, tt.paste}) || inputs[0] != "世界" {
				t.Fatalf("wrong command or input: %v %v", calls, inputs)
			}
		})
	}
}
func TestClipboardUnavailableFailureAndTimeout(t *testing.T) {
	c := SystemClipboard{Platform: "linux", Getenv: func(string) string { return "" }}
	if err := c.Copy("test"); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatal(err)
	}
	c = SystemClipboard{Platform: "darwin", LookPath: func(s string) (string, error) { return s, nil }, Run: func(context.Context, string, []string, string) (string, error) { return "", errors.New("failed") }}
	if _, err := c.Paste(); err == nil {
		t.Fatal("paste failure hidden")
	}
	c.Timeout = time.Millisecond
	c.Run = func(ctx context.Context, _ string, _ []string, _ string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}
	if err := c.Copy("test"); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatal(err)
	}
}
func TestClipboardXselFallback(t *testing.T) {
	c := SystemClipboard{Platform: "linux", Getenv: func(s string) string {
		if s == "DISPLAY" {
			return ":0"
		}
		return ""
	}, LookPath: func(s string) (string, error) {
		if s == "xsel" {
			return s, nil
		}
		return "", errors.New("missing")
	}}
	name, args, err := c.command(false)
	if err != nil || name != "xsel" || !reflect.DeepEqual(args, []string{"--clipboard", "--output"}) {
		t.Fatal(name, args, err)
	}
}
