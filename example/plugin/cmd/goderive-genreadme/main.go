// goderive-genreadme replaces two blocks of go code in a Readme.md with
// the contents of a normal go file and the contents of a generated derived.gen.go file respectively.
package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

func main() {
	log.SetFlags(0)
	flag.Parse()
	paths := flag.Args()
	for _, path := range paths {
		log.Printf("scanning %s", path)
		files, err := ioutil.ReadDir(path)
		if err != nil {
			panic(err)
		}
		var generatedCode []byte
		var implCode []byte
		var docCode []byte
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			name := file.Name()
			filename := filepath.Join(path, name)
			if strings.HasSuffix(filename, "derived.gen.go") {
				generatedCode, err = ioutil.ReadFile(filename)
			} else if strings.HasSuffix(filename, "Readme.md") {
				docCode, err = ioutil.ReadFile(filename)
			} else {
				implCode, err = ioutil.ReadFile(filename)
			}
			if err != nil {
				panic(err)
			}
		}
		if generatedCode == nil {
			panic("did not find derived.gen.go in " + path)
		}
		if docCode == nil {
			panic("did not find Readme.md in " + path)
		}
		if implCode == nil {
			panic("did not find any other go files in " + path)
		}
		docs := bytes.Split(docCode, []byte("```"))
		docs[1] = append([]byte("go\n"), implCode...)
		docs[3] = append([]byte("go\n"), generatedCode...)
		docCode = bytes.Join(docs, []byte("```"))
		writefilename := filepath.Join(path, "Readme.md")
		log.Printf("writing to %s", writefilename)
		if err := ioutil.WriteFile(writefilename, docCode, 0666); err != nil {
			panic(err)
		}
	}
}
