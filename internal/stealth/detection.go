package stealth

import (
	"strconv"

	"github.com/kyungw00k/sw/internal/browser"
)

// ApplyDetectionEvasion applies all detection evasion scripts.
func (m *Module) ApplyDetectionEvasion(page browser.Page) error {
	// Apply iframe content window detection fix
	if err := m.fixIframeContentWindow(page); err != nil {
		return err
	}

	// Apply media codecs detection fix
	if err := m.fixMediaCodecs(page); err != nil {
		return err
	}

	// Apply screen resolution detection fix
	if err := m.fixScreenResolution(page); err != nil {
		return err
	}

	// Apply outerHTML detection fix
	if err := m.fixOuterHTML(page); err != nil {
		return err
	}

	return nil
}

// fixIframeContentWindow fixes iframe contentWindow detection.
func (m *Module) fixIframeContentWindow(page browser.Page) error {
	script := `
		// Fix iframe contentWindow detection
		const originalContentWindow = Object.getOwnPropertyDescriptor(HTMLIFrameElement.prototype, 'contentWindow');
		Object.defineProperty(HTMLIFrameElement.prototype, 'contentWindow', {
			get: function() {
				const window = originalContentWindow.get.call(this);
				if (window) {
					// Apply stealth to iframe window as well
					try {
						Object.defineProperty(window.navigator, 'webdriver', {
							get: () => undefined
						});
					} catch (e) {
						// Ignore if already defined
					}
				}
				return window;
			}
		});
	`
	_, err := page.Evaluate(script)
	return err
}

// fixMediaCodecs fixes media codecs detection.
func (m *Module) fixMediaCodecs(page browser.Page) error {
	script := `
		// Fix media codecs to match real Chrome
		if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
			const originalEnumerateDevices = navigator.mediaDevices.enumerateDevices;
			navigator.mediaDevices.enumerateDevices = function() {
				return originalEnumerateDevices.call(this).then(devices => {
					return devices.map(device => {
						// Ensure device has proper labels
						if (!device.label && device.kind === 'audioinput') {
							return {
								deviceId: device.deviceId,
								kind: device.kind,
								label: 'Default - Microphone (Built-in)',
								groupId: device.groupId
							};
						}
						return device;
					});
				});
			};
		}
	`
	_, err := page.Evaluate(script)
	return err
}

// fixScreenResolution fixes screen resolution detection.
func (m *Module) fixScreenResolution(page browser.Page) error {
	script := `
		// Override screen properties
		Object.defineProperties(window.screen, {
			width: { value: ` + intToString(m.config.Screen.Width) + ` },
			height: { value: ` + intToString(m.config.Screen.Height) + ` },
			availWidth: { value: ` + intToString(m.config.Screen.Width) + ` },
			availHeight: { value: ` + intToString(m.config.Screen.Height-40) + ` },
			colorDepth: { value: 24 },
			pixelDepth: { value: 24 }
		});
	`
	_, err := page.Evaluate(script)
	return err
}

// fixOuterHTML fixes outerHTML detection for headless browsers.
func (m *Module) fixOuterHTML(page browser.Page) error {
	script := `
		// No specific fix needed, but ensure document structure is normal
		if (!document.documentElement.getAttribute('webdriver')) {
			// Good, no webdriver attribute on html element
		}
	`
	_, err := page.Evaluate(script)
	return err
}

// intToString converts int to string.
func intToString(i int) string {
	return strconv.Itoa(i)
}
