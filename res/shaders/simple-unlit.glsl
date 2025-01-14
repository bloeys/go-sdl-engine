//shader:vertex
#version 410

layout(location=0) in vec3 vertPosIn;
layout(location=1) in vec3 vertNormalIn;
layout(location=2) in vec3 vertTangentIn;
layout(location=3) in vec2 vertUV0In;
layout(location=4) in vec3 vertColorIn;

out vec2 vertUV0;
out vec3 vertColor;
out vec3 fragPos;

uniform mat4 modelMat;
uniform mat4 projViewMat;

void main()
{
    vertUV0 = vertUV0In;
    vertColor = vertColorIn;

    vec4 modelVert = modelMat * vec4(vertPosIn, 1);
    fragPos = modelVert.xyz;
    gl_Position = projViewMat * modelVert;
}

//shader:fragment
#version 410

struct Material {
    sampler2D diffuse;
};

uniform Material material;

in vec3 vertColor;
in vec2 vertUV0;
in vec3 fragPos;

out vec4 fragColor;

void main()
{
    vec4 diffuseTexColor = texture(material.diffuse, vertUV0);
    fragColor = vec4(diffuseTexColor.rgb, 1);
}
