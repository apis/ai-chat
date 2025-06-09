package staticAssets

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
)

type pseudoFs func(name string) (fs.File, error)

func (f pseudoFs) Open(name string) (fs.File, error) {
	return f(name)
}

func Handler(embedFs embed.FS, embedFsRoot string, urlPrefix string, defaultUrl string) http.Handler {
	handler := pseudoFs(func(name string) (fs.File, error) {
		assetPath := path.Join(embedFsRoot, name)

		file, err := embedFs.Open(assetPath)
		if os.IsNotExist(err) {
			return embedFs.Open(path.Join(embedFsRoot, defaultUrl))
		}

		return file, err
	})

	return http.StripPrefix(urlPrefix, http.FileServer(http.FS(handler)))
}
