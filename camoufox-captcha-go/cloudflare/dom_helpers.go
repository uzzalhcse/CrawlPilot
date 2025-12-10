package cloudflare

import (
	"log"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/camoufox-captcha-go/common"
)

// CheckboxData holds a checkbox element and its parent frame
type CheckboxData struct {
	Frame    playwright.Frame
	Checkbox playwright.ElementHandle
}

// GetReadyCheckbox finds a visible checkbox in the given iframes.
//
// This function searches through Cloudflare iframes, looking for checkbox inputs
// and waits until at least one is visible and ready to be clicked.
//
// Parameters:
//   - iframes: List of Cloudflare iframes to search in
//   - delay: Delay between attempts
//   - attempts: Maximum number of attempts
//   - debug: Enable debug logging
//
// Returns:
//   - *CheckboxData: Contains the frame and checkbox if found, nil otherwise
//   - error: any error that occurred
func GetReadyCheckbox(iframes []playwright.Frame, delay time.Duration, attempts int, debug bool) (*CheckboxData, error) {
	// Ensure at least one attempt
	if attempts <= 0 {
		attempts = 1
	}

	for attempt := 0; attempt < attempts; attempt++ {
		var checkboxes []CheckboxData

		// Search for checkboxes in each iframe
		for _, iframe := range iframes {
			// Skip detached iframes
			if iframe.IsDetached() {
				continue
			}

			// Wrap iframe for shadow root search
			wrappedFrame := common.WrapFrame(iframe)

			// Search for checkbox inputs in shadow DOM
			iframeCheckboxes, err := common.SearchShadowRootElements(wrappedFrame, `input[type="checkbox"]`)
			if err != nil {
				if debug {
					log.Printf("[Captcha] Error searching for checkboxes in iframe: %v", err)
				}
				continue
			}

			// Add found checkboxes with their parent iframe
			for _, checkbox := range iframeCheckboxes {
				checkboxes = append(checkboxes, CheckboxData{
					Frame:    iframe,
					Checkbox: checkbox,
				})
			}
		}

		if debug {
			log.Printf("[Captcha] Found %d checkboxes in %d Cloudflare iframes", len(checkboxes), len(iframes))
		}

		// Filter for visible checkboxes
		for _, data := range checkboxes {
			visible, err := data.Checkbox.IsVisible()
			if err != nil {
				continue
			}
			if visible {
				if debug {
					log.Println("[Captcha] Checkbox input is ready to be clicked")
				}
				return &data, nil
			}
		}

		if debug {
			log.Println("[Captcha] Waiting for Cloudflare checkbox input...")
		}
		time.Sleep(delay)
	}

	if debug {
		log.Println("[Captcha] Max attempts reached while waiting for Cloudflare checkbox input")
	}
	return nil, nil
}
