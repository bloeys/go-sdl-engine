package buffers

import (
	"github.com/bloeys/nmage/assert"
	"github.com/bloeys/nmage/logging"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type FramebufferAttachmentType int32

const (
	FramebufferAttachmentType_Unknown FramebufferAttachmentType = iota
	FramebufferAttachmentType_Texture
	FramebufferAttachmentType_Texture_Array
	FramebufferAttachmentType_Renderbuffer
	FramebufferAttachmentType_Cubemap
	FramebufferAttachmentType_Cubemap_Array
)

func (f FramebufferAttachmentType) IsValid() bool {

	switch f {
	case FramebufferAttachmentType_Texture:
		fallthrough
	case FramebufferAttachmentType_Texture_Array:
		fallthrough
	case FramebufferAttachmentType_Renderbuffer:
		fallthrough
	case FramebufferAttachmentType_Cubemap:
		fallthrough
	case FramebufferAttachmentType_Cubemap_Array:
		return true

	default:
		return false
	}
}

type FramebufferAttachmentDataFormat int32

const (
	FramebufferAttachmentDataFormat_Unknown FramebufferAttachmentDataFormat = iota
	FramebufferAttachmentDataFormat_R32Int
	FramebufferAttachmentDataFormat_RGBA8
	FramebufferAttachmentDataFormat_SRGBA
	FramebufferAttachmentDataFormat_DepthF32
	FramebufferAttachmentDataFormat_Depth24Stencil8
)

func (f FramebufferAttachmentDataFormat) IsColorFormat() bool {
	return f == FramebufferAttachmentDataFormat_R32Int ||
		f == FramebufferAttachmentDataFormat_RGBA8 ||
		f == FramebufferAttachmentDataFormat_SRGBA
}

func (f FramebufferAttachmentDataFormat) IsDepthFormat() bool {
	return f == FramebufferAttachmentDataFormat_Depth24Stencil8 ||
		f == FramebufferAttachmentDataFormat_DepthF32
}

func (f FramebufferAttachmentDataFormat) GlInternalFormat() int32 {

	switch f {
	case FramebufferAttachmentDataFormat_R32Int:
		return gl.R32I
	case FramebufferAttachmentDataFormat_RGBA8:
		return gl.RGB8
	case FramebufferAttachmentDataFormat_SRGBA:
		return gl.SRGB_ALPHA
	case FramebufferAttachmentDataFormat_DepthF32:
		return gl.DEPTH_COMPONENT
	case FramebufferAttachmentDataFormat_Depth24Stencil8:
		return gl.DEPTH24_STENCIL8
	default:
		logging.ErrLog.Fatalf("unknown framebuffer attachment data format. Format=%d\n", f)
		return 0
	}
}

func (f FramebufferAttachmentDataFormat) GlFormat() uint32 {

	switch f {
	case FramebufferAttachmentDataFormat_R32Int:
		return gl.RED_INTEGER

	case FramebufferAttachmentDataFormat_RGBA8:
		fallthrough
	case FramebufferAttachmentDataFormat_SRGBA:
		return gl.RGBA

	case FramebufferAttachmentDataFormat_DepthF32:
		return gl.DEPTH_COMPONENT

	case FramebufferAttachmentDataFormat_Depth24Stencil8:
		return gl.DEPTH_STENCIL

	default:
		logging.ErrLog.Fatalf("unknown framebuffer attachment data format. Format=%d\n", f)
		return 0
	}
}

type FramebufferAttachment struct {
	Id     uint32
	Type   FramebufferAttachmentType
	Format FramebufferAttachmentDataFormat
}

type Framebuffer struct {
	Id                    uint32
	ClearFlags            uint32
	Attachments           []FramebufferAttachment
	ColorAttachmentsCount uint32
	Width                 uint32
	Height                uint32
}

func (fbo *Framebuffer) Bind() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo.Id)
}

func (fbo *Framebuffer) BindWithViewport() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo.Id)
	gl.Viewport(0, 0, int32(fbo.Width), int32(fbo.Height))
}

// Clear calls gl.Clear with the fob's clear flags.
// Note that the fbo must be complete and bound.
// Calling this without a bound fbo will clear something else, like your screen.
func (fbo *Framebuffer) Clear() {
	gl.Clear(fbo.ClearFlags)
}

func (fbo *Framebuffer) UnBind() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

func (fbo *Framebuffer) UnBindWithViewport(width, height uint32) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, int32(width), int32(height))
}

// IsComplete returns true if OpenGL reports that the fbo is complete/usable.
// Note that this function binds and then unbinds the fbo
func (fbo *Framebuffer) IsComplete() bool {
	fbo.Bind()
	isComplete := gl.CheckFramebufferStatus(gl.FRAMEBUFFER) == gl.FRAMEBUFFER_COMPLETE
	fbo.UnBind()
	return isComplete
}

func (fbo *Framebuffer) HasColorAttachment() bool {
	return fbo.ColorAttachmentsCount > 0
}

func (fbo *Framebuffer) HasDepthAttachment() bool {

	for i := 0; i < len(fbo.Attachments); i++ {

		a := &fbo.Attachments[i]
		if a.Format.IsDepthFormat() {
			return true
		}
	}

	return false
}

func (fbo *Framebuffer) NewColorAttachment(
	attachType FramebufferAttachmentType,
	attachFormat FramebufferAttachmentDataFormat,
) {

	if fbo.ColorAttachmentsCount == 8 {
		logging.ErrLog.Fatalf("failed creating color attachment for framebuffer due it already having %d attached\n", fbo.ColorAttachmentsCount)
	}

	if !attachType.IsValid() {
		logging.ErrLog.Fatalf("failed creating color attachment for framebuffer due to unknown attachment type. Type=%d\n", attachType)
	}

	if attachType == FramebufferAttachmentType_Cubemap || attachType == FramebufferAttachmentType_Cubemap_Array {
		logging.ErrLog.Fatalf("failed creating color attachment because cubemaps can not be color attachments (at least in this implementation. You might be able to do it manually)\n")
	}

	if attachType == FramebufferAttachmentType_Texture_Array {
		logging.ErrLog.Fatalf("failed creating color attachment because texture arrays can not be color attachments (implementation can be updated to support it or you can do it manually)\n")
	}

	if !attachFormat.IsColorFormat() {
		logging.ErrLog.Fatalf("failed creating color attachment for framebuffer due to attachment data format not being a valid color type. Data format=%d\n", attachFormat)
	}

	a := FramebufferAttachment{
		Type:   attachType,
		Format: attachFormat,
	}

	fbo.Bind()

	if attachType == FramebufferAttachmentType_Texture {

		// Create texture
		gl.GenTextures(1, &a.Id)
		if a.Id == 0 {
			logging.ErrLog.Fatalf("failed to generate texture for framebuffer. GlError=%d\n", gl.GetError())
		}

		gl.BindTexture(gl.TEXTURE_2D, a.Id)
		gl.TexImage2D(gl.TEXTURE_2D, 0, attachFormat.GlInternalFormat(), int32(fbo.Width), int32(fbo.Height), 0, attachFormat.GlFormat(), gl.UNSIGNED_BYTE, nil)

		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.BindTexture(gl.TEXTURE_2D, 0)

		// Attach to fbo
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0+fbo.ColorAttachmentsCount, gl.TEXTURE_2D, a.Id, 0)

	} else if attachType == FramebufferAttachmentType_Renderbuffer {

		// Create rbo
		gl.GenRenderbuffers(1, &a.Id)
		if a.Id == 0 {
			logging.ErrLog.Fatalf("failed to generate render buffer for framebuffer. GlError=%d\n", gl.GetError())
		}

		gl.BindRenderbuffer(gl.RENDERBUFFER, a.Id)
		gl.RenderbufferStorage(gl.RENDERBUFFER, uint32(attachFormat.GlInternalFormat()), int32(fbo.Width), int32(fbo.Height))
		gl.BindRenderbuffer(gl.RENDERBUFFER, 0)

		// Attach to fbo
		gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0+fbo.ColorAttachmentsCount, gl.RENDERBUFFER, a.Id)
	}

	fbo.UnBind()
	fbo.ColorAttachmentsCount++
	fbo.ClearFlags |= gl.COLOR_BUFFER_BIT
	fbo.Attachments = append(fbo.Attachments, a)
}

// SetNoColorBuffer sets the read and draw buffers of this fbo to 'NONE',
// which tells the graphics driver that we don't want a color buffer for this fbo.
//
// This is required because normally an fbo must have a color buffer to be considered complete, but by
// doing this we get marked as complete even without one.
//
// Usually used when you only care about some other buffer, like a depth buffer.
func (fbo *Framebuffer) SetNoColorBuffer() {

	if fbo.HasColorAttachment() {
		logging.ErrLog.Fatalf("failed SetNoColorBuffer because framebuffer already has a color attachment\n")
	}

	fbo.Bind()
	gl.DrawBuffer(gl.NONE)
	gl.ReadBuffer(gl.NONE)
	fbo.UnBind()
}

func (fbo *Framebuffer) NewDepthAttachment(
	attachType FramebufferAttachmentType,
	attachFormat FramebufferAttachmentDataFormat,
) {

	if fbo.HasDepthAttachment() {
		logging.ErrLog.Fatalf("failed creating depth attachment for framebuffer because a depth attachment already exists\n")
	}

	if !attachType.IsValid() {
		logging.ErrLog.Fatalf("failed creating depth attachment for framebuffer due to unknown attachment type. Type=%d\n", attachType)
	}

	if !attachFormat.IsDepthFormat() {
		logging.ErrLog.Fatalf("failed creating depth attachment for framebuffer due to attachment data format not being a valid depth-stencil type. Data format=%d\n", attachFormat)
	}

	if attachType == FramebufferAttachmentType_Cubemap_Array {
		logging.ErrLog.Fatalf("failed creating cubemap array depth attachment because 'NewDepthCubemapArrayAttachment' must be used for that\n")
	}

	if attachType == FramebufferAttachmentType_Texture_Array {
		logging.ErrLog.Fatalf("failed creating texture array depth attachment because 'NewDepthTextureArrayAttachment' must be used for that\n")
	}

	a := FramebufferAttachment{
		Type:   attachType,
		Format: attachFormat,
	}

	fbo.Bind()

	if attachType == FramebufferAttachmentType_Texture {

		// Create texture
		gl.GenTextures(1, &a.Id)
		if a.Id == 0 {
			logging.ErrLog.Fatalf("failed to generate texture for framebuffer. GlError=%d\n", gl.GetError())
		}

		gl.BindTexture(gl.TEXTURE_2D, a.Id)
		gl.TexImage2D(gl.TEXTURE_2D, 0, attachFormat.GlInternalFormat(), int32(fbo.Width), int32(fbo.Height), 0, attachFormat.GlFormat(), gl.FLOAT, nil)

		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

		// This is so that any sampling outside the depth map gives a full depth value.
		// Useful for example when doing shadow maps where we want things outside
		// the range of the texture to not show shadow
		borderColor := []float32{1, 1, 1, 1}
		gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)

		gl.BindTexture(gl.TEXTURE_2D, 0)

		// Attach to fbo
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, a.Id, 0)

	} else if attachType == FramebufferAttachmentType_Renderbuffer {

		// Create rbo
		gl.GenRenderbuffers(1, &a.Id)
		if a.Id == 0 {
			logging.ErrLog.Fatalf("failed to generate render buffer for framebuffer. GlError=%d\n", gl.GetError())
		}

		gl.BindRenderbuffer(gl.RENDERBUFFER, a.Id)
		gl.RenderbufferStorage(gl.RENDERBUFFER, uint32(attachFormat.GlInternalFormat()), int32(fbo.Width), int32(fbo.Height))
		gl.BindRenderbuffer(gl.RENDERBUFFER, 0)

		// Attach to fbo
		gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, a.Id)

	} else if attachType == FramebufferAttachmentType_Cubemap {

		// Create cubemap
		gl.GenTextures(1, &a.Id)
		if a.Id == 0 {
			logging.ErrLog.Fatalf("failed to generate texture for framebuffer. GlError=%d\n", gl.GetError())
		}

		gl.BindTexture(gl.TEXTURE_CUBE_MAP, a.Id)
		for i := 0; i < 6; i++ {
			gl.TexImage2D(uint32(gl.TEXTURE_CUBE_MAP_POSITIVE_X+i), 0, attachFormat.GlInternalFormat(), int32(fbo.Width), int32(fbo.Height), 0, attachFormat.GlFormat(), gl.FLOAT, nil)
		}

		gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)

		gl.BindTexture(gl.TEXTURE_2D, 0)

		// Attach to fbo
		gl.FramebufferTexture(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, a.Id, 0)
	}

	fbo.UnBind()
	fbo.ClearFlags |= gl.DEPTH_BUFFER_BIT
	fbo.Attachments = append(fbo.Attachments, a)
}

func (fbo *Framebuffer) NewDepthCubemapArrayAttachment(
	attachFormat FramebufferAttachmentDataFormat,
	numCubemaps int32,
) {

	if fbo.HasDepthAttachment() {
		logging.ErrLog.Fatalf("failed creating cubemap array depth attachment for framebuffer because a depth attachment already exists\n")
	}

	if !attachFormat.IsDepthFormat() {
		logging.ErrLog.Fatalf("failed creating depth attachment for framebuffer due to attachment data format not being a valid depth-stencil type. Data format=%d\n", attachFormat)
	}

	a := FramebufferAttachment{
		Type:   FramebufferAttachmentType_Cubemap_Array,
		Format: attachFormat,
	}

	fbo.Bind()

	// Create cubemap array
	gl.GenTextures(1, &a.Id)
	if a.Id == 0 {
		logging.ErrLog.Fatalf("failed to generate texture for framebuffer. GlError=%d\n", gl.GetError())
	}

	gl.BindTexture(gl.TEXTURE_CUBE_MAP_ARRAY, a.Id)

	gl.TexImage3D(
		gl.TEXTURE_CUBE_MAP_ARRAY,
		0,
		attachFormat.GlInternalFormat(),
		int32(fbo.Width),
		int32(fbo.Height),
		6*numCubemaps,
		0,
		attachFormat.GlFormat(),
		gl.FLOAT,
		nil,
	)

	gl.TexParameteri(gl.TEXTURE_CUBE_MAP_ARRAY, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP_ARRAY, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP_ARRAY, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP_ARRAY, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP_ARRAY, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)

	gl.BindTexture(gl.TEXTURE_2D, 0)

	// Attach to fbo
	gl.FramebufferTexture(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, a.Id, 0)

	fbo.UnBind()
	fbo.ClearFlags |= gl.DEPTH_BUFFER_BIT
	fbo.Attachments = append(fbo.Attachments, a)
}

func (fbo *Framebuffer) NewDepthTextureArrayAttachment(
	attachFormat FramebufferAttachmentDataFormat,
	numTextures int32,
) {

	if fbo.HasDepthAttachment() {
		logging.ErrLog.Fatalf("failed creating texture array depth attachment for framebuffer because a depth attachment already exists\n")
	}

	if !attachFormat.IsDepthFormat() {
		logging.ErrLog.Fatalf("failed creating depth attachment for framebuffer due to attachment data format not being a valid depth-stencil type. Data format=%d\n", attachFormat)
	}

	a := FramebufferAttachment{
		Type:   FramebufferAttachmentType_Texture_Array,
		Format: attachFormat,
	}

	fbo.Bind()

	// Create cubemap array
	gl.GenTextures(1, &a.Id)
	if a.Id == 0 {
		logging.ErrLog.Fatalf("failed to generate texture for framebuffer. GlError=%d\n", gl.GetError())
	}

	gl.BindTexture(gl.TEXTURE_2D_ARRAY, a.Id)

	gl.TexImage3D(
		gl.TEXTURE_2D_ARRAY,
		0,
		attachFormat.GlInternalFormat(),
		int32(fbo.Width),
		int32(fbo.Height),
		numTextures,
		0,
		attachFormat.GlFormat(),
		gl.FLOAT,
		nil,
	)

	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	// This is so that any sampling outside the depth map gives a full depth value.
	// Useful for example when doing shadow maps where we want things outside
	// the range of the texture to not show shadow
	borderColor := []float32{1, 1, 1, 1}
	gl.TexParameterfv(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_BORDER_COLOR, &borderColor[0])
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)

	gl.BindTexture(gl.TEXTURE_2D_ARRAY, 0)

	// Attach to fbo
	gl.FramebufferTexture(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, a.Id, 0)

	fbo.UnBind()
	fbo.ClearFlags |= gl.DEPTH_BUFFER_BIT
	fbo.Attachments = append(fbo.Attachments, a)
}

func (fbo *Framebuffer) NewDepthStencilAttachment(
	attachType FramebufferAttachmentType,
	attachFormat FramebufferAttachmentDataFormat,
) {

	if fbo.HasDepthAttachment() {
		logging.ErrLog.Fatalf("failed creating depth-stencil attachment for framebuffer because a depth-stencil attachment already exists\n")
	}

	if !attachType.IsValid() {
		logging.ErrLog.Fatalf("failed creating depth-stencil attachment for framebuffer due to unknown attachment type. Type=%d\n", attachType)
	}

	if !attachFormat.IsDepthFormat() {
		logging.ErrLog.Fatalf("failed creating depth-stencil attachment for framebuffer due to attachment data format not being a valid depth-stencil type. Data format=%d\n", attachFormat)
	}

	a := FramebufferAttachment{
		Type:   attachType,
		Format: attachFormat,
	}

	fbo.Bind()

	if attachType == FramebufferAttachmentType_Texture {

		// Create texture
		gl.GenTextures(1, &a.Id)
		if a.Id == 0 {
			logging.ErrLog.Fatalf("failed to generate texture for framebuffer. GlError=%d\n", gl.GetError())
		}

		gl.BindTexture(gl.TEXTURE_2D, a.Id)
		gl.TexImage2D(gl.TEXTURE_2D, 0, attachFormat.GlInternalFormat(), int32(fbo.Width), int32(fbo.Height), 0, attachFormat.GlFormat(), gl.UNSIGNED_INT_24_8, nil)

		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.BindTexture(gl.TEXTURE_2D, 0)

		// Attach to fbo
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_STENCIL_ATTACHMENT, gl.TEXTURE_2D, a.Id, 0)

	} else if attachType == FramebufferAttachmentType_Renderbuffer {

		// Create rbo
		gl.GenRenderbuffers(1, &a.Id)
		if a.Id == 0 {
			logging.ErrLog.Fatalf("failed to generate render buffer for framebuffer. GlError=%d\n", gl.GetError())
		}

		gl.BindRenderbuffer(gl.RENDERBUFFER, a.Id)
		gl.RenderbufferStorage(gl.RENDERBUFFER, uint32(attachFormat.GlInternalFormat()), int32(fbo.Width), int32(fbo.Height))
		gl.BindRenderbuffer(gl.RENDERBUFFER, 0)

		// Attach to fbo
		gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_STENCIL_ATTACHMENT, gl.RENDERBUFFER, a.Id)
	}

	fbo.UnBind()
	fbo.ClearFlags |= gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT
	fbo.Attachments = append(fbo.Attachments, a)
}

// SetCubemapArrayLayerFace 'binds' a single face of a cubemap from the cubemap
// array to the fbo, such that rendering only affects that one face and the others inaccessible.
//
// If this is not called, the default is that the entire cubemap array and all the faces in it
// are bound and available for use when binding the fbo.
func (fbo *Framebuffer) SetCubemapArrayLayerFace(layerFace int32) {

	for i := 0; i < len(fbo.Attachments); i++ {

		a := &fbo.Attachments[i]
		if a.Type != FramebufferAttachmentType_Cubemap_Array {
			continue
		}

		assert.T(a.Format.IsDepthFormat(), "SetCubemapFromArray called but a cubemap array is set on a color attachment, which is not currently handled. Code must be updated!")
		gl.FramebufferTextureLayer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, a.Id, 0, layerFace)
		return
	}

	logging.ErrLog.Fatalf("SetCubemapFromArray failed because no cubemap array attachment was found on fbo. Fbo=%+v\n", *fbo)
}

func (fbo *Framebuffer) Delete() {

	if fbo.Id == 0 {
		return
	}

	gl.DeleteFramebuffers(1, &fbo.Id)
	fbo.Id = 0
}

func NewFramebuffer(width, height uint32) Framebuffer {

	// It is allowed to have attachments of differnt sizes in one FBO,
	// but that complicates things (e.g. which size to use for gl.viewport) and I don't see much use
	// for it now, so we will have all attachments share size
	fbo := Framebuffer{
		Width:  width,
		Height: height,
	}

	gl.GenFramebuffers(1, &fbo.Id)
	if fbo.Id == 0 {
		logging.ErrLog.Fatalf("failed to generate framebuffer. GlError=%d\n", gl.GetError())
	}

	return fbo
}
