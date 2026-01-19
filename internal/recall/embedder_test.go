package recall

import "testing"

func TestTrimLastSentence_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single sentence",
			input:    "Hello world.",
			expected: "",
		},
		{
			name:     "two sentences",
			input:    "First sentence. Second sentence.",
			expected: "First sentence.",
		},
		{
			name:     "three sentences",
			input:    "One. Two. Three.",
			expected: "One. Two.",
		},
		{
			name:     "question mark",
			input:    "What is this? It is a test.",
			expected: "What is this?",
		},
		{
			name:     "exclamation mark",
			input:    "Wow! That is cool.",
			expected: "Wow!",
		},
		{
			name:     "newline separator",
			input:    "First sentence.\nSecond sentence.",
			expected: "First sentence.",
		},
		{
			name:     "paragraph boundary fallback",
			input:    "No sentence end here\n\nAnother paragraph",
			expected: "No sentence end here",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "no boundary at all",
			input:    "Just some text without any ending",
			expected: "",
		},
		{
			name:     "trims whitespace",
			input:    "First.   Second.  ",
			expected: "First.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := trimLastSentence(tc.input)
			if result != tc.expected {
				t.Errorf("trimLastSentence(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestTrimLastSentence_Progressive(t *testing.T) {
	// Test that repeated trimming eventually empties the text
	text := "First. Second. Third. Fourth."

	text = trimLastSentence(text)
	if text != "First. Second. Third." {
		t.Errorf("after 1 trim: got %q", text)
	}

	text = trimLastSentence(text)
	if text != "First. Second." {
		t.Errorf("after 2 trims: got %q", text)
	}

	text = trimLastSentence(text)
	if text != "First." {
		t.Errorf("after 3 trims: got %q", text)
	}

	text = trimLastSentence(text)
	if text != "" {
		t.Errorf("after 4 trims: expected empty, got %q", text)
	}
}

func TestIsSentenceEnd(t *testing.T) {
	tests := []struct {
		char     byte
		next     byte
		expected bool
	}{
		{'.', ' ', true},
		{'.', '\n', true},
		{'?', ' ', true},
		{'?', '\n', true},
		{'!', ' ', true},
		{'!', '\n', true},
		{'.', '.', false},
		{'.', 'a', false},
		{',', ' ', false},
		{'a', ' ', false},
	}

	for _, tc := range tests {
		result := isSentenceEnd(tc.char, tc.next)
		if result != tc.expected {
			t.Errorf("isSentenceEnd(%q, %q) = %v, expected %v",
				tc.char, tc.next, result, tc.expected)
		}
	}
}
