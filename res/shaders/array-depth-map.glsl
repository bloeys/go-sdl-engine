//shader:vertex
#version 410

layout(location=0) in vec3 vertPosIn;

uniform mat4 modelMat;

void main()
{
    gl_Position = modelMat * vec4(vertPosIn, 1);
}

//shader:geometry
#version 410

layout (triangles) in;

#define NUM_PROJ_VIEW_MATS 4

// 3 * NUM_PROJ_VIEW_MATS
layout (triangle_strip, max_vertices=12) out;

// This is the same number as max spot lights or whatever else is being rendered
uniform mat4 projViewMats[NUM_PROJ_VIEW_MATS];

out vec4 FragPos;

void main()
{
    for(int projViewMatIndex = 0; projViewMatIndex < NUM_PROJ_VIEW_MATS; projViewMatIndex++){

        gl_Layer = projViewMatIndex;
        mat4 projViewMat = projViewMats[projViewMatIndex];

        for(int i = 0; i < 3; i++)
        {
            FragPos = gl_in[i].gl_Position;
            gl_Position = projViewMat * FragPos;
            EmitVertex();
        }
        EndPrimitive();
    }
}

//shader:fragment
#version 410

in vec4 FragPos;

void main()
{
    // This implicitly writes to the depth buffer with no color operations
    // Equivalent: gl_FragDepth = gl_FragCoord.z;
}
