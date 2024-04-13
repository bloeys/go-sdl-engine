//shader:vertex
#version 410

layout(location=0) in vec3 vertPosIn;
layout(location=1) in vec3 vertNormalIn;
layout(location=2) in vec2 vertUV0In;
layout(location=3) in vec3 vertColorIn;

out vec3 vertNormal;
out vec2 vertUV0;
out vec3 vertColor;
out vec3 fragPos;

//MVP = Model View Projection
uniform mat4 modelMat;
uniform mat4 projViewMat;

void main()
{
    vertNormal = mat3(transpose(inverse(modelMat))) * vertNormalIn;
    vertUV0 = vertUV0In;
    vertColor = vertColorIn;

    vec4 modelVert = modelMat * vec4(vertPosIn, 1);
    fragPos = modelVert.xyz;

    gl_Position = projViewMat * modelVert;
}

//shader:fragment
#version 410

in vec3 vertColor;
in vec3 vertNormal;
in vec2 vertUV0;
in vec3 fragPos;

out vec4 fragColor;

uniform float near = 0.1;
uniform float far  = 200.0;
  
float LinearizeDepth(float depth) 
{
    float z = depth * 2.0 - 1.0; // back to NDC 
    return (2.0 * near * far) / (far + near - z * (far - near));	
}

void main()
{
    float depth = LinearizeDepth(gl_FragCoord.z) / far;
    fragColor = vec4(vec3(depth), 1.0);
} 
