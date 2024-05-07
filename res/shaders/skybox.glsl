//shader:vertex
#version 410

layout(location=0) in vec3 vertPosIn;
layout(location=1) in vec3 vertNormalIn;
layout(location=2) in vec3 vertTangentIn;
layout(location=3) in vec2 vertUV0In;
layout(location=4) in vec3 vertColorIn;

out vec3 vertUV0;

uniform mat4 projViewMat;

void main()
{
    vertUV0 = vec3(vertPosIn.x, vertPosIn.y, -vertPosIn.z);
    vec4 pos = projViewMat * vec4(vertPosIn, 1.0);
    gl_Position = pos.xyww;
}

//shader:fragment
#version 410

in vec3 vertUV0;

out vec4 fragColor;

uniform samplerCube skybox;

void main()
{
    fragColor = texture(skybox, vertUV0);
} 
