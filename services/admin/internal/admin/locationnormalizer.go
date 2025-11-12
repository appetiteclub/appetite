package admin

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

type NormalizeLocationRequest struct {
	ProviderRef  string
	SelectedText string
}

type NormalizedLocation struct {
	Provider     string
	ProviderRef  string
	ProviderURL  string
	SearchValue  string
	SelectedText string
	Street       string
	Number       string
	Unit         string
	City         string
	State        string
	PostalCode   string
	Country      string
	Region       string
	Latitude     string
	Longitude    string
	RawJSON      string
}

var countryNames = map[string]string{
	"ad": "Andorra",
	"ae": "United Arab Emirates",
	"af": "Afghanistan",
	"ag": "Antigua and Barbuda",
	"ai": "Anguilla",
	"al": "Albania",
	"am": "Armenia",
	"ao": "Angola",
	"aq": "Antarctica",
	"ar": "Argentina",
	"as": "American Samoa",
	"at": "Austria",
	"au": "Australia",
	"aw": "Aruba",
	"ax": "Aland Islands",
	"az": "Azerbaijan",
	"ba": "Bosnia and Herzegovina",
	"bb": "Barbados",
	"bd": "Bangladesh",
	"be": "Belgium",
	"bf": "Burkina Faso",
	"bg": "Bulgaria",
	"bh": "Bahrain",
	"bi": "Burundi",
	"bj": "Benin",
	"bl": "Saint Barthelemy",
	"bm": "Bermuda",
	"bn": "Brunei",
	"bo": "Bolivia",
	"bq": "Bonaire, Sint Eustatius and Saba",
	"br": "Brazil",
	"bs": "Bahamas",
	"bt": "Bhutan",
	"bv": "Bouvet Island",
	"bw": "Botswana",
	"by": "Belarus",
	"bz": "Belize",
	"ca": "Canada",
	"cc": "Cocos (Keeling) Islands",
	"cd": "Democratic Republic of the Congo",
	"cf": "Central African Republic",
	"cg": "Republic of the Congo",
	"ch": "Switzerland",
	"ci": "Cote d'Ivoire",
	"ck": "Cook Islands",
	"cl": "Chile",
	"cm": "Cameroon",
	"cn": "China",
	"co": "Colombia",
	"cr": "Costa Rica",
	"cu": "Cuba",
	"cv": "Cape Verde",
	"cw": "Curacao",
	"cx": "Christmas Island",
	"cy": "Cyprus",
	"cz": "Czechia",
	"de": "Germany",
	"dj": "Djibouti",
	"dk": "Denmark",
	"dm": "Dominica",
	"do": "Dominican Republic",
	"dz": "Algeria",
	"ec": "Ecuador",
	"ee": "Estonia",
	"eg": "Egypt",
	"eh": "Western Sahara",
	"er": "Eritrea",
	"es": "Spain",
	"et": "Ethiopia",
	"fi": "Finland",
	"fj": "Fiji",
	"fk": "Falkland Islands",
	"fm": "Micronesia",
	"fo": "Faroe Islands",
	"fr": "France",
	"ga": "Gabon",
	"gb": "United Kingdom",
	"gd": "Grenada",
	"ge": "Georgia",
	"gf": "French Guiana",
	"gg": "Guernsey",
	"gh": "Ghana",
	"gi": "Gibraltar",
	"gl": "Greenland",
	"gm": "Gambia",
	"gn": "Guinea",
	"gp": "Guadeloupe",
	"gq": "Equatorial Guinea",
	"gr": "Greece",
	"gs": "South Georgia and the South Sandwich Islands",
	"gt": "Guatemala",
	"gu": "Guam",
	"gw": "Guinea-Bissau",
	"gy": "Guyana",
	"hk": "Hong Kong",
	"hm": "Heard Island and McDonald Islands",
	"hn": "Honduras",
	"hr": "Croatia",
	"ht": "Haiti",
	"hu": "Hungary",
	"id": "Indonesia",
	"ie": "Ireland",
	"il": "Israel",
	"im": "Isle of Man",
	"in": "India",
	"io": "British Indian Ocean Territory",
	"iq": "Iraq",
	"ir": "Iran",
	"is": "Iceland",
	"it": "Italy",
	"je": "Jersey",
	"jm": "Jamaica",
	"jo": "Jordan",
	"jp": "Japan",
	"ke": "Kenya",
	"kg": "Kyrgyzstan",
	"kh": "Cambodia",
	"ki": "Kiribati",
	"km": "Comoros",
	"kn": "Saint Kitts and Nevis",
	"kp": "North Korea",
	"kr": "South Korea",
	"kw": "Kuwait",
	"ky": "Cayman Islands",
	"kz": "Kazakhstan",
	"la": "Laos",
	"lb": "Lebanon",
	"lc": "Saint Lucia",
	"li": "Liechtenstein",
	"lk": "Sri Lanka",
	"lr": "Liberia",
	"ls": "Lesotho",
	"lt": "Lithuania",
	"lu": "Luxembourg",
	"lv": "Latvia",
	"ly": "Libya",
	"ma": "Morocco",
	"mc": "Monaco",
	"md": "Moldova",
	"me": "Montenegro",
	"mf": "Saint Martin",
	"mg": "Madagascar",
	"mh": "Marshall Islands",
	"mk": "North Macedonia",
	"ml": "Mali",
	"mm": "Myanmar",
	"mn": "Mongolia",
	"mo": "Macau",
	"mp": "Northern Mariana Islands",
	"mq": "Martinique",
	"mr": "Mauritania",
	"ms": "Montserrat",
	"mt": "Malta",
	"mu": "Mauritius",
	"mv": "Maldives",
	"mw": "Malawi",
	"mx": "Mexico",
	"my": "Malaysia",
	"mz": "Mozambique",
	"na": "Namibia",
	"nc": "New Caledonia",
	"ne": "Niger",
	"nf": "Norfolk Island",
	"ng": "Nigeria",
	"ni": "Nicaragua",
	"nl": "Netherlands",
	"no": "Norway",
	"np": "Nepal",
	"nr": "Nauru",
	"nu": "Niue",
	"nz": "New Zealand",
	"om": "Oman",
	"pa": "Panama",
	"pe": "Peru",
	"pf": "French Polynesia",
	"pg": "Papua New Guinea",
	"ph": "Philippines",
	"pk": "Pakistan",
	"pl": "Poland",
	"pm": "Saint Pierre and Miquelon",
	"pn": "Pitcairn Islands",
	"pr": "Puerto Rico",
	"ps": "Palestine",
	"pt": "Portugal",
	"pw": "Palau",
	"py": "Paraguay",
	"qa": "Qatar",
	"re": "Reunion",
	"ro": "Romania",
	"rs": "Serbia",
	"ru": "Russia",
	"rw": "Rwanda",
	"sa": "Saudi Arabia",
	"sb": "Solomon Islands",
	"sc": "Seychelles",
	"sd": "Sudan",
	"se": "Sweden",
	"sg": "Singapore",
	"sh": "Saint Helena, Ascension and Tristan da Cunha",
	"si": "Slovenia",
	"sj": "Svalbard and Jan Mayen",
	"sk": "Slovakia",
	"sl": "Sierra Leone",
	"sm": "San Marino",
	"sn": "Senegal",
	"so": "Somalia",
	"sr": "Suriname",
	"ss": "South Sudan",
	"st": "Sao Tome and Principe",
	"sv": "El Salvador",
	"sx": "Sint Maarten",
	"sy": "Syria",
	"sz": "Eswatini",
	"tc": "Turks and Caicos Islands",
	"td": "Chad",
	"tf": "French Southern Territories",
	"tg": "Togo",
	"th": "Thailand",
	"tj": "Tajikistan",
	"tk": "Tokelau",
	"tl": "Timor-Leste",
	"tm": "Turkmenistan",
	"tn": "Tunisia",
	"to": "Tonga",
	"tr": "Turkey",
	"tt": "Trinidad and Tobago",
	"tv": "Tuvalu",
	"tw": "Taiwan",
	"tz": "Tanzania",
	"ua": "Ukraine",
	"ug": "Uganda",
	"um": "United States Minor Outlying Islands",
	"us": "United States",
	"uy": "Uruguay",
	"uz": "Uzbekistan",
	"va": "Vatican City",
	"vc": "Saint Vincent and the Grenadines",
	"ve": "Venezuela",
	"vg": "British Virgin Islands",
	"vi": "United States Virgin Islands",
	"vn": "Vietnam",
	"vu": "Vanuatu",
	"wf": "Wallis and Futuna",
	"ws": "Samoa",
	"ye": "Yemen",
	"yt": "Mayotte",
	"za": "South Africa",
	"zm": "Zambia",
	"zw": "Zimbabwe",
}

var diacriticReplacements = map[rune]string{
	'Ł': "L",
	'ł': "l",
	'Ð': "D",
	'đ': "d",
	'Ø': "O",
	'ø': "o",
	'œ': "oe",
	'Œ': "OE",
	'Æ': "AE",
	'æ': "ae",
	'ß': "ss",
	'Þ': "Th",
	'þ': "th",
}

func buildNormalizedLocation(resolved *ResolvedAddress, selected string) NormalizedLocation {
	if resolved == nil {
		return NormalizedLocation{}
	}

	fallback := buildFallbackDictionary(resolved.Raw)
	address := resolved.Address

	cleanSelected := cleanString(selected)

	result := NormalizedLocation{
		Provider:     resolved.Provider,
		ProviderRef:  resolved.ProviderRef,
		ProviderURL:  resolved.ProviderURL,
		Region:       "",
		SelectedText: cleanSelected,
	}

	if result.SelectedText == "" {
		result.SelectedText = cleanString(resolved.Formatted)
	}

	result.Street = cleanString(firstNonEmptyString(address.Street, fallback["street"]))
	result.Number = cleanString(firstNonEmptyString(address.Number, fallback["number"]))
	result.Unit = cleanString(firstNonEmptyString(address.Unit, fallback["unit"]))
	result.City = cleanString(firstNonEmptyString(address.City, fallback["city"]))
	result.State = cleanString(firstNonEmptyString(address.State, fallback["state"]))
	result.PostalCode = cleanString(firstNonEmptyString(address.PostalCode, fallback["postal_code"]))
	result.Country = expandCountry(cleanString(firstNonEmptyString(address.Country, fallback["country"])))
	result.Region = cleanString(fallback["region"])

	if resolved.Coordinates.Latitude != 0 {
		result.Latitude = formatCoordinate(resolved.Coordinates.Latitude)
	}
	if resolved.Coordinates.Longitude != 0 {
		result.Longitude = formatCoordinate(resolved.Coordinates.Longitude)
	}

	if result.Latitude == "" {
		result.Latitude = fallback["latitude"]
	}
	if result.Longitude == "" {
		result.Longitude = fallback["longitude"]
	}

	result.SearchValue = cleanString(firstNonEmptyString(resolved.Formatted, fallback["formatted"], selected))

	if resolved.Raw != nil {
		if rawJSON, err := json.Marshal(resolved.Raw); err == nil {
			result.RawJSON = string(rawJSON)
		}
	}

	return result
}

func buildFallbackDictionary(raw map[string]any) map[string]string {
	fallback := map[string]string{
		"street":      "",
		"number":      "",
		"unit":        "",
		"city":        "",
		"state":       "",
		"postal_code": "",
		"country":     "",
		"formatted":   "",
		"latitude":    "",
		"longitude":   "",
	}

	if raw == nil {
		return fallback
	}

	register := func(target, value string) {
		trimmed := cleanString(value)
		if target == "" || trimmed == "" {
			return
		}
		existing := fallback[target]
		if existing == "" || preferFallbackOverride(existing, trimmed) {
			fallback[target] = trimmed
		}
	}

	keyMapping := map[string]string{
		"street":         "street",
		"road":           "street",
		"pedestrian":     "street",
		"footway":        "street",
		"neighbourhood":  "city",
		"residential":    "street",
		"highway":        "street",
		"route":          "street",
		"number":         "number",
		"house_number":   "number",
		"housenumber":    "number",
		"house":          "number",
		"building":       "number",
		"unit":           "unit",
		"suite":          "unit",
		"level":          "unit",
		"apartment":      "unit",
		"city":           "city",
		"town":           "city",
		"village":        "city",
		"hamlet":         "city",
		"municipality":   "city",
		"county":         "city",
		"suburb":         "city",
		"neighborhood":   "city",
		"state_district": "state",
		"state":          "state",
		"region":         "state",
		"province":       "state",
		"administrative": "state",
		"postal_code":    "postal_code",
		"postcode":       "postal_code",
		"zip":            "postal_code",
		"country":        "country",
		"country_code":   "country",
		"display_name":   "formatted",
		"name":           "formatted",
		"formatted":      "formatted",
	}

	for key, value := range raw {
		if str := valueToString(value); str != "" {
			if mapped, ok := keyMapping[strings.ToLower(key)]; ok {
				register(mapped, str)
			}
		}
	}

	if addr, ok := raw["address"].(map[string]any); ok {
		for key, value := range addr {
			if str := valueToString(value); str != "" {
				if mapped, ok := keyMapping[strings.ToLower(key)]; ok {
					register(mapped, str)
				}
			}
		}
	}

	if addrList, ok := raw["address"].([]any); ok {
		for _, item := range addrList {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := strings.ToLower(valueToString(entry["type"]))
			if key == "" {
				key = strings.ToLower(valueToString(entry["class"]))
			}
			value := valueToString(entry["localname"])
			if value == "" {
				value = valueToString(entry["name"])
			}
			if value == "" {
				value = valueToString(entry["display_name"])
			}
			if key == "" || value == "" {
				continue
			}
			if mapped, ok := keyMapping[key]; ok {
				register(mapped, value)
			}
		}
	}

	if comps, ok := raw["address_components"].([]any); ok {
		for _, item := range comps {
			comp, ok := item.(map[string]any)
			if !ok {
				continue
			}
			longName := valueToString(comp["long_name"])
			typesAny, ok := comp["types"].([]any)
			if !ok {
				continue
			}
			for _, typeEntry := range typesAny {
				key := strings.ToLower(valueToString(typeEntry))
				mapped := mapGoogleType(key)
				if mapped != "" {
					register(mapped, longName)
				}
			}
		}
	}

	if fallback["formatted"] == "" {
		register("formatted", valueToString(raw["display_name"]))
	}

	if fallback["formatted"] == "" {
		register("formatted", valueToString(raw["name"]))
	}

	if fallback["latitude"] == "" {
		register("latitude", valueToString(raw["lat"]))
	}
	if fallback["latitude"] == "" {
		register("latitude", valueToString(raw["latitude"]))
	}

	if fallback["longitude"] == "" {
		register("longitude", valueToString(raw["lon"]))
	}
	if fallback["longitude"] == "" {
		register("longitude", valueToString(raw["lng"]))
	}
	if fallback["longitude"] == "" {
		register("longitude", valueToString(raw["longitude"]))
	}

	if tags, ok := raw["addresstags"].(map[string]any); ok {
		for key, value := range tags {
			str := valueToString(value)
			if str == "" {
				continue
			}
			switch strings.ToLower(key) {
			case "street", "road", "addr_street":
				register("street", str)
			case "housenumber", "house_number", "addr_housenumber":
				register("number", str)
			case "city", "town", "village", "municipality":
				register("city", str)
			case "state", "state_district", "region", "province":
				register("state", str)
			case "postcode", "postalcode", "postal_code", "zip":
				register("postal_code", str)
			case "country":
				register("country", str)
			case "countrycode", "country_code":
				if len(str) > 3 {
					register("country", str)
				} else {
					register("country", strings.ToUpper(str))
				}
			}
		}
	}

	return fallback
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func formatCoordinate(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func valueToString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func mapGoogleType(value string) string {
	switch value {
	case "street_number":
		return "number"
	case "route":
		return "street"
	case "sublocality", "sublocality_level_1", "neighborhood":
		return "city"
	case "locality":
		return "city"
	case "postal_town":
		return "city"
	case "administrative_area_level_1":
		return "state"
	case "administrative_area_level_2":
		return "city"
	case "country":
		return "country"
	case "postal_code":
		return "postal_code"
	case "subpremise":
		return "unit"
	default:
		return ""
	}
}

func preferFallbackOverride(existing, candidate string) bool {
	if existing == "" {
		return true
	}
	if isLikelyCode(existing) && !isLikelyCode(candidate) {
		return true
	}
	if len(candidate) > len(existing)+2 && strings.Contains(candidate, " ") {
		return true
	}
	return false
}

func isLikelyCode(value string) bool {
	if len(value) == 0 || len(value) > 3 {
		return false
	}
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func cleanString(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	original := trimmed
	if strings.Contains(trimmed, "\\u") {
		if decoded, err := strconv.Unquote("\"" + trimmed + "\""); err == nil {
			trimmed = decoded
		}
	}
	if strings.ContainsAny(trimmed, "ÃÂ") {
		if decoded := decodeLatin1(trimmed); decoded != "" {
			trimmed = decoded
		}
	}
	if !utf8.ValidString(trimmed) || strings.ContainsRune(trimmed, utf8.RuneError) {
		if fallback := stripDiacritics(original); fallback != "" {
			trimmed = fallback
		}
	}
	return trimmed
}

func decodeLatin1(value string) string {
	buf := make([]byte, 0, len(value))
	for _, r := range value {
		if r > 255 {
			return ""
		}
		buf = append(buf, byte(r))
	}
	if !utf8.Valid(buf) {
		return ""
	}
	return string(buf)
}

func stripDiacritics(value string) string {
	decomposed := norm.NFKD.String(value)
	buf := make([]rune, 0, len(decomposed))
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r < 128 {
			buf = append(buf, r)
			continue
		}
		if replacement, ok := diacriticReplacements[r]; ok {
			buf = append(buf, []rune(replacement)...)
			continue
		}
		buf = append(buf, r)
	}
	return norm.NFC.String(string(buf))
}

func expandCountry(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if name, ok := countryNames[lower]; ok {
		return name
	}
	if len(lower) == 2 {
		return strings.ToUpper(lower)
	}
	return trimmed
}
