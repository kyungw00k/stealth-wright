package seleniumbase

import (
	"github.com/kyungw00k/sw/internal/browser"
	"github.com/playwright-community/playwright-go"
)

// Element wraps playwright.Locator or playwright.ElementHandle.
type Element struct {
	locator playwright.Locator
	handle  playwright.ElementHandle
}

// Click clicks the element.
func (e *Element) Click(opts ...browser.ClickOption) error {
	if e.locator != nil {
		return e.locator.Click()
	}
	if e.handle != nil {
		return e.handle.Click()
	}
	return nil
}

// Hover hovers over the element.
func (e *Element) Hover() error {
	if e.locator != nil {
		return e.locator.Hover()
	}
	if e.handle != nil {
		return e.handle.Hover()
	}
	return nil
}

// Type types text into the element.
func (e *Element) Type(text string, opts ...browser.TypeOption) error {
	if e.locator != nil {
		return e.locator.Fill(text)
	}
	if e.handle != nil {
		return e.handle.Fill(text)
	}
	return nil
}

// Fill fills text into the element.
func (e *Element) Fill(text string) error {
	return e.Type(text)
}

// TextContent returns the text content.
func (e *Element) TextContent() (string, error) {
	if e.locator != nil {
		return e.locator.TextContent()
	}
	if e.handle != nil {
		return e.handle.TextContent()
	}
	return "", nil
}

// InnerText returns the inner text.
func (e *Element) InnerText() (string, error) {
	if e.locator != nil {
		return e.locator.InnerText()
	}
	if e.handle != nil {
		return e.handle.InnerText()
	}
	return "", nil
}

// GetAttribute returns an attribute value.
func (e *Element) GetAttribute(name string) (string, error) {
	if e.locator != nil {
		return e.locator.GetAttribute(name)
	}
	if e.handle != nil {
		return e.handle.GetAttribute(name)
	}
	return "", nil
}

// IsVisible returns whether the element is visible.
func (e *Element) IsVisible() bool {
	if e.locator != nil {
		v, _ := e.locator.IsVisible()
		return v
	}
	if e.handle != nil {
		v, _ := e.handle.IsVisible()
		return v
	}
	return false
}

// IsEnabled returns whether the element is enabled.
func (e *Element) IsEnabled() bool {
	if e.locator != nil {
		v, _ := e.locator.IsEnabled()
		return v
	}
	if e.handle != nil {
		v, _ := e.handle.IsEnabled()
		return v
	}
	return false
}

// IsChecked returns whether the element is checked.
func (e *Element) IsChecked() bool {
	if e.locator != nil {
		v, _ := e.locator.IsChecked()
		return v
	}
	if e.handle != nil {
		v, _ := e.handle.IsChecked()
		return v
	}
	return false
}

// BoundingBox returns the element's bounding box.
func (e *Element) BoundingBox() (*browser.Rect, error) {
	var rect *playwright.Rect
	var err error

	if e.locator != nil {
		rect, err = e.locator.BoundingBox()
	} else if e.handle != nil {
		rect, err = e.handle.BoundingBox()
	}

	if err != nil || rect == nil {
		return nil, err
	}

	return &browser.Rect{
		X:      rect.X,
		Y:      rect.Y,
		Width:  rect.Width,
		Height: rect.Height,
	}, nil
}

// Screenshot takes a screenshot of the element.
func (e *Element) Screenshot(opts ...browser.ScreenshotOption) ([]byte, error) {
	if e.locator != nil {
		return e.locator.Screenshot()
	}
	if e.handle != nil {
		return e.handle.Screenshot()
	}
	return nil, nil
}
