// Copyright 2019 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package blobs

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/blobs/blobspb"
	"github.com/cockroachdb/cockroach/pkg/testutils"
)

func TestBlobServiceGetBlob(t *testing.T) {
	tmpDir, cleanupFn := testutils.TempDir(t)
	defer cleanupFn()

	fileContent := []byte("file_content")
	filename := "path/to/file/content.txt"
	writeTestFile(t, filepath.Join(tmpDir, filename), fileContent)

	service, err := NewBlobService(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.TODO()

	t.Run("get-correct-file", func(t *testing.T) {
		resp, err := service.GetBlob(ctx, &blobspb.GetRequest{
			Filename: filename,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(resp.Payload, fileContent) {
			t.Fatal(fmt.Sprintf(
				`file content is incorrect. expected: %s got: %s`,
				fileContent, resp.Payload,
			))
		}
	})
	t.Run("file-not-exist", func(t *testing.T) {
		_, err := service.GetBlob(ctx, &blobspb.GetRequest{
			Filename: "file/does/not/exist",
		})
		if err == nil {
			t.Fatal("expected error but was not caught")
		}
		if !testutils.IsError(err, "no such file") {
			t.Fatal("incorrect error message: " + err.Error())
		}
	})
	t.Run("not-in-external-io-dir", func(t *testing.T) {
		_, err := service.PutBlob(ctx, &blobspb.PutRequest{
			Filename: "file/../../content.txt",
		})
		if err == nil {
			t.Fatal("expected error but was not caught")
		}
		if !testutils.IsError(err, "outside of external-io-dir is not allowed") {
			t.Fatal("incorrect error message: " + err.Error())
		}
	})
}

func TestBlobServicePutBlob(t *testing.T) {
	tmpDir, cleanupFn := testutils.TempDir(t)
	defer cleanupFn()

	service, err := NewBlobService(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.TODO()

	t.Run("put-correct-file", func(t *testing.T) {
		fileContent := []byte("file_content")
		filename := "path/to/file/content.txt"
		_, err := service.PutBlob(ctx, &blobspb.PutRequest{
			Filename: filename,
			Payload:  fileContent,
		})
		if err != nil {
			t.Fatal(err)
		}
		result, err := ioutil.ReadFile(filepath.Join(tmpDir, filename))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(result, fileContent) {
			t.Fatal(fmt.Sprintf(
				`file content is incorrect. expected: %s got: %s`,
				fileContent, result,
			))
		}
	})
	t.Run("not-in-external-io-dir", func(t *testing.T) {
		_, err := service.PutBlob(ctx, &blobspb.PutRequest{
			Filename: "file/../../content.txt",
		})
		if err == nil {
			t.Fatal("expected error but was not caught")
		}
		if !testutils.IsError(err, "outside of external-io-dir is not allowed") {
			t.Fatal("incorrect error message: " + err.Error())
		}
	})
}

func TestBlobServiceList(t *testing.T) {
	tmpDir, cleanupFn := testutils.TempDir(t)
	defer cleanupFn()

	fileContent := []byte("a")
	dir := filepath.Join(tmpDir, "file/dir")
	files := []string{"a.csv", "b.csv", "c.csv"}
	for _, file := range files {
		writeTestFile(t, filepath.Join(dir, file), fileContent)
	}

	service, err := NewBlobService(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.TODO()

	t.Run("list-correct-files", func(t *testing.T) {
		resp, err := service.List(ctx, &blobspb.GlobRequest{
			Pattern: "file/dir/*.csv",
		})
		if err != nil {
			t.Fatal(err)
		}
		resultList := resp.Files
		if len(resultList) != len(files) {
			t.Fatal("result list does not have the correct number of files")
		}
		for i, f := range resultList {
			if f != filepath.Join(dir, files[i]) {
				t.Fatalf("result list is incorrect %s", resultList)
			}
		}
	})
	t.Run("not-in-external-io-dir", func(t *testing.T) {
		_, err := service.List(ctx, &blobspb.GlobRequest{
			Pattern: "file/../../*.csv",
		})
		if err == nil {
			t.Fatal("expected error but was not caught")
		}
		if !testutils.IsError(err, "outside of external-io-dir is not allowed") {
			t.Fatal("incorrect error message: " + err.Error())
		}
	})
}

func TestBlobServiceDelete(t *testing.T) {
	tmpDir, cleanupFn := testutils.TempDir(t)
	defer cleanupFn()

	fileContent := []byte("file_content")
	filename := "path/to/file/content.txt"
	writeTestFile(t, filepath.Join(tmpDir, filename), fileContent)

	service, err := NewBlobService(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.TODO()

	t.Run("delete-correct-file", func(t *testing.T) {
		_, err := service.Delete(ctx, &blobspb.DeleteRequest{
			Filename: filename,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(tmpDir, filename)); !os.IsNotExist(err) {
			t.Fatalf("expected not exists err, got: %s", err)
		}
	})
	t.Run("file-not-exist", func(t *testing.T) {
		_, err := service.Delete(ctx, &blobspb.DeleteRequest{
			Filename: "file/does/not/exist",
		})
		if err == nil {
			t.Fatal("expected error but was not caught")
		}
		if !testutils.IsError(err, "no such file") {
			t.Fatal("incorrect error message: " + err.Error())
		}
	})
	t.Run("not-in-external-io-dir", func(t *testing.T) {
		_, err := service.Delete(ctx, &blobspb.DeleteRequest{
			Filename: "file/../../content.txt",
		})
		if err == nil {
			t.Fatal("expected error but was not caught")
		}
		if !testutils.IsError(err, "outside of external-io-dir is not allowed") {
			t.Fatal("incorrect error message: " + err.Error())
		}
	})
}

func TestBlobServiceStat(t *testing.T) {
	tmpDir, cleanupFn := testutils.TempDir(t)
	defer cleanupFn()

	fileContent := []byte("file_content")
	filename := "path/to/file/content.txt"
	writeTestFile(t, filepath.Join(tmpDir, filename), fileContent)

	service, err := NewBlobService(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.TODO()

	t.Run("get-correct-file-size", func(t *testing.T) {
		resp, err := service.Stat(ctx, &blobspb.StatRequest{
			Filename: filename,
		})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Filesize != int64(len(fileContent)) {
			t.Fatalf("expected filesize: %d, got %d", len(fileContent), resp.Filesize)
		}
	})
	t.Run("file-not-exist", func(t *testing.T) {
		_, err := service.Stat(ctx, &blobspb.StatRequest{
			Filename: "file/does/not/exist",
		})
		if err == nil {
			t.Fatal("expected error but was not caught")
		}
		if !testutils.IsError(err, "no such file") {
			t.Fatal("incorrect error message: " + err.Error())
		}
	})
	t.Run("not-in-external-io-dir", func(t *testing.T) {
		_, err := service.Stat(ctx, &blobspb.StatRequest{
			Filename: "file/../../content.txt",
		})
		if err == nil {
			t.Fatal("expected error but was not caught")
		}
		if !testutils.IsError(err, "outside of external-io-dir is not allowed") {
			t.Fatal("incorrect error message: " + err.Error())
		}
	})
	t.Run("stat-directory", func(t *testing.T) {
		_, err := service.Stat(ctx, &blobspb.StatRequest{
			Filename: filepath.Dir(filename),
		})
		if err == nil {
			t.Fatalf("expected error but was not caught")
		}
		if !testutils.IsError(err, "expected a file") {
			t.Fatal("incorrect error message: " + err.Error())
		}
	})
}
