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
import re

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


def get_firefox_version():
    """Get the actual installed Firefox version from version.json"""
    version_paths = [
        os.path.expanduser("~/.cache/camoufox/version.json"),
        os.path.expanduser("~/.cache/camoufox-go/version.json"),
    ]
    
    for path in version_paths:
        if os.path.exists(path):
            try:
                with open(path, 'r') as f:
                    data = json.load(f)
                    version = data.get('version', '142.0')
                    # Extract major version (e.g., "142" from "142.0.1")
                    return version.split('.')[0]
            except:
                pass
    
    return "142"  # Default to 142 for coryking's fork


def update_version_in_string(value, ff_version):
    r"""Replace Firefox version numbers in a string with the actual version.
    
    Python pattern from camoufox: re.sub(r'(?<!\d)(1[0-9]{2})(\.0)(?!\d)', rf'{ff_version}\2', data)
    """
    if not isinstance(value, str):
        return value
    # Replace 1XX.0 patterns (e.g., 140.0 -> 142.0)
    return re.sub(r'(?<!\d)(1[0-9]{2})(\.0)(?!\d)', rf'{ff_version}\2', value)


def main():
    # Parse command line arguments
    os_target = sys.argv[1] if len(sys.argv) > 1 else "linux"
    max_width = int(sys.argv[2]) if len(sys.argv) > 2 else None
    max_height = int(sys.argv[3]) if len(sys.argv) > 3 else None

    # Get actual Firefox version
    ff_version = get_firefox_version()

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

    # Extract navigator properties (matching browserforge.yml)
    nav = fingerprint.navigator
    extra_props = getattr(nav, 'extraProperties', None) or {}
    
    # Get raw values
    user_agent = getattr(nav, 'userAgent', '')
    app_version = getattr(nav, 'appVersion', '')
    
    # Fix version in UserAgent and appVersion to match actual Firefox
    user_agent = update_version_in_string(user_agent, ff_version)
    app_version = update_version_in_string(app_version, ff_version)
    
    result = {
        "navigator": {
            "userAgent": user_agent,
            "platform": getattr(nav, 'platform', ''),
            "language": getattr(nav, 'language', 'en-US'),
            "languages": getattr(nav, 'languages', ['en-US']),
            "hardwareConcurrency": getattr(nav, 'hardwareConcurrency', 4),
            "maxTouchPoints": getattr(nav, 'maxTouchPoints', 0),
            "oscpu": getattr(nav, 'oscpu', ''),
            "appCodeName": getattr(nav, 'appCodeName', 'Mozilla'),
            "appName": getattr(nav, 'appName', 'Netscape'),
            "appVersion": app_version,
            "product": getattr(nav, 'product', 'Gecko'),
            # Never override productSub per browserforge.yml #105
            "doNotTrack": getattr(nav, 'doNotTrack', None),
            "globalPrivacyControl": extra_props.get('globalPrivacyControl', None),
        },
        "screen": {
            # Core screen properties
            "width": getattr(fingerprint.screen, 'width', 1920),
            "height": getattr(fingerprint.screen, 'height', 1080),
            "availWidth": getattr(fingerprint.screen, 'availWidth', 1920),
            "availHeight": getattr(fingerprint.screen, 'availHeight', 1040),
            "availLeft": getattr(fingerprint.screen, 'availLeft', 0),
            "availTop": getattr(fingerprint.screen, 'availTop', 0),
            "colorDepth": getattr(fingerprint.screen, 'colorDepth', 24),
            "pixelDepth": getattr(fingerprint.screen, 'pixelDepth', 24),
            # Window properties
            "outerWidth": getattr(fingerprint.screen, 'outerWidth', 1920),
            "outerHeight": getattr(fingerprint.screen, 'outerHeight', 1080),
            "innerWidth": getattr(fingerprint.screen, 'innerWidth', 1920),
            "innerHeight": getattr(fingerprint.screen, 'innerHeight', 969),
            # Screen position
            "screenX": getattr(fingerprint.screen, 'screenX', 0),
            "screenY": getattr(fingerprint.screen, 'screenY', 0),
            "pageXOffset": getattr(fingerprint.screen, 'pageXOffset', 0),
            "pageYOffset": getattr(fingerprint.screen, 'pageYOffset', 0),
        }
    }
    
    # Add headers if available
    if hasattr(fingerprint, 'headers'):
        result["headers"] = {
            "Accept-Encoding": getattr(fingerprint.headers, 'accept_encoding', 'gzip, deflate, br'),
        }
    else:
        result["headers"] = {
            "Accept-Encoding": "gzip, deflate, br",
        }
    
    # Add battery if available
    if hasattr(fingerprint, 'battery'):
        discharging = getattr(fingerprint.battery, 'dischargingTime', None)
        # Convert infinity to None (null in JSON)
        if discharging is not None and discharging == float('inf'):
            discharging = None
        result["battery"] = {
            "charging": getattr(fingerprint.battery, 'charging', True),
            "chargingTime": getattr(fingerprint.battery, 'chargingTime', 0),
            "dischargingTime": discharging,
            "level": getattr(fingerprint.battery, 'level', 1.0),
        }

    # Output only JSON to stdout
    print(json.dumps(result))


if __name__ == "__main__":
    main()

