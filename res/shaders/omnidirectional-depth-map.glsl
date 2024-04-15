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

// Cubemap means 6 faces, and the
// input 3 triangle vertices are drawn once per face, so 6*3=18
layout (triangle_strip, max_vertices=18) out;

uniform int cubemapIndex;
uniform mat4 cubemapProjViewMats[6];

out vec4 FragPos;

void main()
{
    for(int face = 0; face < 6; ++face)
    {
        // Built in variable that specifies which cubemap face we are rendering to
        // and only works when a cubemap is attached to the active fbo.
        //
        // We use an additional index here because our fbo has a cubemap array
        gl_Layer = (cubemapIndex * 6) + face;

        // Transform each triangle vertex
        for(int i = 0; i < 3; ++i)
        {
            FragPos = gl_in[i].gl_Position;
            gl_Position = cubemapProjViewMats[face] * FragPos;
            EmitVertex();
        }
        EndPrimitive();
    }
}

//shader:fragment
#version 410

in vec4 FragPos;

uniform vec3 lightPos;
uniform float farPlane;

void main()
{
    // Get distance between fragment and light source
    float lightDistance = length(FragPos.xyz - lightPos);

    // Map to [0, 1] by dividing by far plane and use it as our depth
    lightDistance = lightDistance / farPlane;

    gl_FragDepth = lightDistance;
}
