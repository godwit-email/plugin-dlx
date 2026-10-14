package main

import (
	"strings"
	"time"
	"unicode"

	dlx "github.com/OwO-Network/DLX/translate"
	"golang.org/x/text/language"
)

const batchLimit = 1500

type translateRequest struct {
	Texts  []string `json:"texts"`
	Source string   `json:"source"`
	Target string   `json:"target"`
}

type translateResult struct {
	Texts  []string `json:"texts"`
	Source string   `json:"source"`
}

type translateFunc func(source, target string, texts []string) (dlx.DLXTranslationResult, error)

func resolveLanguage(code string, source bool) (string, *pluginError) {
	if source && (code == "" || strings.EqualFold(code, "auto")) {
		return "", nil
	}
	tag, err := language.Parse(code)
	if err != nil || code == "" {
		return "", failure("unsupported_language", "Unsupported language tag.", false)
	}
	base, _ := tag.Base()
	b := strings.ToUpper(base.String())
	if !strings.Contains("|AR|BG|CS|DA|DE|EL|EN|ES|ET|FI|FR|HE|HU|ID|IT|JA|KO|LT|LV|NB|NL|PL|PT|RO|RU|SK|SL|SV|TR|UK|VI|ZH|", "|"+b+"|") {
		return "", failure("unsupported_language", "DLX does not support this language.", false)
	}
	if source {
		return b, nil
	}
	region, _ := tag.Region()
	r := region.String()
	switch b {
	case "ZH":
		script, _ := tag.Script()
		if script.String() == "Hant" {
			return "ZH-HANT", nil
		}
		return "ZH-HANS", nil
	case "EN":
		if r == "GB" {
			return "EN-GB", nil
		}
		return "EN-US", nil
	case "PT":
		if r == "PT" {
			return "PT-PT", nil
		}
		return "PT-BR", nil
	case "DE":
		if r == "CH" {
			return "DE-CH", nil
		}
	case "FR":
		if r == "CA" {
			return "FR-CA", nil
		}
	case "ES":
		if r == "419" {
			return "ES-419", nil
		}
	}
	return b, nil
}

func utf16Length(s string) int {
	n := 0
	for _, r := range s {
		n++
		if r > 0xffff {
			n++
		}
	}
	return n
}

// Prefer a sentence or word boundary; hard-split unbroken text on a rune boundary.
func splitText(text string) []string {
	var chunks []string
	for len(text) > 0 {
		units, end, boundary := 0, 0, 0
		for i, r := range text {
			width := 1
			if r > 0xffff {
				width = 2
			}
			if units+width > batchLimit {
				break
			}
			units += width
			end = i + len(string(r))
			if unicode.IsSpace(r) || strings.ContainsRune(".!?。！？", r) {
				boundary = end
			}
		}
		if end < len(text) && boundary >= end/2 {
			end = boundary
		}
		chunks = append(chunks, text[:end])
		text = text[end:]
	}
	return chunks
}

func translateTexts(req translateRequest, call translateFunc) (translateResult, *pluginError) {
	source, err := resolveLanguage(req.Source, true)
	if err != nil {
		return translateResult{}, err
	}
	target, err := resolveLanguage(req.Target, false)
	if err != nil {
		return translateResult{}, err
	}
	result := translateResult{Texts: make([]string, len(req.Texts))}
	builders := make([]strings.Builder, len(req.Texts))
	type piece struct {
		paragraph int
		text      string
	}
	var pieces []piece
	paragraphs := 0
	for i, text := range req.Texts {
		if strings.TrimSpace(text) == "" {
			result.Texts[i] = text
			continue
		}
		paragraphs++
		for _, chunk := range splitText(text) {
			pieces = append(pieces, piece{i, chunk})
		}
	}
	detected, consistent := "", true
	// DLX has no context API. Its cold call takes at most 3+5+20 seconds;
	// stop starting batches after 30s to leave room below the host's 60s deadline.
	deadline := time.Now().Add(30 * time.Second)
	for start := 0; start < len(pieces); {
		if time.Now().After(deadline) {
			return translateResult{}, failure("upstream", "Translation took too long; translate a smaller selection.", true)
		}
		var texts []string
		var positions []int
		units, end := 0, start
		for end < len(pieces) {
			p := pieces[end]
			if strings.TrimSpace(p.text) == "" {
				end++
				continue
			}
			n := utf16Length(p.text)
			if units+n > batchLimit {
				break
			}
			units += n
			texts = append(texts, p.text)
			positions = append(positions, end)
			end++
		}
		var translated []string
		if len(texts) > 0 {
			up, callErr := call(source, target, texts)
			if callErr != nil {
				return translateResult{}, failure("upstream", "DLX could not reach the translation service.", true)
			}
			if up.Code != 200 {
				return translateResult{}, upstreamFailure(up.Code)
			}
			if len(up.Data) != len(texts) {
				return translateResult{}, failure("upstream", "DLX returned an incomplete translation.", true)
			}
			for _, text := range up.Data {
				if strings.TrimSpace(text) == "" {
					return translateResult{}, failure("upstream", "DLX returned an empty translation.", true)
				}
			}
			translated = up.Data
			lang, langErr := resolveLanguage(up.SourceLang, true)
			if langErr != nil || lang == "" {
				consistent = false
			}
			if start == 0 {
				detected = strings.ToLower(lang)
			} else if detected != strings.ToLower(lang) {
				consistent = false
			}
		}
		j := 0
		for i := start; i < end; i++ {
			text := pieces[i].text
			if j < len(positions) && positions[j] == i {
				text = translated[j]
				j++
			}
			builders[pieces[i].paragraph].WriteString(text)
		}
		start = end
	}
	for i := range builders {
		if builders[i].Len() > 0 {
			result.Texts[i] = builders[i].String()
		}
	}
	if source != "" {
		result.Source = strings.ToLower(source)
	} else if paragraphs == 1 && consistent {
		result.Source = detected
	}
	return result, nil
}

func upstreamFailure(status int) *pluginError {
	switch status {
	case 400:
		return failure("unsupported_language", "DLX rejected the language pair.", false)
	case 413:
		return failure("invalid_request", "DLX rejected the text length.", false)
	case 429:
		return failure("rate_limited", "DLX is rate limited; try again later.", true)
	case 401, 403:
		return failure("upstream", "The translation service refused this request.", false)
	default:
		return failure("upstream", "The translation service is unavailable.", true)
	}
}
