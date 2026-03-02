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

// Generate generates a snapshot for the given page.
func (g *Generator) Generate(page browser.Page) (*protocol.SnapshotResult, error) {
	// Get page info
	url := page.URL()
	title := page.Title()

	// Run JavaScript to extract elements
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
	
	// Get all visible elements
	const elements = document.querySelectorAll('a, button, input, select, textarea, [role="button"], [onclick], h1, h2, h3, h4, h5, h6, label, img, [aria-label]');
	
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

	// Create snapshot result
	snapshot := &protocol.SnapshotResult{
		PageURL:   url,
		PageTitle: title,
		Elements:  elements,
	}

	// Save to file
	if g.outputDir != "" {
		if err := os.MkdirAll(g.outputDir, 0755); err != nil {
			return nil, err
		}

		filename := fmt.Sprintf("snapshot-%s.yml", time.Now().Format("2006-01-02T15-04-05"))
		snapshot.Filename = filename

		if err := g.Save(snapshot, filepath.Join(g.outputDir, filename)); err != nil {
			return nil, err
		}
	}

	return snapshot, nil
}

// Save saves a snapshot to a file.
func (g *Generator) Save(snapshot *protocol.SnapshotResult, path string) error {
	var buf strings.Builder

	buf.WriteString("# SW Snapshot\n")
	buf.WriteString(fmt.Sprintf("# Generated: %s\n\n", time.Now().Format(time.RFC3339)))
	buf.WriteString(fmt.Sprintf("url: %s\n", snapshot.PageURL))
	buf.WriteString(fmt.Sprintf("title: %s\n\n", snapshot.PageTitle))
	buf.WriteString("elements:\n")

	for _, el := range snapshot.Elements {
		buf.WriteString(fmt.Sprintf("  - ref: %s\n", el.Ref))
		buf.WriteString(fmt.Sprintf("    selector: %q\n", el.Selector))
		buf.WriteString(fmt.Sprintf("    tag: %s\n", el.TagName))
		if el.Text != "" {
			buf.WriteString(fmt.Sprintf("    text: %q\n", el.Text))
		}
	}

	return os.WriteFile(path, []byte(buf.String()), 0644)
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
