/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"archive/zip"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

import (
	billy "github.com/go-git/go-billy/v5"
)

// Filesystem
// Wrap the implementations of FS with their subtle differences into the
// common interface for accessing template files defined herein.
// os:    standard for on-disk extensible template repositories.
// zip:   embedded filesystem backed by the byte array representing zipfile.
// billy: go-git library's filesystem used for remote git template repos.
type Filesystem interface {
	fs.ReadDirFS
	fs.StatFS
	Readlink(link string) (string, error)
}

type zipFS struct {
	archive *zip.Reader
}

func NewZipFS(archive *zip.Reader) zipFS {
	return zipFS{archive: archive}
}

func (z zipFS) Open(name string) (fs.File, error) {
	return z.archive.Open(name)
}

func (z zipFS) ReadDir(name string) ([]fs.DirEntry, error) {
	var dirEntries []fs.DirEntry
	for _, file := range z.archive.File {
		cleanName := strings.TrimRight(file.Name, "/")
		if path.Dir(cleanName) == name {
			f, err := z.archive.Open(cleanName)
			if err != nil {
				return nil, err
			}
			fi, err := f.Stat()
			if err != nil {
				return nil, err
			}
			dirEntries = append(dirEntries, dirEntry{FileInfo: fi})
		}
	}
	return dirEntries, nil
}

func (z zipFS) Readlink(link string) (string, error) {
	f, err := z.archive.Open(link)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}

	if fi.Mode()&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: "readlink", Path: link, Err: fs.ErrInvalid}
	}

	bs, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(bs), nil
}

func (z zipFS) Stat(name string) (fs.FileInfo, error) {
	f, err := z.archive.Open(name)
	if err != nil {
		return nil, err
	}
	return f.Stat()
}

// BillyFilesystem is a template file accessor backed by a billy FS
type BillyFilesystem struct{ fs billy.Filesystem }

func NewBillyFilesystem(fs billy.Filesystem) BillyFilesystem {
	return BillyFilesystem{fs: fs}
}

func (b BillyFilesystem) Readlink(link string) (string, error) {
	return b.fs.Readlink(link)
}

type bfsFile struct {
	billy.File
	stats func() (fs.FileInfo, error)
}

func (b bfsFile) Stat() (fs.FileInfo, error) {
	return b.stats()
}

func (b BillyFilesystem) Open(name string) (fs.File, error) {
	f, err := b.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return bfsFile{
		File: f,
		stats: func() (fs.FileInfo, error) {
			return b.fs.Stat(name)
		},
	}, nil
}

type dirEntry struct {
	fs.FileInfo
}

func (d dirEntry) Type() fs.FileMode {
	return d.Mode().Type()
}

func (d dirEntry) Info() (fs.FileInfo, error) {
	return d, nil
}

func (b BillyFilesystem) ReadDir(name string) ([]fs.DirEntry, error) {
	fis, err := b.fs.ReadDir(name)
	if err != nil {
		return nil, err
	}
	des := make([]fs.DirEntry, len(fis))
	for i, fi := range fis {
		des[i] = dirEntry{fi}
	}
	return des, nil
}

func (b BillyFilesystem) Stat(name string) (fs.FileInfo, error) {
	return b.fs.Lstat(name)
}

// osFilesystem is a template file accessor backed by the os.
type osFilesystem struct{ root string }

func NewOsFilesystem(root string) osFilesystem {
	return osFilesystem{root: root}
}

func (o osFilesystem) Open(name string) (fs.File, error) {
	name = filepath.FromSlash(name)
	return os.Open(filepath.Join(o.root, name))
}

func (o osFilesystem) ReadDir(name string) ([]fs.DirEntry, error) {
	name = filepath.FromSlash(name)
	return os.ReadDir(filepath.Join(o.root, name))
}

func (o osFilesystem) Stat(name string) (fs.FileInfo, error) {
	name = filepath.FromSlash(name)
	return os.Lstat(filepath.Join(o.root, name))
}

func (o osFilesystem) Readlink(link string) (string, error) {
	link = filepath.FromSlash(link)
	return os.Readlink(filepath.Join(o.root, link))
}

// subFS exposes subdirectory of underlying FS, this is similar to `chroot`.
type subFS struct {
	root string
	fs   Filesystem
}

func NewSubFS(root string, fs Filesystem) subFS {
	return subFS{root: root, fs: fs}
}

func (o subFS) Open(name string) (fs.File, error) {
	return o.fs.Open(path.Join(o.root, name))
}

func (o subFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return o.fs.ReadDir(path.Join(o.root, name))
}

func (o subFS) Stat(name string) (fs.FileInfo, error) {
	return o.fs.Stat(path.Join(o.root, name))
}

func (o subFS) Readlink(link string) (string, error) {
	return o.fs.Readlink(path.Join(o.root, link))
}

type maskingFS struct {
	masked func(path string) bool
	fs     Filesystem
}

func NewMaskingFS(masked func(path string) bool, fs Filesystem) maskingFS {
	return maskingFS{masked: masked, fs: fs}
}

func (m maskingFS) Open(name string) (fs.File, error) {
	if m.masked(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return m.fs.Open(name)
}

func (m maskingFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if m.masked(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	des, err := m.fs.ReadDir(name)
	if err != nil {
		return nil, err
	}
	result := make([]fs.DirEntry, 0, len(des))
	for _, de := range des {
		if !m.masked(path.Join(name, de.Name())) {
			result = append(result, de)
		}
	}
	return result, nil
}

func (m maskingFS) Stat(name string) (fs.FileInfo, error) {
	if m.masked(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	return m.fs.Stat(name)
}

func (m maskingFS) Readlink(link string) (string, error) {
	if m.masked(link) {
		return "", &fs.PathError{Op: "readlink", Path: link, Err: fs.ErrNotExist}
	}
	return m.fs.Readlink(link)
}

const (
	EmbedSchema = "embed"
)

// UnionFS is an os.FS and embed.FS union fs.
// Files in embed.FS has the header "embed://", and files in os.FS don't have this header.
type UnionFS struct {
	embedFS embed.FS
}

func NewUnionFS(embedFS embed.FS) UnionFS {
	return UnionFS{
		embedFS: embedFS,
	}
}

func (u UnionFS) ReadFile(name string) ([]byte, error) {
	uri, _ := url.Parse(name)
	if uri != nil && uri.Scheme == EmbedSchema {
		name = strings.TrimLeft(uri.Opaque, "/")
		return u.embedFS.ReadFile(name)
	}

	return os.ReadFile(name)
}

func (u UnionFS) Open(name string) (fs.File, error) {
	uri, _ := url.Parse(name)
	if uri != nil && uri.Scheme == EmbedSchema {
		name = strings.TrimLeft(uri.Opaque, "/")
		return u.embedFS.Open(name)
	}

	return os.Open(name)
}

func (u UnionFS) ReadDir(name string) ([]fs.DirEntry, error) {
	uri, _ := url.Parse(name)
	if uri != nil && uri.Scheme == EmbedSchema {
		name = strings.TrimLeft(uri.Opaque, "/")
		return u.embedFS.ReadDir(name)
	}

	return os.ReadDir(name)
}

func (u UnionFS) Stat(name string) (fs.FileInfo, error) {
	f, err := u.Open(name)
	if err != nil {
		return nil, err
	}

	return f.Stat()
}

func (u UnionFS) Readlink(link string) (string, error) {
	uri, _ := url.Parse(link)
	if uri != nil && uri.Scheme == EmbedSchema {
		return "", errors.New("embed FS not support read link")
	}

	return os.Readlink(link)
}

// CopyFromFS copies files from the `src` dir on the accessor Filesystem to local filesystem into `dest` dir.
// The src path uses slashes as their separator.
// The dest path uses OS specific separator.
func CopyFromFS(root, dest string, fsys Filesystem) (err error) {
	// Walks the filesystem rooted at root.
	return fs.WalkDir(fsys, root, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		p, err := filepath.Rel(filepath.FromSlash(root), filepath.FromSlash(path))
		if err != nil {
			return err
		}

		dest := filepath.Join(dest, p)

		switch {
		case de.IsDir():
			// Ideally we should use the file mode of the src node
			// but it seems the git module is reporting directories
			// as 0644 instead of 0755. For now, just do it this way.
			// See https://github.com/go-git/go-git/issues/364
			// Upon resolution, return accessor.Stat(src).Mode()
			return os.MkdirAll(dest, 0o755)
		case de.Type()&fs.ModeSymlink != 0:
			var symlinkTarget string
			symlinkTarget, err = fsys.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(symlinkTarget, dest)
		case de.Type().IsRegular():
			fi, err := de.Info()
			if err != nil {
				return err
			}
			destFile, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fi.Mode())
			if err != nil {
				return err
			}
			defer destFile.Close()

			srcFile, err := fsys.Open(path)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			_, err = io.Copy(destFile, srcFile)
			return err
		default:
			return fmt.Errorf("unsuported file type: %s", de.Type().String())
		}
	})
}
