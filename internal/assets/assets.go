package assets

import _ "embed"

// Embedded JavaScript tracking library files
// These are compiled into the binary at build time

//go:embed pixel.umd.js
var PixelUMDJS []byte

//go:embed pixel.esm.js
var PixelESMJS []byte
