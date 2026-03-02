package text

import "testing"

func TestDedent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no indent",
			input: "hello\nworld",
			want:  "hello\nworld",
		},
		{
			name:  "uniform indent",
			input: "\t\tfoo\n\t\tbar\n\t\tbaz",
			want:  "foo\nbar\nbaz",
		},
		{
			name:  "mixed indent levels",
			input: "    line1\n      line2\n    line3",
			want:  "line1\n  line2\nline3",
		},
		{
			name:  "leading blank lines stripped",
			input: "\n\n\t\thello\n\t\tworld",
			want:  "hello\nworld",
		},
		{
			name:  "trailing blank lines stripped",
			input: "\t\thello\n\t\tworld\n\n",
			want:  "hello\nworld",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   \n   \n   ",
			want:  "   \n   \n   ",
		},
		{
			name:  "single line",
			input: "    hello",
			want:  "hello",
		},
		{
			name: "backtick heredoc style",
			input: `
				Chains three vulnerabilities: unauthenticated access
				to install.php, SQL injection in LDAP config update,
				and command injection via the 'dot' binary path.
			`,
			want: "Chains three vulnerabilities: unauthenticated access\n" +
				"to install.php, SQL injection in LDAP config update,\n" +
				"and command injection via the 'dot' binary path.",
		},
		{
			name:  "preserves empty lines between content",
			input: "\t\tline1\n\n\t\tline2",
			want:  "line1\n\nline2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Dedent(tt.input); got != tt.want {
				t.Errorf("Dedent() = %q, want %q", got, tt.want)
			}
		})
	}
}
