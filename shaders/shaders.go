package shaders

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bloeys/nmage/logging"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type Shader struct {
	Id   uint32
	Type ShaderType
}

func (s *Shader) Delete() {
	gl.DeleteShader(s.Id)
	s.Id = 0
}

func NewShaderProgram() (ShaderProgram, error) {

	id := gl.CreateProgram()
	if id == 0 {
		return ShaderProgram{}, errors.New("failed to create shader program")
	}

	return ShaderProgram{Id: id}, nil
}

func LoadAndCompileCombinedShader(shaderPath string) (ShaderProgram, error) {

	combinedSource, err := os.ReadFile(shaderPath)
	if err != nil {
		logging.ErrLog.Println("Failed to read shader. Err: ", err)
		return ShaderProgram{}, err
	}

	return LoadAndCompileCombinedShaderSrc(combinedSource)

}
func LoadAndCompileCombinedShaderSrc(shaderSrc []byte) (ShaderProgram, error) {

	shaderSources := bytes.Split(shaderSrc, []byte("//shader:"))
	if len(shaderSources) < 2 {
		return ShaderProgram{}, errors.New("failed to read combined shader. The minimum shader types to have are '//shader:vertex' and '//shader:fragment'")
	}

	shdrProg, err := NewShaderProgram()
	if err != nil {
		return ShaderProgram{}, errors.New("failed to create new shader program. Err: " + err.Error())
	}

	loadedShdrCount := 0
	for i := 0; i < len(shaderSources); i++ {

		src := shaderSources[i]

		//This can happen when the shader type is at the start of the file
		if len(bytes.TrimSpace(src)) == 0 {
			continue
		}

		var shdrType ShaderType
		if bytes.HasPrefix(src, []byte("vertex")) {
			src = src[6:]
			shdrType = ShaderType_Vertex
		} else if bytes.HasPrefix(src, []byte("fragment")) {
			src = src[8:]
			shdrType = ShaderType_Fragment
		} else if bytes.HasPrefix(src, []byte("geometry")) {
			src = src[8:]
			shdrType = ShaderType_Geometry
		} else {
			return ShaderProgram{}, errors.New("unknown shader type. Must be '//shader:vertex' or '//shader:fragment' or '//shader:geometry'")
		}

		shdr, err := CompileShaderOfType(src, shdrType)
		if err != nil {
			return ShaderProgram{}, err
		}

		loadedShdrCount++
		shdrProg.AttachShader(shdr)
	}

	if loadedShdrCount == 0 {
		return ShaderProgram{}, errors.New("no valid shaders found. Please put '//shader:vertex' or '//shader:fragment' or '//shader:geometry' before your shaders")
	}

	if shdrProg.VertShaderId == 0 {
		return ShaderProgram{}, errors.New("no valid vertex shader found. Please put '//shader:vertex' before your vertex shader")
	}

	if shdrProg.FragShaderId == 0 {
		return ShaderProgram{}, errors.New("no valid fragment shader found. Please put '//shader:fragment' before your vertex shader")
	}

	shdrProg.Link()
	return shdrProg, nil
}

func CompileShaderOfType(shaderSource []byte, shaderType ShaderType) (Shader, error) {

	shaderId := gl.CreateShader(shaderType.ToGl())
	if shaderId == 0 {
		return Shader{}, fmt.Errorf("failed to create OpenGl shader. OpenGl Error=%d", gl.GetError())
	}

	//Load shader source and compile
	shaderCStr, shaderFree := gl.Strs(string(shaderSource) + "\x00")
	defer shaderFree()
	gl.ShaderSource(shaderId, 1, shaderCStr, nil)

	gl.CompileShader(shaderId)
	if err := getShaderCompileErrors(shaderId); err != nil {
		gl.DeleteShader(shaderId)
		return Shader{}, err
	}

	return Shader{Id: shaderId, Type: shaderType}, nil
}

func getShaderCompileErrors(shaderId uint32) error {

	var compiledSuccessfully int32
	gl.GetShaderiv(shaderId, gl.COMPILE_STATUS, &compiledSuccessfully)
	if compiledSuccessfully == gl.TRUE {
		return nil
	}

	var logLength int32
	gl.GetShaderiv(shaderId, gl.INFO_LOG_LENGTH, &logLength)

	log := gl.Str(strings.Repeat("\x00", int(logLength)))
	gl.GetShaderInfoLog(shaderId, logLength, nil, log)

	errMsg := gl.GoStr(log)
	logging.ErrLog.Println("Compilation of shader with id ", shaderId, " failed. Err: ", errMsg)
	return errors.New(errMsg)
}
