package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kadevland/kassetpack"
)

var Version = "dev"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	var err error

	// Dispatch to the appropriate command handler.
	switch os.Args[1] {
	case "-help", "--help", "help":
		printHelp()
		return
	case "-version", "--version", "-v":
		fmt.Printf("KAssetPack version %s\n", Version)
		return
	case "build":
		err = RunBuild(os.Args[2:])
	case "unpack":
		err = RunUnpack(os.Args[2:])
	default:
		fmt.Printf("unknown command: %s\n\n", os.Args[1])
		printHelp()
		return
	}

	if err != nil {
		if err == flag.ErrHelp {
			return
		}
		log.Fatalf("X Error: %v", err)
	}
}

// printHelp displays the CLI manual.
func printHelp() {
	fmt.Println("KAssetPack - Lightweight Asset Packaging for Go Games")
	fmt.Println("Documentation: https://github.com/kadevland/kassetpack")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  kassetpack <command> [flags]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  build      Scan a directory and create an asset pack (.kdx/.kdt)")
	fmt.Println("  unpack     Extract files from an existing asset pack")
	fmt.Println("")
	fmt.Println("Global Flags:")
	fmt.Println("  -version   Display the tool version")
	fmt.Println("  -help      Display this help")
	fmt.Println("")
	fmt.Println("Build Flags:")
	fmt.Println("  -path      Source directory to package (e.g. ./assets)")
	fmt.Println("  -ext       Comma-separated extensions to include (e.g. \"*.png,*.ogg\")")
	fmt.Println("  -base      Logical path prefix (e.g. \"images\")")
	fmt.Println("  -name      Base name of the pack (default: game)")
	fmt.Println("  -key       XOR obfuscation key (e.g. mysecret)")
	fmt.Println("  -size      Maximum size of each .kdt file in MB (default: 600)")
	fmt.Println("  -out       Output directory (default: ./build)")
	fmt.Println("")
	fmt.Println("Unpack Flags:")
	fmt.Println("  -pack      Full path to the .kdx file to extract")
	fmt.Println("  -key       XOR key used when the pack was built")
	fmt.Println("  -out       Extraction directory (default: ./unpacked)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  kassetpack build -path ./assets -ext *.png,*.ogg -key secret -out ./data")
	fmt.Println("  kassetpack unpack -pack ./data/game.kdx -key secret")
}

// RunBuild contains the build logic and can be tested without running the executable.
func RunBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)

	path := fs.String("path", "./assets", "Source directory")
	exts := fs.String("ext", "*", "Extensions")
	base := fs.String("base", "", "Logical path prefix")
	name := fs.String("name", "game", "Pack name")
	key := fs.String("key", "", "XOR key")
	size := fs.Int("size", 600, "Maximum size in MB")
	outDir := fs.String("out", "./build", "Output directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	builder := kassetpack.NewBuilder()
	builder.SetMaxDataSize(int64(*size))

	if *key != "" {
		builder.SetXORKey([]byte(*key))
	}

	var extList []string
	if *exts != "*" && *exts != "" {
		extList = strings.Split(
			strings.ReplaceAll(*exts, " ", ""),
			",",
		)
	}

	if err := builder.AddFolder(*path, extList, *base); err != nil {
		return fmt.Errorf("X failed to scan directory: %w", err)
	}

	if err := builder.Save(*outDir, *name); err != nil {
		return fmt.Errorf("X failed to build pack: %w", err)
	}

	fmt.Println("Build completed successfully!")

	return nil
}

// RunUnpack contains the extraction logic.
func RunUnpack(args []string) error {
	fs := flag.NewFlagSet("unpack", flag.ContinueOnError)

	packPath := fs.String("pack", "", "Path to the .kdx file")
	key := fs.String("key", "", "XOR key")
	outDir := fs.String("out", "./unpacked", "Extraction directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *packPath == "" {
		return fmt.Errorf("you must provide -pack")
	}

	// Derive the directory and base name from the provided .kdx path.
	//
	// Example:
	// ./build/game.kdx -> dir = "./build", baseName = "game"
	dir := filepath.Dir(*packPath)
	file := filepath.Base(*packPath)
	baseName := strings.TrimSuffix(file, filepath.Ext(file))

	bank := kassetpack.NewBank()

	if err := bank.Load(dir, baseName, []byte(*key)); err != nil {
		return fmt.Errorf("X failed to load pack: %w", err)
	}
	defer bank.Close()

	if err := bank.Unpack(*outDir); err != nil {
		return fmt.Errorf("X failed to unpack: %w", err)
	}

	fmt.Printf("Unpack completed successfully! %d files extracted\n", len(bank.Index))

	return nil
}
