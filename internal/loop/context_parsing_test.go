package loop

import (
	"testing"
)

func TestParseTasksFromPlan_MultipleFormats(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "dash format",
			content: `
- [ ] First task
- [x] Second task done
- [ ] Third task
`,
			expected: []string{"[ ] First task", "[x] Second task done", "[ ] Third task"},
		},
		{
			name: "asterisk format",
			content: `
* [ ] First task
* [x] Second task done
* [ ] Third task
`,
			expected: []string{"[ ] First task", "[x] Second task done", "[ ] Third task"},
		},
		{
			name: "numbered format",
			content: `
1. [ ] First task
2. [x] Second task done
3. [ ] Third task
`,
			expected: []string{"[ ] First task", "[x] Second task done", "[ ] Third task"},
		},
		{
			name: "bare checkbox format",
			content: `
[ ] First task
[x] Second task done
[ ] Third task
`,
			expected: []string{"[ ] First task", "[x] Second task done", "[ ] Third task"},
		},
		{
			name: "mixed formats",
			content: `
- [ ] Dash task
* [ ] Asterisk task
1. [ ] Numbered task
[ ] Bare checkbox task
`,
			expected: []string{"[ ] Dash task", "[ ] Asterisk task", "[ ] Numbered task", "[ ] Bare checkbox task"},
		},
		{
			name: "uppercase X",
			content: `
- [X] Completed with uppercase
- [x] Completed with lowercase
`,
			expected: []string{"[x] Completed with uppercase", "[x] Completed with lowercase"},
		},
		{
			name:     "empty content",
			content:  "",
			expected: []string{},
		},
		{
			name: "no checkboxes",
			content: `
# Heading
Some regular text
- Bullet without checkbox
1. Numbered item without checkbox
`,
			expected: []string{},
		},
		{
			name: "double digit numbers",
			content: `
10. [ ] Task ten
11. [x] Task eleven
`,
			expected: []string{"[ ] Task ten", "[x] Task eleven"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := parseTasksFromPlan(tt.content, "test.md")
			if err != nil {
				t.Errorf("parseTasksFromPlan() error = %v, want nil", err)
			}

			if len(tasks) != len(tt.expected) {
				t.Errorf("parseTasksFromPlan() returned %d tasks, want %d", len(tasks), len(tt.expected))
				return
			}

			for i, task := range tasks {
				if task != tt.expected[i] {
					t.Errorf("parseTasksFromPlan() task[%d] = %s, want %s", i, task, tt.expected[i])
				}
			}
		})
	}
}

func TestExtractChecklistItem(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		wantChecked    bool
		wantTaskText   string
		wantFound      bool
	}{
		// Dash format
		{"dash unchecked", "- [ ] Task text", false, "Task text", true},
		{"dash checked lowercase", "- [x] Task text", true, "Task text", true},
		{"dash checked uppercase", "- [X] Task text", true, "Task text", true},
		{"dash no space", "- [] Task text", false, "", false},

		// Asterisk format
		{"asterisk unchecked", "* [ ] Task text", false, "Task text", true},
		{"asterisk checked", "* [x] Task text", true, "Task text", true},

		// Numbered format
		{"numbered unchecked", "1. [ ] Task text", false, "Task text", true},
		{"numbered checked", "2. [x] Task text", true, "Task text", true},
		{"double digit numbered", "10. [ ] Task", false, "Task", true},

		// Bare checkbox
		{"bare unchecked", "[ ] Task text", false, "Task text", true},
		{"bare checked", "[x] Task text", true, "Task text", true},

		// Invalid formats
		{"no checkbox", "Just text", false, "", false},
		{"dash only", "- Just a dash", false, "", false},
		{"heading", "# Heading", false, "", false},
		{"empty", "", false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChecked, gotTaskText, gotFound := extractChecklistItem(tt.line)
			if gotChecked != tt.wantChecked {
				t.Errorf("extractChecklistItem() checked = %v, want %v", gotChecked, tt.wantChecked)
			}
			if gotTaskText != tt.wantTaskText {
				t.Errorf("extractChecklistItem() taskText = %q, want %q", gotTaskText, tt.wantTaskText)
			}
			if gotFound != tt.wantFound {
				t.Errorf("extractChecklistItem() found = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

func TestIsNumber(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1", true},
		{"12", true},
		{"123", true},
		{"0", true},
		{"", false},
		{"12a", false},
		{"a12", false},
		{"1.2", false},
		{"12 ", false},
		{" 12", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isNumber(tt.input)
			if got != tt.want {
				t.Errorf("isNumber(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
