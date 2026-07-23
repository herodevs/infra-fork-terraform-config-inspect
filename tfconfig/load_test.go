// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfconfig

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
)

func TestLoadModuleWithIndexedProviderReferences(t *testing.T) {
	fs := fstest.MapFS{
		"main.tofu": &fstest.MapFile{Data: []byte(`
variable "region" {
  type = string
}

provider "aws" {
  for_each = toset(["us-west-2"])
  alias    = "by_region"
}

resource "aws_s3_bucket" "example" {
  for_each = toset(["us-west-2"])
  provider = aws.by_region[each.key]
  bucket   = "example"
}

data "aws_caller_identity" "current" {
  provider = aws.by_region[var.region]
}
`)},
	}

	module, diags := LoadModuleFromFilesystem(WrapFS(fs), ".")
	if diags.HasErrors() {
		t.Fatalf("LoadModuleFromFilesystem returned errors: %s", diags.Error())
	}

	tests := []struct {
		name     string
		resource *Resource
	}{
		{
			name:     "managed resource",
			resource: module.ManagedResources["aws_s3_bucket.example"],
		},
		{
			name:     "data resource",
			resource: module.DataResources["data.aws_caller_identity.current"],
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.resource == nil {
				t.Fatal("resource was not loaded")
			}
			if test.resource.Provider.Name != "aws" || test.resource.Provider.Alias != "by_region" {
				t.Fatalf("provider = %#v, want aws.by_region", test.resource.Provider)
			}
		})
	}
}

func TestLoadModule(t *testing.T) {
	fixturesDir := "testdata"
	testDirs, err := ioutil.ReadDir(fixturesDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range testDirs {
		if !info.IsDir() {
			continue
		}
		t.Run(info.Name(), func(t *testing.T) {
			name := info.Name()
			path := filepath.Join(fixturesDir, name)

			wantSrc, err := ioutil.ReadFile(filepath.Join(path, name+".out.json"))
			if err != nil {
				t.Fatalf("failed to read result file: %s", err)
			}
			var want map[string]interface{}
			err = json.Unmarshal(wantSrc, &want)
			if err != nil {
				t.Fatalf("failed to parse result file: %s", err)
			}

			gotObj, _ := LoadModule(path)
			if gotObj == nil {
				t.Fatalf("result object is nil; want a real object")
			}

			gotSrc, err := json.Marshal(gotObj)
			if err != nil {
				t.Fatalf("result is not JSON-able: %s", err)
			}
			var got map[string]interface{}
			err = json.Unmarshal(gotSrc, &got)
			if err != nil {
				t.Fatalf("failed to parse the actual result (!?): %s", err)
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func TestLoadModuleFromFilesystem(t *testing.T) {
	fixturesDir := "testdata"
	testDirs, err := ioutil.ReadDir(fixturesDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range testDirs {
		if !info.IsDir() {
			continue
		}
		t.Run(info.Name(), func(t *testing.T) {
			name := info.Name()
			path := filepath.Join(fixturesDir, name)
			fs := os.DirFS(".")

			wantSrc, err := ioutil.ReadFile(filepath.Join(path, name+".out.json"))
			if err != nil {
				t.Fatalf("failed to read result file: %s", err)
			}
			var want map[string]interface{}
			err = json.Unmarshal(wantSrc, &want)
			if err != nil {
				t.Fatalf("failed to parse result file: %s", err)
			}

			gotObj, _ := LoadModuleFromFilesystem(WrapFS(fs), path)
			if gotObj == nil {
				t.Fatalf("result object is nil; want a real object")
			}

			gotSrc, err := json.Marshal(gotObj)
			if err != nil {
				t.Fatalf("result is not JSON-able: %s", err)
			}
			var got map[string]interface{}
			err = json.Unmarshal(gotSrc, &got)
			if err != nil {
				t.Fatalf("failed to parse the actual result (!?): %s", err)
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}
