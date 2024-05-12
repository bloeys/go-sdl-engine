package rend3dgl

import (
	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/nmage/buffers"
	"github.com/bloeys/nmage/materials"
	"github.com/bloeys/nmage/meshes"
	"github.com/bloeys/nmage/renderer"
	"github.com/go-gl/gl/v4.1-core/gl"
)

var _ renderer.Render = &Rend3DGL{}

type Rend3DGL struct {
	BoundVaoId     uint32
	BoundMatId     uint32
	BoundMeshVaoId uint32
}

func (r *Rend3DGL) DrawMesh(mesh meshes.Mesh, modelMat gglm.TrMat, mat materials.Material) {

	if mesh.Vao.Id != r.BoundMeshVaoId {
		mesh.Vao.Bind()
		r.BoundMeshVaoId = mesh.Vao.Id
	}

	if mat.Id != r.BoundMatId {
		mat.Bind()
		r.BoundMatId = mat.Id
	}

	if mat.Settings.Has(materials.MaterialSettings_HasModelMtx) {
		mat.SetUnifMat4("modelMat", modelMat.Mat4)
	}

	if mat.Settings.Has(materials.MaterialSettings_HasNormalMtx) {
		normalMat := modelMat.Clone().InvertAndTranspose().ToMat3()
		mat.SetUnifMat3("normalMat", normalMat)
	}

	for i := 0; i < len(mesh.SubMeshes); i++ {
		gl.DrawElementsBaseVertexWithOffset(gl.TRIANGLES, mesh.SubMeshes[i].IndexCount, gl.UNSIGNED_INT, uintptr(mesh.SubMeshes[i].BaseIndex), mesh.SubMeshes[i].BaseVertex)
	}
}

func (r *Rend3DGL) DrawVertexArray(mat materials.Material, vao buffers.VertexArray, firstElement int32, elementCount int32) {

	if vao.Id != r.BoundVaoId {
		vao.Bind()
		r.BoundVaoId = vao.Id
	}

	if mat.Id != r.BoundMatId {
		mat.Bind()
		r.BoundMatId = mat.Id
	}

	gl.DrawArrays(gl.TRIANGLES, firstElement, elementCount)
}

func (r *Rend3DGL) DrawCubemap(mesh meshes.Mesh, mat materials.Material) {

	if mesh.Vao.Id != r.BoundMeshVaoId {
		mesh.Vao.Bind()
		r.BoundMeshVaoId = mesh.Vao.Id
	}

	if mat.Id != r.BoundMatId {
		mat.Bind()
		r.BoundMatId = mat.Id
	}

	for i := 0; i < len(mesh.SubMeshes); i++ {
		gl.DrawElementsBaseVertexWithOffset(gl.TRIANGLES, mesh.SubMeshes[i].IndexCount, gl.UNSIGNED_INT, uintptr(mesh.SubMeshes[i].BaseIndex), mesh.SubMeshes[i].BaseVertex)
	}
}

func (r3d *Rend3DGL) FrameEnd() {
	r3d.BoundVaoId = 0
	r3d.BoundMatId = 0
	r3d.BoundMeshVaoId = 0
}

func NewRend3DGL() *Rend3DGL {
	return &Rend3DGL{}
}
