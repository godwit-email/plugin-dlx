package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	dlx "github.com/OwO-Network/DLX/translate"
)

func TestLanguages(t *testing.T) {
	for _, tc := range []struct{ in, target, source string }{
		{"zh-CN", "ZH-HANS", "ZH"}, {"zh-TW", "ZH-HANT", "ZH"},
		{"zh-HK", "ZH-HANT", "ZH"}, {"zh-Hant", "ZH-HANT", "ZH"},
		{"en-GB", "EN-GB", "EN"}, {"en", "EN-US", "EN"},
		{"pt-PT", "PT-PT", "PT"}, {"fr-CA", "FR-CA", "FR"},
		{"de-DE", "DE", "DE"}, {"es-419", "ES-419", "ES"},
	} {
		for _, source := range []bool{false, true} {
			want := tc.target
			if source {
				want = tc.source
			}
			got, err := resolveLanguage(tc.in, source)
			if err != nil || got != want {
				t.Errorf("%s source=%v: %q, %v; want %s", tc.in, source, got, err, want)
			}
		}
	}
	for _, in := range []string{"", "auto", "xx", "garbage!", "eo"} {
		if _, err := resolveLanguage(in, false); err == nil {
			t.Errorf("accepted target %q", in)
		}
	}
	if got, err := resolveLanguage("", true); err != nil || got != "" {
		t.Fatal(got, err)
	}
}

func TestSplitPreservesTextAndUTF16Limit(t *testing.T) {
	for _, input := range []string{strings.Repeat("😀", 1600), strings.Repeat("word. ", 800), strings.Repeat("中", 3001), "small"} {
		chunks := splitText(input)
		if strings.Join(chunks, "") != input {
			t.Fatal("split lost text")
		}
		for _, chunk := range chunks {
			if !utf8.ValidString(chunk) || len(utf16.Encode([]rune(chunk))) > 1500 || chunk == "" {
				t.Fatal("invalid chunk")
			}
		}
	}
}

func TestBatchingPreservesParagraphs(t *testing.T) {
	input := []string{"", strings.Repeat("a", 900), " \n\t", strings.Repeat("😀", 800), strings.Repeat("z", 900)}
	calls := 0
	fn := func(source, target string, texts []string) (dlx.DLXTranslationResult, error) {
		calls++
		n := 0
		for _, s := range texts {
			n += len(utf16.Encode([]rune(s)))
			if strings.TrimSpace(s) == "" {
				t.Fatal("sent blank text")
			}
		}
		if n > 1500 {
			t.Fatalf("sent %d UTF16 units", n)
		}
		if source != "" || target != "ZH-HANS" {
			t.Fatal(source, target)
		}
		return dlx.DLXTranslationResult{Code: 200, Data: texts, SourceLang: "EN"}, nil
	}
	got, err := translateTexts(translateRequest{Texts: input, Target: "zh-CN"}, fn)
	if err != nil || !reflect.DeepEqual(got.Texts, input) || calls < 3 {
		t.Fatal(got, err, calls)
	}
	if got.Source != "" {
		t.Fatal("multi-paragraph auto detection must not suppress a mixed-language email")
	}
}

func TestEmptyTextsSkipNetwork(t *testing.T) {
	fn := func(string, string, []string) (dlx.DLXTranslationResult, error) {
		t.Fatal("called upstream")
		return dlx.DLXTranslationResult{}, nil
	}
	for _, texts := range [][]string{{}, {"", " \n"}} {
		got, err := translateTexts(translateRequest{Texts: texts, Target: "en"}, fn)
		if err != nil || got.Texts == nil || !reflect.DeepEqual(texts, got.Texts) {
			t.Fatal(got, err)
		}
	}
}

func TestUpstreamFailures(t *testing.T) {
	for _, tc := range []struct {
		code  int
		data  []string
		err   error
		want  string
		retry bool
	}{
		{429, nil, nil, "rate_limited", true}, {503, nil, nil, "upstream", true},
		{403, nil, nil, "upstream", false}, {413, nil, nil, "invalid_request", false},
		{400, nil, nil, "unsupported_language", false}, {200, nil, nil, "upstream", true},
		{200, []string{""}, nil, "upstream", true}, {0, nil, errors.New("secret proxy credentials"), "upstream", true},
	} {
		_, err := translateTexts(translateRequest{Texts: []string{"hello"}, Target: "zh-CN"}, func(string, string, []string) (dlx.DLXTranslationResult, error) {
			return dlx.DLXTranslationResult{Code: tc.code, Data: tc.data}, tc.err
		})
		if err == nil || err.Code != tc.want || err.Retryable != tc.retry || strings.Contains(err.Message, "secret") {
			t.Fatal(tc, err)
		}
	}
}

func TestNoPartialSuccess(t *testing.T) {
	n := 0
	got, err := translateTexts(translateRequest{Texts: []string{strings.Repeat("x", 3000)}, Target: "en"}, func(_ string, _ string, texts []string) (dlx.DLXTranslationResult, error) {
		n++
		if n == 2 {
			return dlx.DLXTranslationResult{Code: 429}, nil
		}
		return dlx.DLXTranslationResult{Code: 200, Data: texts}, nil
	})
	if err == nil || err.Code != "rate_limited" || got.Texts != nil {
		t.Fatal(got, err)
	}
}

func TestSingleParagraphDetection(t *testing.T) {
	got, err := translateTexts(translateRequest{Texts: []string{"hello"}, Target: "zh-CN"}, func(_ string, _ string, texts []string) (dlx.DLXTranslationResult, error) {
		return dlx.DLXTranslationResult{Code: 200, Data: texts, SourceLang: "EN"}, nil
	})
	if err != nil || got.Source != "en" {
		t.Fatal(got, err)
	}
}
