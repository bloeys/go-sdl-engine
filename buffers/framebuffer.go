package buffers

import (
	"github.com/bloeys/nmage/logging"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type FramebufferAttachmentType int32

const (
	FramebufferAttachmentType_Unknown FramebufferAttachmentType = iota
	FramebufferAttachmentType_Texture
	FramebufferAttachmentType_Renderbuffer
)

func (f FramebufferAttachmentType) IsValid() bool {

	switch f {
	case FramebufferAttachmentType_Texture:
		fallthrough
	case FramebufferAttachmentType_Renderbuffer:
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
	FramebufferAttachmentDataFormat_Depth24Stencil8
)

func (f FramebufferAttachmentDataFormat) IsColorFormat() bool {
	return f == FramebufferAttachmentDataFormat_R32Int ||
		f == FramebufferAttachmentDataFormat_RGBA8 ||
		f == FramebufferAttachmentDataFormat_SRGBA
}

func (f FramebufferAttachmentDataFormat) IsDepthFormat() bool {
	return f == FramebufferAttachmentDataFormat_Depth24Stencil8
}

func (f FramebufferAttachmentDataFormat) GlInternalFormat() int32 {

	switch f {
	case FramebufferAttachmentDataFormat_R32Int:
		return gl.R32I
	case FramebufferAttachmentDataFormat_RGBA8:
		return gl.RGB8
	case FramebufferAttachmentDataFormat_SRGBA:
		return gl.SRGB_ALPHA
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
	fbo.Attachments = append(fbo.Attachments, a)
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
