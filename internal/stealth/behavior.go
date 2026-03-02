package stealth

import (
	"strconv"
	"time"

	"github.com/kyungw00k/sw/internal/browser"
)

// HumanizeTyping types text with human-like delays between keystrokes.
func (m *Module) HumanizeTyping(page browser.Page, selector, text string) error {
	// First focus the element
	if err := page.Click(selector); err != nil {
		return err
	}

	// Add initial delay
	time.Sleep(m.RandomDelay())

	// Type each character with human-like delay
	for _, char := range text {
		// Random delay between keystrokes (30-100ms)
		delay := time.Duration(m.rand.Intn(70)+30) * time.Millisecond
		time.Sleep(delay)

		// Type the character
		if err := page.Type(selector, string(char)); err != nil {
			return err
		}

		// Occasionally pause longer (simulate thinking)
		if m.rand.Float64() < 0.05 { // 5% chance
			pause := time.Duration(m.rand.Intn(200)+100) * time.Millisecond
			time.Sleep(pause)
		}
	}

	return nil
}

// HumanizeClick clicks an element with human-like mouse movement.
func (m *Module) HumanizeClick(page browser.Page, selector string) error {
	if !m.config.MouseMovement {
		return page.Click(selector)
	}

	// Get element position
	pos, err := m.getElementPosition(page, selector)
	if err != nil {
		// Fallback to direct click
		return page.Click(selector)
	}

	// Move mouse in human-like path
	if err := m.humanMouseMove(page, pos.X, pos.Y); err != nil {
		return err
	}

	// Small delay before click
	time.Sleep(time.Duration(m.rand.Intn(50)+20) * time.Millisecond)

	// Click
	return page.Click(selector)
}

// ElementPosition represents an element's position.
type ElementPosition struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// getElementPosition gets the position of an element.
func (m *Module) getElementPosition(page browser.Page, selector string) (*ElementPosition, error) {
	script := `
		(() => {
			const el = document.querySelector('` + selector + `');
			if (!el) return null;
			const rect = el.getBoundingClientRect();
			return {
				x: rect.x + rect.width / 2,
				y: rect.y + rect.height / 2,
				width: rect.width,
				height: rect.height
			};
		})()
	`
	result, err := page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	// Parse result
	if result == nil {
		return nil, nil
	}

	// Type assert to map
	if m, ok := result.(map[string]interface{}); ok {
		return &ElementPosition{
			X:      m["x"].(float64),
			Y:      m["y"].(float64),
			Width:  m["width"].(float64),
			Height: m["height"].(float64),
		}, nil
	}

	return nil, nil
}

// humanMouseMove moves the mouse in a human-like curved path.
func (m *Module) humanMouseMove(page browser.Page, targetX, targetY float64) error {
	// Generate a curved path using Bezier curve
	steps := 10 + m.rand.Intn(10) // 10-20 steps

	// Start position (center of viewport)
	startX := float64(m.config.Viewport.Width) / 2
	startY := float64(m.config.Viewport.Height) / 2

	// Generate control points for Bezier curve
	cp1x := startX + (targetX-startX)*0.25 + float64(m.rand.Intn(100)-50)
	cp1y := startY + (targetY-startY)*0.25 + float64(m.rand.Intn(100)-50)
	cp2x := startX + (targetX-startX)*0.75 + float64(m.rand.Intn(100)-50)
	cp2y := startY + (targetY-startY)*0.75 + float64(m.rand.Intn(100)-50)

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)

		// Cubic Bezier curve
		x := cubicBezier(t, startX, cp1x, cp2x, targetX)
		y := cubicBezier(t, startY, cp1y, cp2y, targetY)

		// Move mouse
		script := `
			(() => {
				const event = new MouseEvent('mousemove', {
					bubbles: true,
					cancelable: true,
					clientX: ` + floatToString(x) + `,
					clientY: ` + floatToString(y) + `
				});
				document.dispatchEvent(event);
			})()
		`
		_, _ = page.Evaluate(script)

		// Random delay between movements
		time.Sleep(time.Duration(m.rand.Intn(20)+5) * time.Millisecond)
	}

	return nil
}

// cubicBezier calculates a point on a cubic Bezier curve.
func cubicBezier(t, p0, p1, p2, p3 float64) float64 {
	u := 1 - t
	return u*u*u*p0 + 3*u*u*t*p1 + 3*u*t*t*p2 + t*t*t*p3
}

// floatToString converts float64 to string without scientific notation.
func floatToString(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
