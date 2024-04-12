package assets

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/mandykoh/prism"
)

type ColorFormat int

const (
	ColorFormat_RGBA8 ColorFormat = iota
)

type Texture struct {
	// Path only exists for textures loaded from disk
	Path string

	TexID uint32

	// Width is the width of the texture in pixels (pixels per row).
	// Note that the number of bytes constituting a row is MORE than this (e.g. for RGBA8, bytesPerRow=width*4, since we have 4 bytes per pixel)
	Width int32

	// Height is the height of the texture in pixels (pixels per column).
	// Note that the number of bytes constituting a column is MORE than this (e.g. for RGBA8, bytesPerColumn=height*4, since we have 4 bytes per pixel)
	Height int32

	// Pixels usually stored in RGBA format
	Pixels []byte
}

type TextureLoadOptions struct {
	TryLoadFromCache bool
	WriteToCache     bool
	GenMipMaps       bool
	KeepPixelsInMem  bool
	NoSrgba          bool
}

type Cubemap struct {
	// These only exists for textures loaded from disk
	RightPath string
	LeftPath  string
	TopPath   string
	BotPath   string
	FrontPath string
	BackPath  string
	TexID     uint32
}

func LoadTexturePNG(file string, loadOptions *TextureLoadOptions) (Texture, error) {

	if loadOptions == nil {
		loadOptions = &TextureLoadOptions{}
	}

	if loadOptions.TryLoadFromCache {
		if tex, ok := GetTextureFromCachePath(file); ok {
			return tex, nil
		}
	}

	//Load from disk
	fileBytes, err := os.ReadFile(file)
	if err != nil {
		return Texture{}, err
	}

	bytesReader := bytes.NewReader(fileBytes)
	img, err := png.Decode(bytesReader)
	if err != nil {
		return Texture{}, err
	}

	nrgbaImg := prism.ConvertImageToNRGBA(img, 2)
	tex := Texture{
		Path:   file,
		Pixels: nrgbaImg.Pix,
		Width:  int32(nrgbaImg.Bounds().Dx()),
		Height: int32(nrgbaImg.Bounds().Dy()),
	}
	flipImgPixelsVertically(tex.Pixels, int(tex.Width), int(tex.Height), 4)

	//Prepare opengl stuff
	gl.GenTextures(1, &tex.TexID)
	gl.BindTexture(gl.TEXTURE_2D, tex.TexID)

	// set the texture wrapping/filtering options (on the currently bound texture object)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	// load and generate the texture
	internalFormat := int32(gl.SRGB_ALPHA)
	if loadOptions.NoSrgba {
		internalFormat = gl.RGBA8
	}

	gl.TexImage2D(gl.TEXTURE_2D, 0, internalFormat, tex.Width, tex.Height, 0, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&tex.Pixels[0]))

	if loadOptions.GenMipMaps {
		gl.GenerateMipmap(tex.TexID)
	}

	if loadOptions.WriteToCache {
		AddTextureToCache(tex)
	}

	if !loadOptions.KeepPixelsInMem {
		tex.Pixels = nil
	}

	return tex, nil
}

func LoadTextureInMemPngImg(img image.Image, loadOptions *TextureLoadOptions) (Texture, error) {

	if loadOptions == nil {
		loadOptions = &TextureLoadOptions{}
	}

	nrgbaImg := prism.ConvertImageToNRGBA(img, 2)
	tex := Texture{
		Path:   "",
		Pixels: nrgbaImg.Pix,
		Height: int32(nrgbaImg.Bounds().Dy()),
		Width:  int32(nrgbaImg.Bounds().Dx()),
	}
	flipImgPixelsVertically(tex.Pixels, int(tex.Width), int(tex.Height), 4)

	//Prepare opengl stuff
	gl.GenTextures(1, &tex.TexID)
	gl.BindTexture(gl.TEXTURE_2D, tex.TexID)

	// set the texture wrapping/filtering options (on the currently bound texture object)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	// load and generate the texture
	internalFormat := int32(gl.SRGB_ALPHA)
	if loadOptions.NoSrgba {
		internalFormat = gl.RGBA8
	}

	gl.TexImage2D(gl.TEXTURE_2D, 0, internalFormat, tex.Width, tex.Height, 0, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&tex.Pixels[0]))

	if loadOptions.GenMipMaps {
		gl.GenerateMipmap(tex.TexID)
	}

	if loadOptions.WriteToCache {
		AddTextureToCache(tex)
	}

	if !loadOptions.KeepPixelsInMem {
		tex.Pixels = nil
	}

	return tex, nil
}

func LoadTextureJpeg(file string, loadOptions *TextureLoadOptions) (Texture, error) {

	if loadOptions == nil {
		loadOptions = &TextureLoadOptions{}
	}

	if loadOptions.TryLoadFromCache {
		if tex, ok := GetTextureFromCachePath(file); ok {
			return tex, nil
		}
	}

	//Load from disk
	fileBytes, err := os.ReadFile(file)
	if err != nil {
		return Texture{}, err
	}

	img, err := jpeg.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return Texture{}, err
	}

	nrgbaImg := prism.ConvertImageToNRGBA(img, 2)
	tex := Texture{
		Path:   file,
		Pixels: nrgbaImg.Pix,
		Height: int32(nrgbaImg.Bounds().Dy()),
		Width:  int32(nrgbaImg.Bounds().Dx()),
	}
	flipImgPixelsVertically(tex.Pixels, int(tex.Width), int(tex.Height), 4)

	//Prepare opengl stuff
	gl.GenTextures(1, &tex.TexID)
	gl.BindTexture(gl.TEXTURE_2D, tex.TexID)

	// set the texture wrapping/filtering options (on the currently bound texture object)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	// load and generate the texture
	internalFormat := int32(gl.SRGB_ALPHA)
	if loadOptions.NoSrgba {
		internalFormat = gl.RGBA8
	}

	gl.TexImage2D(gl.TEXTURE_2D, 0, internalFormat, tex.Width, tex.Height, 0, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&tex.Pixels[0]))

	if loadOptions.GenMipMaps {
		gl.GenerateMipmap(tex.TexID)
	}

	if loadOptions.WriteToCache {
		AddTextureToCache(tex)
	}

	if !loadOptions.KeepPixelsInMem {
		tex.Pixels = nil
	}

	return tex, nil
}

// LoadCubemapTextures only supports the 'TextureIsSrgba' option
func LoadCubemapTextures(rightTex, leftTex, topTex, botTex, frontTex, backTex string, loadOptions *TextureLoadOptions) (Cubemap, error) {

	if loadOptions == nil {
		loadOptions = &TextureLoadOptions{}
	}

	var imgDecoder func(r io.Reader) (image.Image, error)
	ext := strings.ToLower(path.Ext(rightTex))
	if ext == ".jpg" || ext == ".jpeg" {
		imgDecoder = jpeg.Decode
	} else if ext == ".png" {
		imgDecoder = png.Decode
	} else {
		return Cubemap{}, fmt.Errorf("unknown image extension: %s. Expected one of: .jpg, .jpeg, .png", ext)
	}

	cmap := Cubemap{
		RightPath: rightTex,
		LeftPath:  leftTex,
		TopPath:   topTex,
		BotPath:   botTex,
		FrontPath: frontTex,
		BackPath:  backTex,
	}

	gl.GenTextures(1, &cmap.TexID)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, cmap.TexID)

	// The order here matters
	texturePaths := []string{rightTex, leftTex, topTex, botTex, frontTex, backTex}
	for i := uint32(0); i < uint32(len(texturePaths)); i++ {

		fPath := texturePaths[i]

		//Load from disk
		fileBytes, err := os.ReadFile(fPath)
		if err != nil {
			return Cubemap{}, err
		}

		img, err := imgDecoder(bytes.NewReader(fileBytes))
		if err != nil {
			return Cubemap{}, err
		}

		nrgbaImg := prism.ConvertImageToNRGBA(img, 2)
		height := int32(nrgbaImg.Bounds().Dy())
		width := int32(nrgbaImg.Bounds().Dx())

		internalFormat := int32(gl.SRGB_ALPHA)
		if loadOptions.NoSrgba {
			internalFormat = gl.RGBA8
		}

		gl.TexImage2D(uint32(gl.TEXTURE_CUBE_MAP_POSITIVE_X)+i, 0, internalFormat, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&nrgbaImg.Pix[0]))
	}

	// set the texture wrapping/filtering options (on the currently bound texture object)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)

	return cmap, nil
}

func flipImgPixelsVertically(bytes []byte, width, height, bytesPerPixel int) {

	// Flip the image vertically such that (e.g. in an image of 10 rows) rows 0<->9, 1<->8, 2<->7 etc are swapped.
	// We do this because images are usually stored top-left to bottom-right, while opengl stores textures bottom-left to top-right, so if we don't swap
	// rows textures will appear inverted
	widthInBytes := width * bytesPerPixel
	rowData := make([]byte, width*bytesPerPixel)
	for rowNum := 0; rowNum < height/2; rowNum++ {

		upperRowStartIndex := rowNum * widthInBytes
		lowerRowStartIndex := (height - rowNum - 1) * widthInBytes
		copy(rowData, bytes[upperRowStartIndex:upperRowStartIndex+widthInBytes])
		copy(bytes[upperRowStartIndex:upperRowStartIndex+widthInBytes], bytes[lowerRowStartIndex:lowerRowStartIndex+widthInBytes])
		copy(bytes[lowerRowStartIndex:lowerRowStartIndex+widthInBytes], rowData)

	}
}
