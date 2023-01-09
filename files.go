// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// LoadedPackageNames are single prefix names of packages that were
// loaded in the list of files to process
var LoadedPackageNames = map[string]bool{}

func IsGoFile(f fs.DirEntry) bool {
	// ignore non-Go files
	name := f.Name()
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") && !f.IsDir()
}

func AddFile(fn string, fls []string, procd map[string]bool) []string {
	if _, has := procd[fn]; has {
		return fls
	}
	fls = append(fls, fn)
	procd[fn] = true
	dir, _ := filepath.Split(fn)
	if dir != "" {
		dir = dir[:len(dir)-1]
		pd, sd := filepath.Split(dir)
		if pd != "" {
			dir = sd
		}
		if !(dir == "mat32") {
			if _, has := LoadedPackageNames[dir]; !has {
				LoadedPackageNames[dir] = true
				// fmt.Printf("package: %s\n", dir)
			}
		}
	}
	return fls
}

func FilesFromPaths(paths []string) []string {
	fls := make([]string, 0, len(paths))
	procd := make(map[string]bool)
	for _, path := range paths {
		switch info, err := os.Stat(path); {
		case err != nil:
			var pkgs []*packages.Package
			dir, fl := filepath.Split(path)
			if dir != "" && fl != "" && strings.HasSuffix(fl, ".go") {
				pkgs, err = packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles}, dir)
			} else {
				fl = ""
				pkgs, err = packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles}, path)
			}
			if err != nil {
				fmt.Println(err)
				continue
			}
			pkg := pkgs[0]
			gofls := pkg.GoFiles
			if fl != "" {
				for _, gf := range gofls {
					if strings.HasSuffix(gf, fl) {
						fls = AddFile(gf, fls, procd)
						// fmt.Printf("added file: %s from package: %s\n", gf, path)
						break
					}
				}
			} else {
				for _, gf := range gofls {
					fls = AddFile(gf, fls, procd)
					// fmt.Printf("added file: %s from package: %s\n", gf, path)
				}
			}
		case !info.IsDir():
			path := path
			fls = AddFile(path, fls, procd)
		default:
			// Directories are walked, ignoring non-Go files.
			err := filepath.WalkDir(path, func(path string, f fs.DirEntry, err error) error {
				if err != nil || !IsGoFile(f) {
					return err
				}
				_, err = f.Info()
				if err != nil {
					return nil
				}
				fls = AddFile(path, fls, procd)
				return nil
			})
			if err != nil {
				log.Println(err)
			}
		}
	}
	return fls
}