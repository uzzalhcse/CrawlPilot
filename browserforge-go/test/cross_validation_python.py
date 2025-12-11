#!/usr/bin/env python3
"""
Cross-validation test: Generate fingerprints using Python browserforge
and output statistics for comparison with Go implementation.
"""

import json
import sys
from collections import Counter

# Add browserforge to path
sys.path.insert(0, '/home/uzzalh/Workplace/github/uzzalhcse/Crawlify/browserforge')

from browserforge.fingerprints import FingerprintGenerator
from browserforge.headers import HeaderGenerator

def main():
    num_samples = 50
    
    print(f"Generating {num_samples} fingerprints with Python browserforge...")
    print()
    
    fp_gen = FingerprintGenerator()
    header_gen = HeaderGenerator()
    
    # Collect statistics
    browsers = Counter()
    platforms = Counter()
    screen_widths = Counter()
    hardware_concurrencies = Counter()
    user_agents = []
    
    for i in range(num_samples):
        try:
            fp = fp_gen.generate()
            
            # Extract browser from user agent
            ua = fp.navigator.userAgent
            user_agents.append(ua)
            
            if 'Chrome' in ua or 'CriOS' in ua:
                browsers['chrome'] += 1
            elif 'Firefox' in ua or 'FxiOS' in ua:
                browsers['firefox'] += 1
            elif 'Safari' in ua and 'Chrome' not in ua:
                browsers['safari'] += 1
            elif 'Edge' in ua or 'Edg' in ua:
                browsers['edge'] += 1
            else:
                browsers['other'] += 1
            
            # Platform
            platform = fp.navigator.platform if hasattr(fp.navigator, 'platform') else 'unknown'
            if 'Win' in platform:
                platforms['windows'] += 1
            elif 'Mac' in platform:
                platforms['macos'] += 1
            elif 'Linux' in platform:
                platforms['linux'] += 1
            elif 'Android' in platform or 'android' in ua:
                platforms['android'] += 1
            elif 'iPhone' in ua or 'iPad' in ua:
                platforms['ios'] += 1
            else:
                platforms['other'] += 1
            
            # Screen width
            width = fp.screen.width if hasattr(fp.screen, 'width') else 0
            screen_widths[width] += 1
            
            # Hardware concurrency
            hc = fp.navigator.hardwareConcurrency if hasattr(fp.navigator, 'hardwareConcurrency') else 0
            hardware_concurrencies[hc] += 1
            
        except Exception as e:
            print(f"Error generating fingerprint {i}: {e}")
    
    # Output results as JSON for comparison
    results = {
        "total_samples": num_samples,
        "browsers": dict(browsers),
        "platforms": dict(platforms),
        "screen_widths": dict(screen_widths),
        "hardware_concurrencies": dict(hardware_concurrencies),
        "sample_user_agents": user_agents[:5]
    }
    
    print("=== Python browserforge Statistics ===")
    print(json.dumps(results, indent=2))
    
    # Also test headers
    print()
    print("=== Header Generation Test ===")
    headers = header_gen.generate()
    print(f"Generated {len(headers)} headers")
    print(f"User-Agent: {headers.get('User-Agent', headers.get('user-agent', 'N/A'))[:80]}...")

if __name__ == "__main__":
    main()
