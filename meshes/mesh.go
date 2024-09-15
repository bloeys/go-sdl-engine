package meshes

import (
	"errors"

	"github.com/bloeys/assimp-go/asig"
	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/nmage/assert"
	"github.com/bloeys/nmage/buffers"
)

type SubMesh struct {
	BaseVertex int32
	BaseIndex  uint32
	IndexCount int32
}

type Mesh struct {
	Name string
	/*
		Vao has the following shader attribute layout:
			- Loc0: Pos
			- Loc1: Normal
			- Loc2: UV0
			- Loc3: Tangent
			- (Optional) Color

		Optional stuff appear in the order in this list, depending on what other optional stuff exists.

		For example:
			- If color exists it will be in Loc3, otherwise it is unset
	*/
	Vao       buffers.VertexArray
	SubMeshes []SubMesh
}

var (
	// DefaultMeshLoadFlags are the flags always applied when loading a new mesh regardless
	// of what post process flags are used when loading a mesh.
	//
	// Defaults to: asig.PostProcessTriangulate | asig.PostProcessCalcTangentSpace;
	// Note: changing this will break the normal lit shaders, which expect tangents to be there
	DefaultMeshLoadFlags asig.PostProcess = asig.PostProcessTriangulate | asig.PostProcessCalcTangentSpace
)

func NewMesh(name, modelPath string, postProcessFlags asig.PostProcess) (Mesh, error) {

	finalPostProcessFlags := DefaultMeshLoadFlags | postProcessFlags

	scene, release, err := asig.ImportFile(modelPath, finalPostProcessFlags)
	if err != nil {
		return Mesh{}, errors.New("Failed to load model. Err: " + err.Error())
	}
	defer release()

	if len(scene.Meshes) == 0 {
		return Mesh{}, errors.New("No meshes found in file: " + modelPath)
	}

	mesh := Mesh{
		Name:      name,
		Vao:       buffers.NewVertexArray(),
		SubMeshes: make([]SubMesh, 0, 1),
	}

	vbo := buffers.NewVertexBuffer()
	ibo := buffers.NewIndexBuffer()

	// Estimate a useful prealloc capacity based on the first submesh that has vertex pos+normals+tangents+texCoords
	vertexBufDataCapacity := len(scene.Meshes[0].Vertices) * 3 * 3 * 3 * 2

	// Increase capacity depending on what the mesh has
	if len(scene.Meshes[0].ColorSets) > 0 && len(scene.Meshes[0].ColorSets[0]) > 0 {
		vertexBufDataCapacity *= 4
	}

	var vertexBufData []float32 = make([]float32, 0, vertexBufDataCapacity)

	// Initial size assumes 3 indices per face
	var indexBufData []uint32 = make([]uint32, 0, len(scene.Meshes[0].Faces)*3)

	// fmt.Printf("\nMesh %s has %d meshe(s) with first mesh having %d vertices\n", name, len(scene.Meshes), len(scene.Meshes[0].Vertices))

	for i := 0; i < len(scene.Meshes); i++ {

		sceneMesh := scene.Meshes[i]

		// We always want tangents and UV0
		if len(sceneMesh.Tangents) == 0 {
			sceneMesh.Tangents = make([]gglm.Vec3, len(sceneMesh.Vertices))
		}

		if len(sceneMesh.TexCoords[0]) == 0 {
			sceneMesh.TexCoords[0] = make([]gglm.Vec3, len(sceneMesh.Vertices))
		}

		hasColorSet0 := len(sceneMesh.ColorSets) > 0 && len(sceneMesh.ColorSets[0]) > 0

		layoutToUse := []buffers.Element{
			{ElementType: buffers.DataTypeVec3}, // Position
			{ElementType: buffers.DataTypeVec3}, // Normals
			{ElementType: buffers.DataTypeVec3}, // Tangents
			{ElementType: buffers.DataTypeVec2}, // UV0
		}

		if hasColorSet0 {
			layoutToUse = append(layoutToUse, buffers.Element{ElementType: buffers.DataTypeVec4})
		}

		if i == 0 {
			vbo.SetLayout(layoutToUse...)
		} else {

			// @TODO @NOTE: This requirement is because we are using one VAO+VBO for all
			// the meshes and so the buffer must have one format.
			//
			// If we want to allow different layouts then we can simply create one vbo per layout and put
			// meshes of the same layout in the same vbo, and we store the index of the vbo the mesh
			// uses in the submesh struct.
			firstSubmeshLayout := vbo.GetLayout()
			assert.T(len(firstSubmeshLayout) == len(layoutToUse), "Vertex layout of submesh '%d' of mesh '%s' at path '%s' does not equal vertex layout of the first submesh. Original layout: %v; This layout: %v", i, name, modelPath, firstSubmeshLayout, layoutToUse)

			for i := 0; i < len(firstSubmeshLayout); i++ {
				assert.T(firstSubmeshLayout[i].ElementType == layoutToUse[i].ElementType, "Vertex layout of submesh '%d' of mesh '%s' at path '%s' does not equal vertex layout of the first submesh. Original layout: %v; This layout: %v", i, name, modelPath, firstSubmeshLayout, layoutToUse)
			}
		}

		arrs := []arrToInterleave{
			{V3s: sceneMesh.Vertices},
			{V3s: sceneMesh.Normals},
			{V3s: sceneMesh.Tangents},
			{V2s: v3sToV2s(sceneMesh.TexCoords[0])},
		}

		if hasColorSet0 {
			arrs = append(arrs, arrToInterleave{V4s: sceneMesh.ColorSets[0]})
		}

		indices := flattenFaces(sceneMesh.Faces)
		mesh.SubMeshes = append(mesh.SubMeshes, SubMesh{

			// Index of the vertex to start from (e.g. if index buffer says use vertex 5, and BaseVertex=3, the vertex used will be vertex 8)
			BaseVertex: int32(len(vertexBufData)*4) / vbo.Stride,
			// Which index (in the index buffer) to start from
			BaseIndex: uint32(len(indexBufData)),
			// How many indices in this submesh
			IndexCount: int32(len(indices)),
		})

		vertexBufData = append(vertexBufData, interleave(arrs...)...)
		indexBufData = append(indexBufData, indices...)
	}

	vbo.SetData(vertexBufData, buffers.BufUsage_Static_Draw)
	ibo.SetData(indexBufData)

	mesh.Vao.AddVertexBuffer(vbo)
	mesh.Vao.SetIndexBuffer(ibo)

	// This is needed so that if you load meshes one after the other the
	// following mesh doesn't attach its vbo/ibo to this vao
	mesh.Vao.UnBind()

	return mesh, nil
}

func v3sToV2s(v3s []gglm.Vec3) []gglm.Vec2 {

	v2s := make([]gglm.Vec2, len(v3s))
	for i := 0; i < len(v3s); i++ {
		v2s[i] = gglm.Vec2{
			Data: [2]float32{v3s[i].X(), v3s[i].Y()},
		}
	}

	return v2s
}

type arrToInterleave struct {
	V2s []gglm.Vec2
	V3s []gglm.Vec3
	V4s []gglm.Vec4
}

func (a *arrToInterleave) get(i int) []float32 {

	assert.T(len(a.V2s) == 0 || len(a.V3s) == 0, "One array should be set in arrToInterleave, but multiple arrays are set")
	assert.T(len(a.V2s) == 0 || len(a.V4s) == 0, "One array should be set in arrToInterleave, but multiple arrays are set")
	assert.T(len(a.V3s) == 0 || len(a.V4s) == 0, "One array should be set in arrToInterleave, but multiple arrays are set")

	if len(a.V2s) > 0 {
		return a.V2s[i].Data[:]
	} else if len(a.V3s) > 0 {
		return a.V3s[i].Data[:]
	} else {
		return a.V4s[i].Data[:]
	}
}

func interleave(arrs ...arrToInterleave) []float32 {

	assert.T(len(arrs) > 0, "No input sent to interleave")
	assert.T(len(arrs[0].V2s) > 0 || len(arrs[0].V3s) > 0 || len(arrs[0].V4s) > 0, "Interleave arrays are empty")

	elementCount := 0
	if len(arrs[0].V2s) > 0 {
		elementCount = len(arrs[0].V2s)
	} else if len(arrs[0].V3s) > 0 {
		elementCount = len(arrs[0].V3s)
	} else {
		elementCount = len(arrs[0].V4s)
	}

	//Calculate final size of the float buffer
	totalSize := 0
	for i := 0; i < len(arrs); i++ {

		assert.T(len(arrs[i].V2s) == elementCount || len(arrs[i].V3s) == elementCount || len(arrs[i].V4s) == elementCount, "Mesh vertex data given to interleave is not the same length")

		if len(arrs[i].V2s) > 0 {
			totalSize += len(arrs[i].V2s) * 2
		} else if len(arrs[i].V3s) > 0 {
			totalSize += len(arrs[i].V3s) * 3
		} else {
			totalSize += len(arrs[i].V4s) * 4
		}
	}

	out := make([]float32, 0, totalSize)
	for i := 0; i < elementCount; i++ {
		for arrToUse := 0; arrToUse < len(arrs); arrToUse++ {
			out = append(out, arrs[arrToUse].get(i)...)
		}
	}

	return out
}

func flattenFaces(faces []asig.Face) []uint32 {

	assert.T(len(faces[0].Indices) == 3, "Face doesn't have 3 indices. Index count: %v\n", len(faces[0].Indices))

	uints := make([]uint32, len(faces)*3)
	for i := 0; i < len(faces); i++ {
		uints[i*3+0] = uint32(faces[i].Indices[0])
		uints[i*3+1] = uint32(faces[i].Indices[1])
		uints[i*3+2] = uint32(faces[i].Indices[2])
	}

	return uints
}
