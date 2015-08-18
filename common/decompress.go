package common



import (
	"archive/tar"
	"compress/gzip"
	log "github.com/Sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
)

func ExtractGZ(directory string, reader io.Reader) error {
	log.Info("Decompressing")


	gzReader, err := gzip.NewReader(reader)

	if err != nil {
		return err
	}
	return Extract(directory, gzReader)
}

func Extract(directory string, reader io.Reader)  error {
	tr := tar.NewReader(reader)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		} else if err != nil {
			log.Fatalln(err)
		}
		filename := filepath.Join(directory, hdr.Name)
		if hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA {
			file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, os.FileMode(hdr.Mode))
			io.Copy(file, tr)
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tr); err != nil {
				return err
			}
			file.Close()
		} else if hdr.Typeflag == tar.TypeDir {
			err := os.Mkdir(filename, 0777)
			if !os.IsExist(err) && err != nil {
				return err
			}
		} else if hdr.Typeflag == tar.TypeSymlink {
			if err := os.Symlink(hdr.Linkname, filename); err != nil {
				return err
			}
			// Hard link
		} else if hdr.Typeflag == tar.TypeLink {
			linkdest := filepath.Join(directory, hdr.Linkname)
			if err := os.Link(linkdest, filename); err != nil {
				return err
			}
		} else {
			log.Fatal("Experienced unknown tar file type: ", hdr.Typeflag)
		}
	}
	return nil
}