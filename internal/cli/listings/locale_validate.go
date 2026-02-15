package listings

import (
	"fmt"
	"sort"
	"strings"
)

// SupportedLocales is the set of BCP-47 locale codes supported by Google Play.
var SupportedLocales = map[string]bool{
	"af":     true,
	"am":     true,
	"ar":     true,
	"az-AZ":  true,
	"be":     true,
	"bg":     true,
	"bn-BD":  true,
	"ca":     true,
	"cs-CZ":  true,
	"da-DK":  true,
	"de-DE":  true,
	"el-GR":  true,
	"en-AU":  true,
	"en-CA":  true,
	"en-GB":  true,
	"en-IN":  true,
	"en-SG":  true,
	"en-US":  true,
	"en-ZA":  true,
	"es-419": true,
	"es-ES":  true,
	"es-US":  true,
	"et":     true,
	"eu-ES":  true,
	"fa":     true,
	"fi-FI":  true,
	"fil":    true,
	"fr-CA":  true,
	"fr-FR":  true,
	"gl-ES":  true,
	"gu":     true,
	"hi-IN":  true,
	"hr":     true,
	"hu-HU":  true,
	"hy-AM":  true,
	"id":     true,
	"is-IS":  true,
	"it-IT":  true,
	"iw-IL":  true,
	"ja-JP":  true,
	"ka-GE":  true,
	"kk":     true,
	"km-KH":  true,
	"kn-IN":  true,
	"ko-KR":  true,
	"ky-KG":  true,
	"lo-LA":  true,
	"lt":     true,
	"lv":     true,
	"mk-MK":  true,
	"ml-IN":  true,
	"mn-MN":  true,
	"mr-IN":  true,
	"ms":     true,
	"ms-MY":  true,
	"my-MM":  true,
	"nb-NO":  true,
	"ne-NP":  true,
	"nl-NL":  true,
	"or":     true,
	"pa":     true,
	"pl-PL":  true,
	"pt-BR":  true,
	"pt-PT":  true,
	"ro":     true,
	"ru-RU":  true,
	"si-LK":  true,
	"sk":     true,
	"sl":     true,
	"sq":     true,
	"sr":     true,
	"sv-SE":  true,
	"sw":     true,
	"ta-IN":  true,
	"te-IN":  true,
	"th":     true,
	"tr-TR":  true,
	"uk":     true,
	"ur":     true,
	"uz":     true,
	"vi":     true,
	"zh-CN":  true,
	"zh-HK":  true,
	"zh-TW":  true,
	"zu":     true,
}

// ValidateLocale checks whether code is a known Google Play locale.
// It returns a helpful error with suggestions for common mistakes.
func ValidateLocale(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("locale code is empty")
	}

	if SupportedLocales[code] {
		return nil
	}

	// Check for underscore instead of hyphen (e.g. en_US -> en-US).
	if strings.Contains(code, "_") {
		fixed := strings.ReplaceAll(code, "_", "-")
		if SupportedLocales[fixed] {
			return fmt.Errorf("invalid locale %q: did you mean %q? Use hyphens, not underscores", code, fixed)
		}
	}

	// Check for case mismatch (e.g. en-us -> en-US).
	for loc := range SupportedLocales {
		if strings.EqualFold(loc, code) {
			return fmt.Errorf("invalid locale %q: did you mean %q? Check capitalization", code, loc)
		}
	}

	// Check for underscore+case mismatch combined (e.g. en_us -> en-US).
	if strings.Contains(code, "_") {
		fixed := strings.ReplaceAll(code, "_", "-")
		for loc := range SupportedLocales {
			if strings.EqualFold(loc, fixed) {
				return fmt.Errorf("invalid locale %q: did you mean %q? Use hyphens, not underscores", code, loc)
			}
		}
	}

	return fmt.Errorf("invalid locale %q: not a supported Google Play locale. Run `gplay listings locales` to see all supported locales", code)
}

// SortedLocales returns the supported locale codes in sorted order.
func SortedLocales() []string {
	locales := make([]string, 0, len(SupportedLocales))
	for loc := range SupportedLocales {
		locales = append(locales, loc)
	}
	sort.Strings(locales)
	return locales
}
