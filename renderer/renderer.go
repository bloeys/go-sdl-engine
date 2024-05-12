package renderer

import (
	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/nmage/buffers"
	"github.com/bloeys/nmage/materials"
	"github.com/bloeys/nmage/meshes"
)

type Render interface {
	DrawMesh(mesh meshes.Mesh, trMat gglm.TrMat, mat materials.Material)
	DrawVertexArray(mat materials.Material, vao buffers.VertexArray, firstElement int32, count int32)
	DrawCubemap(mesh meshes.Mesh, mat materials.Material)
	FrameEnd()
}
