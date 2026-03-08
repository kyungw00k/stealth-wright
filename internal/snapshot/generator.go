// Package snapshot generates page snapshots with element references.
package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kyungw00k/sw/internal/browser"
	"github.com/kyungw00k/sw/pkg/protocol"
)

// Generator generates page snapshots.
type Generator struct {
	outputDir string
}

// NewGenerator creates a new snapshot generator.
func NewGenerator(outputDir string) *Generator {
	return &Generator{outputDir: outputDir}
}

// interactiveRoles is the set of ARIA roles that an AI agent can meaningfully
// interact with (click, fill, hover, etc.). Only these roles receive [ref=eN]
// annotations in the snapshot. Structural/presentational roles are omitted to
// reduce noise.
var interactiveRoles = map[string]bool{
	"link":              true,
	"button":            true,
	"textbox":           true,
	"searchbox":         true,
	"combobox":          true,
	"checkbox":          true,
	"radio":             true,
	"listbox":           true,
	"option":            true,
	"menuitem":          true,
	"menuitemcheckbox":  true,
	"menuitemradio":     true,
	"tab":               true,
	"switch":            true,
	"slider":            true,
	"spinbutton":        true,
	"img":               true, // needed for screenshot --ref and annotated screenshots
}

// roleToTags maps ARIA roles to HTML tag names for matching
var roleToTags = map[string][]string{
	"heading":   {"h1", "h2", "h3", "h4", "h5", "h6"},
	"link":      {"a"},
	"checkbox":  {"input"},
	"radio":     {"input"},
	"textbox":   {"input", "textarea"},
	"searchbox": {"input"},
	"combobox":  {"select"},
	"listbox":   {"select"},
	"button":    {"button"},
	"listitem":  {"li"},
	"paragraph": {"p"},
}

// annotateAriaSnapshot injects [ref=eN] into ARIA snapshot lines by matching to extracted elements.
func annotateAriaSnapshot(ariaSnapshot string, elements []protocol.ElementInfo) string {
	lines := strings.Split(ariaSnapshot, "\n")
	usedElements := make(map[string]bool)
	roleUsed := make(map[string]int) // tracks per-role ordinal for unkeyed elements
	extraRefCounter := len(elements) + 1

	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if !strings.HasPrefix(trimmed, "- ") {
			result = append(result, line)
			continue
		}
		rest := trimmed[2:]
		// Skip property lines (- /key: value)
		if strings.HasPrefix(rest, "/") {
			result = append(result, line)
			continue
		}
		// Extract role (first word)
		roleEnd := strings.IndexAny(rest, " \":")
		if roleEnd < 0 {
			roleEnd = len(rest)
		}
		role := rest[:roleEnd]

		// Skip ref injection for non-interactive structural roles.
		// This keeps the snapshot clean — only elements an agent can act on get refs.
		if !interactiveRoles[role] {
			result = append(result, line)
			continue
		}

		// Extract aria name from quoted string: - role "name" ...
		// Handles backslash-escaped quotes inside the name (e.g. \"foo\").
		ariaName := ""
		if idx := strings.Index(rest, "\""); idx >= 0 {
			i := idx + 1
			var buf strings.Builder
			for i < len(rest) {
				if rest[i] == '\\' && i+1 < len(rest) {
					buf.WriteByte(rest[i+1])
					i += 2
				} else if rest[i] == '"' {
					break
				} else {
					buf.WriteByte(rest[i])
					i++
				}
			}
			ariaName = buf.String()
		}
		// Inline text: - role: text
		if ariaName == "" {
			if idx := strings.Index(rest, ": "); idx >= 0 {
				ariaName = rest[idx+2:]
			}
		}

		matchedRef := ""

		// Try name-based match first
		if ariaName != "" {
			tags := roleToTags[role]
			for _, el := range elements {
				if usedElements[el.Ref] {
					continue
				}
				tagMatch := len(tags) == 0
				for _, t := range tags {
					if el.TagName == t {
						tagMatch = true
						break
					}
				}
				if !tagMatch {
					continue
				}
				if strings.Contains(el.Text, ariaName) || strings.Contains(ariaName, el.Text) && ariaName != "" {
					matchedRef = el.Ref
					usedElements[el.Ref] = true
					break
				}
			}
		}

		// Ordinal match for unkeyed elements (checkbox, textbox, etc.)
		if matchedRef == "" {
			if role == "checkbox" || role == "radio" || role == "textbox" || role == "searchbox" || role == "combobox" {
				tags := roleToTags[role]
				idx := roleUsed[role]
				count := 0
				for _, el := range elements {
					if usedElements[el.Ref] {
						continue
					}
					tagMatch := false
					for _, t := range tags {
						if el.TagName == t {
							tagMatch = true
							break
						}
					}
					if tagMatch {
						if count == idx {
							matchedRef = el.Ref
							usedElements[el.Ref] = true
							roleUsed[role] = idx + 1
							break
						}
						count++
					}
				}
			}
		}

		// No match: assign display-only ref
		if matchedRef == "" {
			matchedRef = fmt.Sprintf("e%d", extraRefCounter)
			extraRefCounter++
		}

		result = append(result, injectRef(line, matchedRef))
	}
	return strings.Join(result, "\n")
}

// injectRef inserts [ref=N] into an ARIA snapshot line at the correct position.
func injectRef(line, ref string) string {
	trimmed := strings.TrimLeft(line, " ")
	indent := line[:len(line)-len(trimmed)]
	rest := trimmed[2:] // remove "- "
	refStr := fmt.Sprintf("[ref=%s]", ref)

	if strings.HasSuffix(rest, ":") {
		return indent + "- " + rest[:len(rest)-1] + " " + refStr + ":"
	}
	if idx := strings.Index(rest, ": "); idx >= 0 {
		return indent + "- " + rest[:idx] + " " + refStr + ": " + rest[idx+2:]
	}
	return indent + "- " + rest + " " + refStr
}

// Generate generates a snapshot for the given page using Playwright's native AriaSnapshot.
func (g *Generator) Generate(page browser.Page) (*protocol.SnapshotResult, error) {
	url := page.URL()
	title := page.Title()

	ariaSnapshot, err := page.AriaSnapshot()
	if err != nil {
		return nil, fmt.Errorf("failed to get aria snapshot: %w", err)
	}

	elements, err := g.extractElements(page)
	if err != nil {
		elements = nil
	}

	// Annotate aria snapshot with [ref=eN]
	annotated := annotateAriaSnapshot(ariaSnapshot, elements)

	snapshot := &protocol.SnapshotResult{
		PageURL:      url,
		PageTitle:    title,
		AriaSnapshot: annotated,
		Elements:     elements,
	}

	if g.outputDir != "" {
		if err := os.MkdirAll(g.outputDir, 0755); err != nil {
			return nil, err
		}
		t := time.Now().UTC()
		ms := t.Nanosecond() / 1e6
		filename := fmt.Sprintf("page-%s-%03dZ.yml", t.Format("2006-01-02T15-04-05"), ms)
		fullPath := filepath.Join(g.outputDir, filename)
		snapshot.Filename = fullPath
		if err := g.Save(snapshot, fullPath); err != nil {
			return nil, err
		}
	}

	return snapshot, nil
}

// extractElements extracts elements with selectors using JavaScript
func (g *Generator) extractElements(page browser.Page) ([]protocol.ElementInfo, error) {
	script := `
() => {
	const refs = [];
	let counter = 1;
	
	// Helper to generate a stable CSS selector
	function generateSelector(el) {
		if (el.id) return '#' + el.id;
		
		const path = [];
		let current = el;
		
		while (current && current !== document.body) {
			let selector = current.tagName.toLowerCase();
			
			// Add useful attributes
			if (current.getAttribute('type')) {
				selector += '[type="' + current.getAttribute('type') + '"]';
			}
			if (current.getAttribute('name')) {
				selector += '[name="' + current.getAttribute('name') + '"]';
			}
			if (current.getAttribute('data-testid')) {
				selector += '[data-testid="' + current.getAttribute('data-testid') + '"]';
			}
			
			// Add nth-child if needed
			const parent = current.parentElement;
			if (parent) {
				const siblings = Array.from(parent.children).filter(c => c.tagName === current.tagName);
				if (siblings.length > 1) {
					const index = siblings.indexOf(current) + 1;
					selector += ':nth-child(' + index + ')';
				}
			}
			
			path.unshift(selector);
			current = current.parentElement;
			
			// Limit path depth
			if (path.length > 5) break;
		}
		
		return path.join(' > ');
	}
	
	// Get all visible elements (including divs with id, class, or event handlers)
	const elements = document.querySelectorAll('a, button, input, select, textarea, [role="button"], [onclick], h1, h2, h3, h4, h5, h6, label, img, [aria-label], div[id], div[onclick], div[style], p, span[id], section, article, form, table, ul, ol, li');
	
	elements.forEach((el) => {
		const rect = el.getBoundingClientRect();
		
		// Skip invisible elements
		if (rect.width === 0 || rect.height === 0) return;
		
		// Get text content
		let text = '';
		if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
			text = el.placeholder || el.value || '';
		} else {
			text = (el.innerText || el.textContent || '').trim().substring(0, 100);
		}
		
		// Get attributes
		const attrs = {};
		['type', 'name', 'id', 'class', 'href', 'src', 'aria-label', 'data-testid', 'placeholder', 'value'].forEach(attr => {
			const val = el.getAttribute(attr);
			if (val) attrs[attr] = val;
		});
		
		refs.push({
			ref: 'e' + counter++,
			selector: generateSelector(el),
			tagName: el.tagName.toLowerCase(),
			text: text,
			attributes: attrs
		});
	});
	
	return refs;
}
`

	result, err := page.Evaluate(script)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate script: %w", err)
	}

	// Parse result
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var elements []protocol.ElementInfo
	if err := json.Unmarshal(data, &elements); err != nil {
		return nil, fmt.Errorf("failed to unmarshal elements: %w", err)
	}

	return elements, nil
}

// Save saves a snapshot to a file.
func (g *Generator) Save(snapshot *protocol.SnapshotResult, path string) error {
	return os.WriteFile(path, []byte(snapshot.AriaSnapshot), 0644)
}

// ResolveRef resolves an element reference to a selector.
func ResolveRef(snapshot *protocol.SnapshotResult, ref string) (string, error) {
	for _, el := range snapshot.Elements {
		if el.Ref == ref {
			return el.Selector, nil
		}
	}
	return "", fmt.Errorf("element reference not found: %s", ref)
}
