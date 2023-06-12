package archiving

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type ZipArchive struct {
	out *zip.Writer
	f   *os.File
}

func NewZipArchive(dest string) (*ZipArchive, error) {
	output, err := os.Create(dest)
	if err != nil {
		return nil, fmt.Errorf("Failed to create output file:", err)
	}

	return &ZipArchive{
		f:   output,
		out: zip.NewWriter(output),
	}, nil
}

func (z *ZipArchive) Finish() {
	z.out.Close()
	z.f.Close()
}

// Receives a map of path outside the zip => path inside zip
func (z *ZipArchive) CompressFile(src, dest string, info fs.FileInfo) error {
	if !info.IsDir() {
		// Open the file for reading.
		file, err := os.Open(src)
		if err != nil {
			return err
		}
		defer file.Close()

		// Create a new file in the zip archive.
		zipFile, err := z.out.Create(dest)
		if err != nil {
			return err
		}

		// Copy the contents of the file to the zip archive.
		_, err = io.Copy(zipFile, file)
		if err != nil {
			return err
		}
	}

	return nil
}

// unzip extracts a zip archive to the specified destination directory.
func ExtractZip(src string, dst string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	zipReader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		target := filepath.Join(dst, file.Name)

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, file.Mode()); err != nil {
				return err
			}
			continue
		}

		outfile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer outfile.Close()

		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		if _, err := io.Copy(outfile, rc); err != nil {
			return err
		}
	}

	return nil
}
