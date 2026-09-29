package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var uiStaticRaw embed.FS

// uiStaticFS is the mounted subtree served at /ui/. The dist folder is the
// Vue3 SPA produced by `npm run build` in web/dist (the bootstrap path
// includes a self-contained CDN-based build so the simulator is runnable
// without Node installed).
var uiStaticFS, _ = fs.Sub(uiStaticRaw, "dist")
