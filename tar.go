package archiving

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type TarArchive struct {
	out *tar.Writer
	f   *os.File
}

func NewTarArchive(dest string) (*TarArchive, error) {
	output, err := os.Create(dest)
	if err != nil {
		return nil, fmt.Errorf("Failed to create output file: %w", err)
	}

	return &TarArchive{
		f:   output,
		out: tar.NewWriter(output),
	}, nil
}

func (t *TarArchive) Finish() {
	t.out.Close()
	t.f.Close()
}

func (t *TarArchive) CompressFile(src, dest string, info fs.FileInfo) error {
	if !info.IsDir() {
		// Create a new tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = dest

		// Write the header to the tar file
		if err := t.out.WriteHeader(header); err != nil {
			return err
		}

		// If it's not a directory, write the file contents to the tar file
		file, err := os.Open(src)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(t.out, file)
		if err != nil {
			return err
		}
	}

	return nil
}

// untar extracts a tar archive to the specified destination directory.
func ExtractTar(src string, dst string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	tarReader := tar.NewReader(file)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			outfile, err := os.Create(target)
			if err != nil {
				return err
			}
			defer outfile.Close()
			if _, err := io.Copy(outfile, tarReader); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown type: %v in %s", header.Typeflag, header.Name)
		}
	}

	return nil
}
