package stealth

import (
	"github.com/kyungw00k/sw/internal/browser"
)

// hideWebDriver hides the webdriver property from detection.
func (m *Module) hideWebDriver(page browser.Page) error {
	script := `
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined,
			configurable: true
		});

		// Remove webdriver flag from Chrome
		if (window.chrome && window.chrome.runtime) {
			delete window.chrome.runtime;
		}

		// Remove automation flags
		const originalQuery = window.navigator.permissions.query;
		window.navigator.permissions.query = (parameters) => (
			parameters.name === 'notifications' ?
				Promise.resolve({ state: Notification.permission }) :
				originalQuery(parameters)
		);
	`
	_, err := page.Evaluate(script)
	return err
}

// modifyNavigator modifies navigator properties to appear more human-like.
func (m *Module) modifyNavigator(page browser.Page) error {
	// Set platform and vendor
	script := `
		// Override plugins to look like a real browser
		Object.defineProperty(navigator, 'plugins', {
			get: () => [
				{
					name: 'Chrome PDF Plugin',
					description: 'Portable Document Format',
					filename: 'internal-pdf-viewer',
					length: 1
				},
				{
					name: 'Chrome PDF Viewer',
					description: '',
					filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai',
					length: 1
				},
				{
					name: 'Native Client',
					description: '',
					filename: 'internal-nacl-plugin',
					length: 2
				}
			]
		});

		// Override languages
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en']
		});

		// Override hardwareConcurrency to hide headless detection
		Object.defineProperty(navigator, 'hardwareConcurrency', {
			get: () => 8
		});

		// Override deviceMemory
		Object.defineProperty(navigator, 'deviceMemory', {
			get: () => 8
		});
	`
	_, err := page.Evaluate(script)
	return err
}

// modifyWebGL modifies WebGL renderer information.
func (m *Module) modifyWebGL(page browser.Page) error {
	renderer := m.config.WebGLRenderer
	if renderer == "" {
		renderer = "ANGLE (Intel, Intel(R) UHD Graphics 630, OpenGL 4.6)"
	}

	script := `
		const getParameter = WebGLRenderingContext.prototype.getParameter;
		WebGLRenderingContext.prototype.getParameter = function(parameter) {
			// UNMASKED_VENDOR_WEBGL
			if (parameter === 37445) {
				return 'Google Inc. (Intel)';
			}
			// UNMASKED_RENDERER_WEBGL
			if (parameter === 37446) {
				return '` + renderer + `';
			}
			return getParameter.call(this, parameter);
		};

		// Also patch WebGL2
		if (typeof WebGL2RenderingContext !== 'undefined') {
			const getParameter2 = WebGL2RenderingContext.prototype.getParameter;
			WebGL2RenderingContext.prototype.getParameter = function(parameter) {
				if (parameter === 37445) {
					return 'Google Inc. (Intel)';
				}
				if (parameter === 37446) {
					return '` + renderer + `';
				}
				return getParameter2.call(this, parameter);
			};
		}
	`
	_, err := page.Evaluate(script)
	return err
}

// patchChromeRuntime patches Chrome runtime detection.
func (m *Module) patchChromeRuntime(page browser.Page) error {
	script := `
		// Patch chrome runtime to avoid detection
		if (!window.chrome) {
			window.chrome = {
				runtime: {}
			};
		}

		// Add connection APIs
		if (!window.chrome.runtime) {
			window.chrome.runtime = {
				connect: function() {
					return {
						onDisconnect: { addListener: function() {} },
						onMessage: { addListener: function() {} },
						postMessage: function() {}
					};
				},
				sendMessage: function() {},
				onMessage: { addListener: function() {} },
				onConnect: { addListener: function() {} }
			};
		}

		// Fix permission API
		const originalQuery = window.navigator.permissions.query;
		window.navigator.permissions.query = (parameters) => (
			parameters.name === 'notifications' ?
				Promise.resolve({ state: Notification.permission }) :
				originalQuery(parameters)
		);
	`
	_, err := page.Evaluate(script)
	return err
}
