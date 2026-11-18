package jsonc

import "strings"

// StripComments blanks // and /* */ comments while leaving string contents
// alone. VS Code's keybindings.json is JSONC, so users will paste comments.
func StripComments(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	escaped := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		if c == '/' && i+1 < len(data) && data[i+1] == '/' {
			i += 2
			for i < len(data) && data[i] != '\n' {
				out = append(out, ' ')
				i++
			}
			if i < len(data) {
				out = append(out, '\n')
			}
			continue
		}
		if c == '/' && i+1 < len(data) && data[i+1] == '*' {
			i += 2
			for i+1 < len(data) && !(data[i] == '*' && data[i+1] == '/') {
				if data[i] == '\n' {
					out = append(out, '\n')
				} else {
					out = append(out, ' ')
				}
				i++
			}
			i++
			continue
		}
		out = append(out, c)
	}
	return out
}

// StripTrailingCommas drops a comma that is followed only by whitespace and a
// closing bracket, leaving string contents alone. Run it after comments are
// stripped so a comment between the comma and the bracket does not hide it.
func StripTrailingCommas(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	escaped := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
		} else if c == '"' {
			inString = true
		} else if c == ',' {
			j := i + 1
			for j < len(data) && strings.IndexByte(" \t\r\n", data[j]) >= 0 {
				j++
			}
			if j < len(data) && (data[j] == ']' || data[j] == '}') {
				continue
			}
		}
		out = append(out, c)
	}
	return out
}
