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
	BoundVao  *buffers.VertexArray
	BoundMesh *meshes.Mesh
	BoundMat  *materials.Material
}

func (r *Rend3DGL) DrawMesh(mesh *meshes.Mesh, modelMat *gglm.TrMat, mat *materials.Material) {

	if mesh != r.BoundMesh {
		mesh.Vao.Bind()
		r.BoundMesh = mesh
	}

	if mat != r.BoundMat {
		mat.Bind()
		r.BoundMat = mat
	}

	if mat.Settings.Has(materials.MaterialSettings_HasModelMat) {
		mat.SetUnifMat4("modelMat", &modelMat.Mat4)
	}

	if mat.Settings.Has(materials.MaterialSettings_HasNormalMat) {
		normalMat := modelMat.Clone().InvertAndTranspose().ToMat3()
		mat.SetUnifMat3("normalMat", &normalMat)
	}

	for i := 0; i < len(mesh.SubMeshes); i++ {
		gl.DrawElementsBaseVertexWithOffset(gl.TRIANGLES, mesh.SubMeshes[i].IndexCount, gl.UNSIGNED_INT, uintptr(mesh.SubMeshes[i].BaseIndex), mesh.SubMeshes[i].BaseVertex)
	}
}

func (r *Rend3DGL) DrawVertexArray(mat *materials.Material, vao *buffers.VertexArray, firstElement int32, elementCount int32) {

	if vao != r.BoundVao {
		vao.Bind()
		r.BoundVao = vao
	}

	if mat != r.BoundMat {
		mat.Bind()
		r.BoundMat = mat
	}

	gl.DrawArrays(gl.TRIANGLES, firstElement, elementCount)
}

func (r *Rend3DGL) DrawCubemap(mesh *meshes.Mesh, mat *materials.Material) {

	if mesh != r.BoundMesh {
		mesh.Vao.Bind()
		r.BoundMesh = mesh
	}

	if mat != r.BoundMat {
		mat.Bind()
		r.BoundMat = mat
	}

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
