package output

import "testing"

func TestSanitizeTerminal(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text passthrough",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "ANSI color codes stripped",
			in:   "\x1b[31mred\x1b[0m",
			want: "red",
		},
		{
			name: "ANSI bold stripped",
			in:   "\x1b[1mbold\x1b[22m",
			want: "bold",
		},
		{
			name: "ANSI CSI cursor movement stripped",
			in:   "\x1b[2Aup two lines",
			want: "up two lines",
		},
		{
			name: "ANSI CSI erase display stripped",
			in:   "\x1b[2Jcleared",
			want: "cleared",
		},
		{
			name: "ANSI SGR with multiple params stripped",
			in:   "\x1b[38;5;196mred256\x1b[0m",
			want: "red256",
		},
		{
			name: "ANSI OSC sequence stripped",
			in:   "\x1b]0;window title\x07rest",
			want: "rest",
		},
		{
			name: "control characters removed",
			in:   "hello\x00\x01\x02\x7fworld",
			want: "helloworld",
		},
		{
			name: "newlines preserved",
			in:   "line1\nline2\n",
			want: "line1\nline2\n",
		},
		{
			name: "tabs preserved",
			in:   "col1\tcol2\tcol3",
			want: "col1\tcol2\tcol3",
		},
		{
			name: "carriage returns preserved",
			in:   "hello\rworld",
			want: "hello\rworld",
		},
		{
			name: "unicode characters preserved",
			in:   "emoji: \U0001F600 CJK: \u4E16\u754C",
			want: "emoji: \U0001F600 CJK: \u4E16\u754C",
		},
		{
			name: "empty string returns empty",
			in:   "",
			want: "",
		},
		{
			name: "only ANSI codes returns empty",
			in:   "\x1b[31m\x1b[0m",
			want: "",
		},
		{
			name: "mixed ANSI and control characters",
			in:   "\x1b[32mgreen\x00text\x1b[0m\x01end",
			want: "greentextend",
		},
		{
			name: "preserves whitespace within text",
			in:   "  spaced  out  ",
			want: "  spaced  out  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeTerminal(tt.in)
			if got != tt.want {
				t.Errorf("SanitizeTerminal(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
