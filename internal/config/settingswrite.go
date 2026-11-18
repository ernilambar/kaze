package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
)

// Section is one top-level object of settings.json that a write path
// replaced: before is the section as the file last loaded it, after is the
// normalized value to store.
type Section struct {
	Name   string
	Before any
	After  any
}

// WriteSections stores the sections a command changed and leaves the
// rest of settings.json alone. Only the section keys whose values actually
// differ are rewritten, so unknown keys and the user's formatting survive a
// view toggle. The whole file is rewritten only when it cannot be read as a
// JSON object, the one case where there is no existing text to preserve.
func WriteSections(path string, settings Settings, sections ...Section) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(bytes.TrimSpace(data)) == 0) {
		return Write(path, settings)
	}
	if err != nil {
		return fmt.Errorf("read settings file: %w", err)
	}

	for _, section := range sections {
		members, err := changedMembers(section.Before, section.After)
		if err != nil {
			return fmt.Errorf("encode %s settings: %w", section.Name, err)
		}
		if len(members) == 0 {
			continue
		}
		whole, err := json.Marshal(section.After)
		if err != nil {
			return fmt.Errorf("encode %s settings: %w", section.Name, err)
		}
		data, err = patchSettingsDocument(data, section.Name, members, whole)
		if err != nil {
			return Write(path, settings)
		}
	}
	return writeSettingsFile(path, data)
}

// DropSection removes one top-level section and leaves the rest of the
// file's text untouched. It is used once per upgrade, to move the Open Recent
// list out of settings.json; a missing key or unreadable file is a no-op.
func DropSection(path, name string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(bytes.TrimSpace(data)) == 0) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read settings file: %w", err)
	}
	fields, err := parseJSONObject(data)
	if err != nil {
		return err
	}

	for _, field := range fields {
		if field.key != name {
			continue
		}
		start := field.keyStart
		end := field.valueEnd
		next := skipJSONSpace(data, end)
		if next < len(data) && data[next] == ',' {
			end = next + 1
			// The newline after the removed member ends the line that held it;
			// dropping it too keeps the neighbouring members on adjacent lines.
			if end < len(data) && data[end] == '\r' {
				end++
			}
			if end < len(data) && data[end] == '\n' {
				end++
				for start > 0 && (data[start-1] == ' ' || data[start-1] == '\t') {
					start--
				}
			}
		} else {
			// Last member: remove the separator before it instead.
			previous := start
			for previous > 0 && isJSONSpace(data[previous-1]) {
				previous--
			}
			if previous > 0 && data[previous-1] == ',' {
				start = previous - 1
			}
		}
		return writeSettingsFile(path, applyEdits(data, []textEdit{{start, end, nil}}))
	}
	return nil
}

// changedMembers lists the keys of a settings section whose JSON value moved,
// in struct order. Both sides are marshaled from the same Go type, so a key
// equal on both sides is the one whose text the file already has.
func changedMembers(before, after any) ([]jsonMember, error) {
	oldValue, err := json.Marshal(before)
	if err != nil {
		return nil, err
	}
	newValue, err := json.Marshal(after)
	if err != nil {
		return nil, err
	}

	oldFields, err := parseJSONObject(oldValue)
	if err != nil {
		return nil, err
	}
	newFields, err := parseJSONObject(newValue)
	if err != nil {
		return nil, err
	}

	oldByKey := make(map[string][]byte, len(oldFields))
	for _, field := range oldFields {
		oldByKey[field.key] = oldValue[field.valueStart:field.valueEnd]
	}

	members := make([]jsonMember, 0, len(newFields))
	for _, field := range newFields {
		value := newValue[field.valueStart:field.valueEnd]
		if old, ok := oldByKey[field.key]; ok && bytes.Equal(old, value) {
			continue
		}
		members = append(members, jsonMember{key: field.key, value: value})
	}
	return members, nil
}

// patchSettingsDocument returns data with one top-level section updated. The
// section keeps its unknown keys and its key order; only the values in members
// move. A missing or non-object section is written whole, since there is
// nothing in it to preserve.
func patchSettingsDocument(data []byte, name string, members []jsonMember, whole []byte) ([]byte, error) {
	fields, err := parseJSONObject(data)
	if err != nil {
		return nil, err
	}

	for _, field := range fields {
		if field.key != name {
			continue
		}
		existing := data[field.valueStart:field.valueEnd]
		indent := lineIndent(data, field.keyStart)
		if isJSONObject(existing) {
			patched, err := patchJSONObject(existing, members, indent)
			if err != nil {
				return nil, err
			}
			return applyEdits(data, []textEdit{{field.valueStart, field.valueEnd, patched}}), nil
		}
		return applyEdits(data, []textEdit{{field.valueStart, field.valueEnd, indentJSON(whole, indent)}}), nil
	}

	if len(fields) == 0 {
		addition := append([]byte("\n"), appendMember(nil, "  ", name, whole)...)
		addition = append(addition, '\n')
		return applyEdits(data, []textEdit{{1, 1, addition}}), nil
	}

	addition := append([]byte(",\n"), appendMember(nil, lineIndent(data, fields[0].keyStart), name, whole)...)
	insert := closingBrace(data)
	return applyEdits(data, []textEdit{{insert, insert, addition}}), nil
}

// patchJSONObject replaces the values in members and appends the keys the
// object does not have yet. baseIndent is the indentation of the line that
// holds the object's key, used only when the object is empty.
func patchJSONObject(object []byte, members []jsonMember, baseIndent string) ([]byte, error) {
	fields, err := parseJSONObject(object)
	if err != nil {
		return nil, err
	}

	if len(fields) == 0 {
		indent := baseIndent + "  "
		addition := []byte("\n")
		for i, member := range members {
			if i > 0 {
				addition = append(addition, ",\n"...)
			}
			addition = appendMember(addition, indent, member.key, member.value)
		}
		addition = append(addition, '\n')
		addition = append(addition, baseIndent...)
		return applyEdits(object, []textEdit{{1, 1, addition}}), nil
	}

	indent := lineIndent(object, fields[0].keyStart)
	edits := make([]textEdit, 0, len(members))
	present := make(map[string]bool, len(fields))
	for _, field := range fields {
		present[field.key] = true
		for _, member := range members {
			if member.key == field.key {
				edits = append(edits, textEdit{
					start: field.valueStart,
					end:   field.valueEnd,
					text:  indentJSON(member.value, lineIndent(object, field.keyStart)),
				})
			}
		}
	}

	var addition []byte
	for _, member := range members {
		if present[member.key] {
			continue
		}
		addition = append(addition, ",\n"...)
		addition = appendMember(addition, indent, member.key, member.value)
	}
	if len(addition) > 0 {
		insert := closingBrace(object)
		edits = append(edits, textEdit{insert, insert, addition})
	}
	return applyEdits(object, edits), nil
}

// appendMember writes `"key": value` at indent, indenting the value's
// continuation lines to match.
func appendMember(dst []byte, indent, key string, value []byte) []byte {
	dst = append(dst, indent...)
	dst = append(dst, strconv.Quote(key)...)
	dst = append(dst, ": "...)
	return append(dst, indentJSON(value, indent)...)
}

// closingBrace is where a new member goes: just before the object's closing
// brace, after the separator comma, and before the whitespace that indents the
// brace.
func closingBrace(object []byte) int {
	insert := bytes.LastIndexByte(object, '}')
	for insert > 0 && isJSONSpace(object[insert-1]) {
		insert--
	}
	return insert
}

// indentJSON formats value the way json.MarshalIndent would place it inside
// the document: the first line follows the key, later lines sit one level
// deeper than indent.
func indentJSON(value []byte, indent string) []byte {
	var buf bytes.Buffer
	if err := json.Indent(&buf, value, indent, "  "); err != nil {
		return value
	}
	return buf.Bytes()
}

// jsonMember is one key/value pair of a JSON object.
type jsonMember struct {
	key   string
	value json.RawMessage
}

// jsonField is one member of a parsed JSON object: its key and the byte ranges
// of the key and value inside the text it was parsed from.
type jsonField struct {
	key        string
	keyStart   int
	valueStart int
	valueEnd   int
}

// textEdit replaces data[start:end] with text.
type textEdit struct {
	start int
	end   int
	text  []byte
}

// applyEdits splices non-overlapping edits into data, left to right.
func applyEdits(data []byte, edits []textEdit) []byte {
	sort.Slice(edits, func(i, j int) bool { return edits[i].start < edits[j].start })
	var out bytes.Buffer
	last := 0
	for _, edit := range edits {
		out.Write(data[last:edit.start])
		out.Write(edit.text)
		last = edit.end
	}
	out.Write(data[last:])
	return out.Bytes()
}

// parseJSONObject lists the members of object, which must be a JSON object, and
// records where each value sits so the text can be patched without re-encoding
// it.
func parseJSONObject(object []byte) ([]jsonField, error) {
	decoder := json.NewDecoder(bytes.NewReader(object))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return nil, errors.New("not a JSON object")
	}

	fields := make([]jsonField, 0, 8)
	for decoder.More() {
		keyStart := skipJSONSpace(object, int(decoder.InputOffset()))
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, errors.New("object key is not a string")
		}

		colon := bytes.IndexByte(object[int(decoder.InputOffset()):], ':')
		if colon < 0 {
			return nil, fmt.Errorf("object key %q has no value", key)
		}
		valueStart := skipJSONSpace(object, int(decoder.InputOffset())+colon+1)

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		fields = append(fields, jsonField{
			key:        key,
			keyStart:   keyStart,
			valueStart: valueStart,
			valueEnd:   int(decoder.InputOffset()),
		})
	}
	return fields, nil
}

// lineIndent is the whitespace that starts the line containing pos.
func lineIndent(data []byte, pos int) string {
	start := bytes.LastIndexByte(data[:pos], '\n') + 1
	end := start
	for end < pos && isJSONSpace(data[end]) {
		end++
	}
	return string(data[start:end])
}

func skipJSONSpace(data []byte, pos int) int {
	for pos < len(data) && isJSONSpace(data[pos]) {
		pos++
	}
	return pos
}

func isJSONSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func isJSONObject(data []byte) bool {
	return len(data) > 0 && data[0] == '{'
}
