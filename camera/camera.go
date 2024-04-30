package camera

import (
	"github.com/bloeys/gglm/gglm"
)

type Type int32

const (
	Type_Unknown Type = iota
	Type_Perspective
	Type_Orthographic
)

type Camera struct {
	Type Type

	Pos     gglm.Vec3
	Forward gglm.Vec3
	WorldUp gglm.Vec3

	NearClip float32
	FarClip  float32

	// Perspective data
	Fov         float32
	AspectRatio float32

	// Ortho data
	Left, Right, Top, Bottom float32

	// Matrices
	ViewMat gglm.Mat4
	ProjMat gglm.Mat4
}

// Update recalculates view matrix and projection matrix.
// Should be called whenever a camera parameter changes
func (c *Camera) Update() {

	c.ViewMat = gglm.LookAtRH(&c.Pos, c.Pos.Clone().Add(&c.Forward), &c.WorldUp).Mat4

	if c.Type == Type_Perspective {
		c.ProjMat = *gglm.Perspective(c.Fov, c.AspectRatio, c.NearClip, c.FarClip)
	} else {
		c.ProjMat = gglm.Ortho(c.Left, c.Right, c.Top, c.Bottom, c.NearClip, c.FarClip).Mat4
	}
}

// UpdateRotation calculates a new forward vector and then calls camera.Update()
func (c *Camera) UpdateRotation(pitch, yaw float32) {

	dir := gglm.NewVec3(
		gglm.Cos32(yaw)*gglm.Cos32(pitch),
		gglm.Sin32(pitch),
		gglm.Sin32(yaw)*gglm.Cos32(pitch),
	)
	c.Forward = *dir.Normalize()
	c.Update()
}

func NewPerspective(pos, forward, worldUp *gglm.Vec3, nearClip, farClip, fovRadians, aspectRatio float32) Camera {

	cam := Camera{
		Type:    Type_Perspective,
		Pos:     *pos,
		Forward: *forward,
		WorldUp: *worldUp,

		NearClip: nearClip,
		FarClip:  farClip,

		Fov:         fovRadians,
		AspectRatio: aspectRatio,
	}
	cam.Update()

	return cam
}

func NewOrthographic(pos, forward, worldUp *gglm.Vec3, nearClip, farClip, left, right, top, bottom float32) Camera {

	cam := Camera{
		Type:    Type_Orthographic,
		Pos:     *pos,
		Forward: *forward,
		WorldUp: *worldUp,

		NearClip: nearClip,
		FarClip:  farClip,

		Left:   left,
		Right:  right,
		Top:    top,
		Bottom: bottom,
	}
	cam.Update()

	return cam
}
