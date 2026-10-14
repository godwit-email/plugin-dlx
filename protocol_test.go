package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestDescribeMatchesManifest(t *testing.T) {
	b, err := os.ReadFile("plugin.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Name, Version string
		Native        struct{ ABI int }
		Capabilities  []string
	}
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	response := dispatch("describe", []byte(`{}`))
	var got struct {
		OK     bool
		Result struct {
			Name, Version string
			ABI           int
			Capabilities  []string
		}
	}
	if err := json.Unmarshal(response, &got); err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.Result.Name != manifest.Name || got.Result.Version != manifest.Version || got.Result.ABI != manifest.Native.ABI || len(got.Result.Capabilities) != 1 || got.Result.Capabilities[0] != "translate" {
		t.Fatal(string(response))
	}
}

func TestABISizeLimits(t *testing.T) {
	var env struct {
		OK    bool
		Error pluginError
	}
	if err := json.Unmarshal(dispatch("describe", bytes.Repeat([]byte(" "), (1<<20)+1)), &env); err != nil || env.OK || env.Error.Code != "invalid_request" {
		t.Fatal(env, err)
	}
	if err := json.Unmarshal(envelope(string(bytes.Repeat([]byte("x"), 4<<20)), nil), &env); err != nil || env.OK || env.Error.Code != "internal" {
		t.Fatal(env, err)
	}
}

func TestProtocolValidation(t *testing.T) {
	for _, tc := range []struct{ method, payload, code string }{
		{"missing", `{}`, "invalid_request"}, {"describe", `null`, "invalid_request"},
		{"configure", `{"config":{}}`, ""}, {"configure", `{}`, "invalid_request"},
		{"configure", `{"config":{"endpoint":"evil"}}`, "invalid_request"},
		{"translate", `{"texts":[],"source":"","target":"zh-CN"}`, ""},
		{"translate", `{"texts":null,"target":"en"}`, "invalid_request"},
		{"translate", `{"texts":[null],"target":"en"}`, "invalid_request"},
		{"translate", `{"texts":[],"target":"en","extra":true}`, "invalid_request"},
		{"translate", `{"texts":[]} {}`, "invalid_request"},
		{"translate", `{"texts":[],"target":"xx"}`, "unsupported_language"},
	} {
		var got struct {
			OK     bool
			Error  *pluginError
			Result json.RawMessage
		}
		b := dispatch(tc.method, []byte(tc.payload))
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatal(err)
		}
		if tc.code == "" {
			if !got.OK {
				t.Fatalf("%+v: %s", tc, b)
			}
			continue
		}
		if got.OK || got.Error == nil || got.Error.Code != tc.code {
			t.Fatalf("%+v: %s", tc, b)
		}
	}
}
