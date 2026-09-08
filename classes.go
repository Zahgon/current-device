package device

import (
	"regexp"
	"strings"
	"sync"
)

// classPatternCache memoises the compiled form of each class pattern. The
// original constructs `new RegExp(className, 'i')` on every call; caching keeps
// that behaviour while avoiding a recompile on every orientation event.
var classPatternCache sync.Map // map[string]*regexp.Regexp

// classPattern compiles a class name as a case-insensitive regular expression.
//
// Matching a class with a regex rather than by token equality is inherited from
// the original: `hasClass` builds `new RegExp(className, 'i')` and matches it
// against the whole className string. Multi-token arguments such as
// "ios ipad tablet" are therefore matched as a single pattern, and a substring
// match is enough — "desktop" matches a className that already contains it.
//
// The second return value reports whether the pattern compiled. Go's RE2 syntax
// is not identical to JavaScript's, so a pattern that JavaScript would accept
// could in principle fail here; where JavaScript would throw, this port treats
// the class as absent. No class name used by this package contains regular
// expression metacharacters, so the two behaviours never diverge in practice.
func classPattern(className string) (*regexp.Regexp, bool) {
	if cached, ok := classPatternCache.Load(className); ok {
		re, ok := cached.(*regexp.Regexp)
		return re, ok
	}
	re, err := regexp.Compile("(?i)" + className)
	if err != nil {
		classPatternCache.Store(className, (*regexp.Regexp)(nil))
		return nil, false
	}
	classPatternCache.Store(className, re)
	return re, true
}

// HasClass reports whether current already contains className, using the
// case-insensitive regular expression matching of the original `hasClass`.
func HasClass(current, className string) bool {
	re, ok := classPattern(className)
	if !ok || re == nil {
		return false
	}
	return re.MatchString(current)
}

// AddClass returns current with className appended, unless it is already
// present.
//
// The existing value is trimmed of leading and trailing whitespace and then
// joined with a single space, exactly as the original does with
// `className.replace(/^\s+|\s+$/g, ”) + ' ' + className`. Appending to an
// empty string therefore yields a leading space — " desktop", not "desktop".
// That is upstream behaviour and is preserved: consumers split the className on
// spaces, so the empty leading token is harmless, and changing it would alter
// the DOM output.
func AddClass(current, className string) string {
	if HasClass(current, className) {
		return current
	}
	return strings.TrimSpace(current) + " " + className
}

// RemoveClass returns current with the first occurrence of " "+className
// removed, if className is present.
//
// Only the first occurrence is removed, and only when preceded by a space,
// mirroring `className.replace(' ' + className, ”)` with a string argument.
func RemoveClass(current, className string) string {
	if !HasClass(current, className) {
		return current
	}
	return strings.Replace(current, " "+className, "", 1)
}

// ClassNames returns current with the device classes for this Device appended.
//
// It reproduces the if/else-if cascade the TypeScript module runs at import
// time, including two behaviours that look like oversights but are load-bearing
// for output parity:
//
//   - NodeWebkit is tested before Television, so an NW.js runtime is labelled
//     "node-webkit" and no OS or form factor class is added. Under jsdom, where
//     a Node.js `process` global exists, this branch always wins.
//   - An iOS device that is somehow neither iPad, iPhone nor iPod adds no class
//     at all.
//
// The Cordova class is appended independently of the cascade, so it can combine
// with any branch.
func (d *Device) ClassNames(current string) string {
	add := func(className string) { current = AddClass(current, className) }

	switch {
	case d.IOS():
		switch {
		case d.IPad():
			add("ios ipad tablet")
		case d.IPhone():
			add("ios iphone mobile")
		case d.IPod():
			add("ios ipod mobile")
		}
	case d.MacOS():
		add("macos desktop")
	case d.HarmonyOS():
		if d.find("mobile") {
			add("harmonyos mobile")
		} else {
			add("harmonyos tablet")
		}
	case d.Android():
		if d.AndroidTablet() {
			add("android tablet")
		} else {
			add("android mobile")
		}
	case d.Blackberry():
		if d.BlackberryTablet() {
			add("blackberry tablet")
		} else {
			add("blackberry mobile")
		}
	case d.Windows():
		switch {
		case d.WindowsTablet():
			add("windows tablet")
		case d.WindowsPhone():
			add("windows mobile")
		default:
			add("windows desktop")
		}
	case d.FxOS():
		if d.FxOSTablet() {
			add("fxos tablet")
		} else {
			add("fxos mobile")
		}
	case d.MeeGo():
		add("meego mobile")
	case d.NodeWebkit():
		add("node-webkit")
	case d.Television():
		add("television")
	case d.Desktop():
		add("desktop")
	}

	if d.Cordova() {
		add("cordova")
	}

	return current
}
