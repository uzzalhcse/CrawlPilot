#!/usr/bin/env python3
"""
BrowserForge fingerprint generator for camoufox-go.

This script generates realistic browser fingerprints using BrowserForge
and outputs them as JSON for consumption by the Go library.

Usage:
    python3 generate_fingerprint.py [os] [max_width] [max_height]

Arguments:
    os         - Target OS: "windows", "macos", or "linux" (default: "linux")
    max_width  - Maximum screen width (optional)
    max_height - Maximum screen height (optional)
"""

import sys
import json
import os

# Suppress browserforge download messages by redirecting stderr temporarily
_original_stderr = sys.stderr

try:
    # Suppress output during import (browserforge may download models)
    sys.stderr = open(os.devnull, 'w')
    from browserforge.fingerprints import FingerprintGenerator, Screen
    sys.stderr = _original_stderr
except ImportError:
    sys.stderr = _original_stderr
    print(json.dumps({"error": "browserforge not installed. Run: pip install browserforge"}), file=sys.stderr)
    sys.exit(1)
except Exception as e:
    sys.stderr = _original_stderr
    # Continue anyway, the import might have succeeded
    try:
        from browserforge.fingerprints import FingerprintGenerator, Screen
    except:
        print(json.dumps({"error": f"Failed to import browserforge: {str(e)}"}), file=sys.stderr)
        sys.exit(1)


def main():
    # Parse command line arguments
    os_target = sys.argv[1] if len(sys.argv) > 1 else "linux"
    max_width = int(sys.argv[2]) if len(sys.argv) > 2 else None
    max_height = int(sys.argv[3]) if len(sys.argv) > 3 else None

    # Create screen constraints if provided
    screen = None
    if max_width:
        screen = Screen(max_width=max_width, max_height=max_height or max_width)

    # Generate fingerprint (suppress any output during generation)
    try:
        old_stdout = sys.stdout
        sys.stdout = open(os.devnull, 'w')
        generator = FingerprintGenerator(browser='firefox')
        fingerprint = generator.generate(os=os_target, screen=screen)
        sys.stdout = old_stdout
    except Exception as e:
        sys.stdout = old_stdout
        print(json.dumps({"error": str(e)}), file=sys.stderr)
        sys.exit(1)

    # Extract relevant fields
    result = {
        "navigator": {
            "userAgent": getattr(fingerprint.navigator, 'userAgent', ''),
            "platform": getattr(fingerprint.navigator, 'platform', ''),
            "language": getattr(fingerprint.navigator, 'language', 'en-US'),
            "languages": getattr(fingerprint.navigator, 'languages', ['en-US']),
            "hardwareConcurrency": getattr(fingerprint.navigator, 'hardwareConcurrency', 4),
            "maxTouchPoints": getattr(fingerprint.navigator, 'maxTouchPoints', 0),
            "oscpu": getattr(fingerprint.navigator, 'oscpu', ''),
            "appCodeName": getattr(fingerprint.navigator, 'appCodeName', 'Mozilla'),
            "appName": getattr(fingerprint.navigator, 'appName', 'Netscape'),
            "appVersion": getattr(fingerprint.navigator, 'appVersion', ''),
            "product": getattr(fingerprint.navigator, 'product', 'Gecko'),
            "productSub": getattr(fingerprint.navigator, 'productSub', '20030107'),
            "vendor": getattr(fingerprint.navigator, 'vendor', ''),
            "vendorSub": getattr(fingerprint.navigator, 'vendorSub', ''),
        },
        "screen": {
            "width": getattr(fingerprint.screen, 'width', 1920),
            "height": getattr(fingerprint.screen, 'height', 1080),
            "availWidth": getattr(fingerprint.screen, 'availWidth', 1920),
            "availHeight": getattr(fingerprint.screen, 'availHeight', 1040),
            "colorDepth": getattr(fingerprint.screen, 'colorDepth', 24),
            "pixelDepth": getattr(fingerprint.screen, 'pixelDepth', 24),
            "outerWidth": getattr(fingerprint.screen, 'outerWidth', 1920),
            "outerHeight": getattr(fingerprint.screen, 'outerHeight', 1080),
            "innerWidth": getattr(fingerprint.screen, 'innerWidth', 1920),
            "innerHeight": getattr(fingerprint.screen, 'innerHeight', 969),
        }
    }
    
    # Add headers if available
    if hasattr(fingerprint, 'headers'):
        result["headers"] = {
            "User-Agent": getattr(fingerprint.headers, 'user_agent', result["navigator"]["userAgent"]),
            "Accept-Language": getattr(fingerprint.headers, 'accept_language', 'en-US,en;q=0.9'),
        }

    # Output only JSON to stdout
    print(json.dumps(result))


if __name__ == "__main__":
    main()
