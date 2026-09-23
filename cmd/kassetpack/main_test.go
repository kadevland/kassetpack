package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCLIBuildAndUnpackInMemory(t *testing.T) {
	// 1. Prepare fake assets.
	assetsDir := t.TempDir()

	os.WriteFile(
		filepath.Join(assetsDir, "image.png"),
		[]byte("image data"),
		0644,
	)

	buildDir := t.TempDir()
	correctKey := "MySuperKey"

	// 2. Call RunBuild DIRECTLY without compiling an executable.
	buildArgs := []string{
		"-path", assetsDir,
		"-ext", "*.png",
		"-key", correctKey,
		"-out", buildDir,
		"-name", "game",
	}

	if err := RunBuild(buildArgs); err != nil {
		t.Fatalf("RunBuild failed: %v", err)
	}

	// Verify that the .kdx file was created.
	kdxPath := filepath.Join(buildDir, "game.kdx")

	if _, err := os.Stat(kdxPath); os.IsNotExist(err) {
		t.Fatal("game.kdx was not created")
	}

	// 3. Call RunUnpack DIRECTLY using the correct key.
	unpackDir := t.TempDir()

	unpackArgs := []string{
		"-pack", kdxPath,
		"-key", correctKey,
		"-out", unpackDir,
	}

	if err := RunUnpack(unpackArgs); err != nil {
		t.Fatalf("RunUnpack failed: %v", err)
	}

	// 4. Verify the extracted file.
	data, err := os.ReadFile(filepath.Join(unpackDir, "image.png"))
	if err != nil {
		t.Fatalf("Extracted file does not exist: %v", err)
	}

	if string(data) != "image data" {
		t.Errorf("Extracted content is incorrect: %s", string(data))
	}
}

func TestCLIUnpackWrongKeyInMemory(t *testing.T) {
	assetsDir := t.TempDir()

	os.WriteFile(
		filepath.Join(assetsDir, "secret.txt"),
		[]byte("top secret"),
		0644,
	)

	buildDir := t.TempDir()
	correctKey := "CorrectKey"

	// Initial build.
	RunBuild([]string{
		"-path", assetsDir,
		"-ext", "*.txt",
		"-key", correctKey,
		"-out", buildDir,
		"-name", "secret_pack",
	})

	kdxPath := filepath.Join(buildDir, "secret_pack.kdx")

	// Unpack using an incorrect key.
	unpackDir := t.TempDir()
	wrongKey := "WrongKey"

	unpackArgs := []string{
		"-pack", kdxPath,
		"-key", wrongKey,
		"-out", unpackDir,
	}

	// Unpacking should NOT crash the program.
	// File read/decryption errors are handled individually.
	//
	// However, the extracted file must not contain "top secret".
	RunUnpack(unpackArgs)

	// Verify that the extracted file cannot be decoded correctly.
	extractedPath := filepath.Join(unpackDir, "secret.txt")

	if _, err := os.Stat(extractedPath); err == nil {
		data, _ := os.ReadFile(extractedPath)

		if string(data) == "top secret" {
			t.Error("The file was decoded correctly using an incorrect key (security failure)")
		}
	}
}
