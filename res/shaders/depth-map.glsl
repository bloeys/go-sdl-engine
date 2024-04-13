//shader:vertex
#version 410

layout(location=0) in vec3 vertPosIn;

uniform mat4 modelMat;
uniform mat4 projViewMat;

void main()
{
    gl_Position = projViewMat * modelMat * vec4(vertPosIn, 1);
}

//shader:fragment
#version 410

void main()
{
    // This implicitly writes to the depth buffer with no color operations
    // Equivalent: gl_FragDepth = gl_FragCoord.z;
}
