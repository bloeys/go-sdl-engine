package rend3dgl

import (
	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/nmage/materials"
	"github.com/bloeys/nmage/meshes"
	"github.com/bloeys/nmage/renderer"
	"github.com/go-gl/gl/v4.1-core/gl"
)

var _ renderer.Render = &Rend3DGL{}

type Rend3DGL struct {
	BoundMesh *meshes.Mesh
	BoundMat  *materials.Material
}

func (r3d *Rend3DGL) Draw(mesh *meshes.Mesh, trMat *gglm.TrMat, mat *materials.Material) {

	if mesh != r3d.BoundMesh {
		mesh.Vao.Bind()
		r3d.BoundMesh = mesh
	}

	if mat != r3d.BoundMat {
		mat.Bind()
		r3d.BoundMat = mat
	}

	mat.SetUnifMat4("modelMat", &trMat.Mat4)

	for i := 0; i < len(mesh.SubMeshes); i++ {
		gl.DrawElementsBaseVertexWithOffset(gl.TRIANGLES, mesh.SubMeshes[i].IndexCount, gl.UNSIGNED_INT, uintptr(mesh.SubMeshes[i].BaseIndex), mesh.SubMeshes[i].BaseVertex)
	}
}

func (r3d *Rend3DGL) FrameEnd() {
	r3d.BoundMesh = nil
	r3d.BoundMat = nil
}

func NewRend3DGL() *Rend3DGL {
	return &Rend3DGL{}
}
