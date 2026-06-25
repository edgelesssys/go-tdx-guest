package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestProvenance(t *testing.T) {
	tests := []struct {
		name           string
		ppid           string
		bucket         string
		serverResponse string
		serverStatus   int
		validOutput    bool
		wantErr        string
	}{
		{
			name:           "Direct PPID Success",
			ppid:           "abcdef1234567890abcdef1234567890",
			bucket:         "test-bucket",
			serverResponse: `{"location": "us-east1"}`,
			serverStatus:   http.StatusOK,
			validOutput:    true,
		},
		{
			name:           "No Such Bucket",
			ppid:           "abcdef1234567890abcdef1234567890",
			bucket:         "missing-bucket",
			serverResponse: "NoSuchBucket",
			serverStatus:   http.StatusNotFound,
			wantErr:        "gcs request failed: bucket 'missing-bucket' not found",
		},
		{
			name:           "No Such Key",
			ppid:           "0000000000000000000000000000000a",
			bucket:         "test-bucket",
			serverResponse: "NoSuchKey",
			serverStatus:   http.StatusNotFound,
			wantErr:        "gcs request failed: file '0000000000000000000000000000000a.json' not found in bucket 'test-bucket'",
		},
		{
			name:           "Server Error",
			ppid:           "abcdef1234567890abcdef1234567890",
			bucket:         "test-bucket",
			serverResponse: "internal error",
			serverStatus:   http.StatusInternalServerError,
			wantErr:        "gcs request failed with status: 500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/%s/%s.json", tt.bucket, tt.ppid)
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				w.WriteHeader(tt.serverStatus)
				fmt.Fprint(w, tt.serverResponse)
			}))
			defer ts.Close()

			oldPpid := *ppidFlag
			oldBucket := *bucketNameFlag
			oldOut := *outputFlag
			t.Cleanup(func() {
				*ppidFlag = oldPpid
				*bucketNameFlag = oldBucket
				*outputFlag = oldOut
			})

			*ppidFlag = tt.ppid
			*bucketNameFlag = tt.bucket

			outFile := filepath.Join(t.TempDir(), "output.json")
			*outputFlag = outFile

			err := run(ts.URL)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if err.Error() != tt.wantErr {
					t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("run failed: %v", err)
				}
				if tt.validOutput {
					b, err := os.ReadFile(outFile)
					if err != nil {
						t.Fatalf("failed to read output file: %v", err)
					}
					if string(b) != tt.serverResponse {
						t.Errorf("expected %s, got %s", tt.serverResponse, string(b))
					}
				}
			}
		})
	}
}
