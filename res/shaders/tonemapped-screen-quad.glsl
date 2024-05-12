//shader:vertex
#version 410

out vec2 vertUV0;

// Hardcoded vertex positions for a fullscreen quad.
// Format: vec4(pos.x, pos.y, uv0.x, uv0.y)
vec4 quadData[6] = vec4[](
    vec4(-1.0,  1.0, 0.0, 1.0),
    vec4(-1.0, -1.0, 0.0, 0.0),
    vec4(1.0, -1.0, 1.0, 0.0),
    vec4(-1.0,  1.0, 0.0, 1.0),
    vec4(1.0, -1.0, 1.0, 0.0),
    vec4(1.0,  1.0, 1.0, 1.0)
);

void main()
{
    vec4 vertData = quadData[gl_VertexID];

    vertUV0 = vertData.zw;
    gl_Position = vec4(vertData.xy, 0.0, 1.0);
}

//shader:fragment
#version 410

struct Material {
    sampler2D diffuse;
};

uniform float exposure = 1;
uniform Material material;

in vec2 vertUV0;

out vec4 fragColor;

void main()
{
    vec4 diffuseTexColor = texture(material.diffuse, vertUV0);

    // Reinhard tone mapping
    // vec3 mappedColor = diffuseTexColor.rgb / (diffuseTexColor.rgb + vec3(1.0));

    // Exposure tone mapping
    vec3 mappedColor = vec3(1.0) - exp(-diffuseTexColor.rgb * exposure);

    fragColor = vec4(mappedColor, 1);
}
