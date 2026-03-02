package seleniumbase

import (
	"time"

	"github.com/kyungw00k/sw/internal/browser"
	"github.com/playwright-community/playwright-go"
)

// convertStorageState converts playwright storage state to our type.
func convertStorageState(state playwright.StorageState) *browser.StorageState {
	result := &browser.StorageState{
		Cookies: make([]browser.Cookie, len(state.Cookies)),
		Origins: make([]browser.OriginStorage, len(state.Origins)),
	}

	for i, c := range state.Cookies {
		cookie := browser.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			HTTPOnly: c.HttpOnly,
			Secure:   c.Secure,
		}
		if c.Expires != 0 {
			cookie.Expires = time.Unix(int64(c.Expires), 0)
		}
		if c.SameSite != nil {
			cookie.SameSite = string(*c.SameSite)
		}
		result.Cookies[i] = cookie
	}

	for i, o := range state.Origins {
		result.Origins[i] = browser.OriginStorage{
			Origin: o.Origin,
		}
		if o.LocalStorage != nil {
			result.Origins[i].LocalStorage = make([]browser.StorageEntry, len(o.LocalStorage))
			for j, e := range o.LocalStorage {
				result.Origins[i].LocalStorage[j] = browser.StorageEntry{
					Name:  e.Name,
					Value: e.Value,
				}
			}
		}
	}

	return result
}
