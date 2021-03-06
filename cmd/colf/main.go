package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pascaldekloe/colfer"
)

// ANSI escape codes for markup
const (
	bold      = "\x1b[1m"
	italic    = "\x1b[3m"
	underline = "\x1b[4m"
	clear     = "\x1b[0m"
)

var (
	basedir = flag.String("b", ".", "Use a specific destination base `directory`.")
	prefix  = flag.String("p", "", "Adds a package `prefix`. Use slash as a separator when nesting.")
	format  = flag.Bool("f", false, "Normalizes the format of all input schemas on the fly.")
	verbose = flag.Bool("v", false, "Enables verbose reporting to "+italic+"standard error"+clear+".")

	sizeMax = flag.String("s", "16 * 1024 * 1024", "Sets the default upper limit for serial byte sizes. The\n`expression` is applied to the target language under the name\nColferSizeMax.")
	listMax = flag.String("l", "64 * 1024", "Sets the default upper limit for the number of elements in a\nlist. The `expression` is applied to the target language under\nthe name ColferListMax.")

	superClass  = flag.String("x", "", "Makes all generated classes extend a super `class`. Use slash as\na package separator. Java only.")
	interfaces  = flag.String("i", "", "Makes all generated classes implement the `interfaces`. Use commas\nto list and slash as a package separator. Java only.")
	snippetFile = flag.String("c", "", "Insert code snippet from `file`. Java only.")
)

var report = log.New(ioutil.Discard, os.Args[0]+": ", 0)

func main() {
	flag.Parse()

	log.SetFlags(0)
	if *verbose {
		report.SetOutput(os.Stderr)
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	// select language
	var gen func(string, colfer.Packages) error
	switch lang := flag.Arg(0); strings.ToLower(lang) {
	case "c":
		report.Print("set-up for C")
		gen = colfer.GenerateC
		if *superClass != "" {
			log.Fatal("colf: super class not supported with C")
		}
		if *interfaces != "" {
			log.Fatal("colf: interfaces not supported with C")
		}
		if *snippetFile != "" {
			log.Fatal("colf: snippet not supported with C")
		}

	case "go":
		report.Print("set-up for Go")
		gen = colfer.GenerateGo
		if *superClass != "" {
			log.Fatal("colf: super class not supported with Go")
		}
		if *interfaces != "" {
			log.Fatal("colf: interfaces not supported with Go")
		}
		if *snippetFile != "" {
			log.Fatal("colf: snippet not supported with Go")
		}

	case "java":
		report.Print("set-up for Java")
		gen = colfer.GenerateJava

	case "javascript", "js", "ecmascript":
		report.Print("set-up for ECMAScript")
		gen = colfer.GenerateECMA
		if *superClass != "" {
			log.Fatal("colf: super class not supported with ECMAScript")
		}
		if *interfaces != "" {
			log.Fatal("colf: interfaces not supported with ECMAScript")
		}
		if *snippetFile != "" {
			log.Fatal("colf: snippet not supported with ECMAScript")
		}

	default:
		log.Fatalf("colf: unsupported language %q", lang)
	}

	var schemaFiles []string
	if flag.NArg() <= 1 {
		var err error
		schemaFiles, err = filepath.Glob("*.colf")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		for _, f := range flag.Args()[1:] {
			info, err := os.Stat(f)
			if err != nil {
				log.Fatal(err)
			}
			if !info.IsDir() {
				schemaFiles = append(schemaFiles, f)
				continue
			}
			files, err := filepath.Glob(filepath.Join(f, "*.colf"))
			if err != nil {
				log.Fatal(err)
			}
			schemaFiles = append(schemaFiles, files...)
		}
	}
	// normalize and deduplicate
	fileSet := make(map[string]bool, len(schemaFiles))
	for _, f := range schemaFiles {
		f = filepath.Clean(f)
		if fileSet[f] {
			report.Printf("duplicate inclusion of %q ignored", f)
			continue
		}
		schemaFiles[len(fileSet)] = f
		fileSet[f] = true
	}
	schemaFiles = schemaFiles[:len(fileSet)]
	report.Print("found schema files: ", strings.Join(schemaFiles, ", "))

	packages, err := colfer.ParseFiles(schemaFiles)
	if err != nil {
		log.Fatal(err)
	}

	if *format {
		for _, file := range schemaFiles {
			changed, err := colfer.Format(file)
			if err != nil {
				log.Fatal(err)
			}
			if changed {
				log.Printf("colf: formatted %q", file)
			}
		}
	}

	if len(packages) == 0 {
		log.Fatal("colf: no struct definitons found")
	}

	for _, p := range packages {
		p.Name = path.Join(*prefix, p.Name)
		p.SizeMax = *sizeMax
		p.ListMax = *listMax
		p.SuperClass = *superClass
		if *interfaces != "" {
			p.Interfaces = strings.Split(*interfaces, ",")
		}
		if len(*snippetFile) > 0 {
			snippet, err := ioutil.ReadFile(*snippetFile)
			if err != nil {
				log.Fatal(err)
			}
			p.CodeSnippet = string(snippet)
		}
	}

	if err := gen(*basedir, packages); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cmd := os.Args[0]

	help := bold + "NAME\n\t" + cmd + clear + " \u2014 compile Colfer schemas\n\n"
	help += bold + "SYNOPSIS\n\t" + cmd + clear
	help += " [ " + underline + "options" + clear + " ] " + underline + "language" + clear
	help += " [ " + underline + "file" + clear + " " + underline + "..." + clear + " ]\n\n"
	help += bold + "DESCRIPTION\n\t" + clear
	help += "Generates source code for a " + underline + "language" + clear + ". The options are: "
	help += bold + "C" + clear + ", " + bold + "Go" + clear + ",\n"
	help += "\t" + bold + "Java" + clear + " and " + bold + "JavaScript" + clear + ".\n"
	help += "\tThe " + underline + "file" + clear + " operands specify schema input. Directories are scanned\n"
	help += "\tfor files with the colf extension. When no files are given, then\n"
	help += "\tthe current " + italic + "working directory" + clear + " is used.\n"
	help += "\tA package definition may be spread over several schema files.\n"
	help += "\tThe directory hierarchy of the input is not relevant for the\n"
	help += "\tgenerated code.\n\n"
	help += bold + "OPTIONS\n" + clear

	tail := "\n" + bold + "EXIT STATUS" + clear + "\n"
	tail += "\tThe command exits 0 on succes, 1 on compilation failure and 2\n"
	tail += "\twhen invoked without arguments.\n"
	tail += "\n" + bold + "EXAMPLES" + clear + "\n"
	tail += "\tCompile ./io.colf with compact limits as C:\n\n"
	tail += "\t\t" + cmd + " -b src -s 2048 -l 96 C io.colf\n\n"
	tail += "\tCompile ./api/*.colf in package com.example as Java:\n\n"
	tail += "\t\t" + cmd + " -p com/example -x com/example/Parent Java api\n"
	tail += "\n" + bold + "BUGS" + clear + "\n"
	tail += "\tReport bugs at <https://github.com/pascaldekloe/colfer/issues>.\n\n"
	tail += "\tText validation is not part of the marshalling and unmarshalling\n"
	tail += "\tprocess. C and Go just pass any malformed UTF-8 characters. Java\n"
	tail += "\tand JavaScript replace unmappable content with the '?' character\n"
	tail += "\t(ASCII 63).\n\n"
	tail += bold + "SEE ALSO\n\t" + clear + "protoc(1), flatc(1)\n"

	flag.Usage = func() {
		os.Stderr.WriteString(help)
		flag.PrintDefaults()
		os.Stderr.WriteString(tail)
	}
}
