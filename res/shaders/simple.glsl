//shader:vertex
#version 410

#define NUM_SPOT_LIGHTS 4
#define NUM_POINT_LIGHTS 8

//
// Inputs
//
layout(location=0) in vec3 vertPosIn;
layout(location=1) in vec3 vertNormalIn;
layout(location=2) in vec3 vertTangentIn;
layout(location=3) in vec2 vertUV0In;
layout(location=4) in vec3 vertColorIn;

//
// Uniforms
//
uniform vec3 camPos;
uniform mat4 modelMat;
uniform mat3 normalMat;
uniform mat4 projViewMat;
uniform mat4 dirLightProjViewMat;
uniform mat4 spotLightProjViewMats[NUM_SPOT_LIGHTS];

struct DirLight {
    vec3 dir;
    vec3 diffuseColor;
    vec3 specularColor;
    sampler2D shadowMap;
};
uniform DirLight dirLight;

struct PointLight {
    vec3 pos;
    vec3 diffuseColor;
    vec3 specularColor;
    float falloff;
    float radius;
    float farPlane;
};
uniform PointLight pointLights[NUM_POINT_LIGHTS];

struct SpotLight {
    vec3 pos;
    vec3 dir;
    vec3 diffuseColor;
    vec3 specularColor;
    float innerCutoff;
    float outerCutoff;
};
uniform SpotLight spotLights[NUM_SPOT_LIGHTS];

//
// Outputs
//
out vec2 vertUV0;
out vec3 vertColor;

out vec3 fragPos;
out vec3 fragPosDirLight;
out vec4 fragPosSpotLight[NUM_SPOT_LIGHTS];

out vec3 tangentCamPos;
out vec3 tangentFragPos;
out vec3 tangentDirLightDir;
out vec3 tangentSpotLightPositions[NUM_SPOT_LIGHTS];
out vec3 tangentSpotLightDirections[NUM_SPOT_LIGHTS];
out vec3 tangentPointLightPositions[NUM_POINT_LIGHTS];

void main()
{
    vertUV0 = vertUV0In;
    vertColor = vertColorIn;
    vec4 modelVert = modelMat * vec4(vertPosIn, 1);

    // Tangent-BiTangent-Normal matrix for normal mapping
    vec3 T = normalize(vec3(modelMat * vec4(vertTangentIn,   0.0)));
    vec3 N = normalize(vec3(modelMat * vec4(vertNormalIn,    0.0)));

    // Ensure T is orthogonal with respect to N
    T = normalize(T - dot(T, N) * N);

    vec3 B = cross(N, T);
    mat3 tbnMtx = transpose(mat3(T, B, N));

    // Lighting related
    fragPos = modelVert.xyz;
    fragPosDirLight = vec3(dirLightProjViewMat * vec4(fragPos, 1));

    tangentCamPos = tbnMtx * camPos;
    tangentFragPos = tbnMtx * fragPos;
    tangentDirLightDir = tbnMtx * dirLight.dir;

    for (int i = 0; i < NUM_POINT_LIGHTS; i++)
        tangentPointLightPositions[i] = tbnMtx * pointLights[i].pos;

    for (int i = 0; i < NUM_SPOT_LIGHTS; i++)
    {
        fragPosSpotLight[i] = spotLightProjViewMats[i] * vec4(fragPos, 1);

        tangentSpotLightPositions[i] = tbnMtx * spotLights[i].pos;
        tangentSpotLightDirections[i] = tbnMtx * spotLights[i].dir;
    }

    gl_Position = projViewMat * modelVert;
}

//shader:fragment
#version 410

/*
    Note that while all lighting calculations are done in tangent space,
    shadow mapping is done in world space.

    The exception is the bias calculation. Since the bias relies on the normal
    and the normal is in tangent space, we use a tangent space fragment position
    with it, but the rest of shadow processing is in world space.
*/

#define NUM_SPOT_LIGHTS 4
#define NUM_POINT_LIGHTS 8

//
// Inputs
//
in vec3 fragPos;
in vec2 vertUV0;
in vec3 vertColor;
in vec3 fragPosDirLight;
in vec4 fragPosSpotLight[NUM_SPOT_LIGHTS];

in vec3 tangentCamPos;
in vec3 tangentFragPos;
in vec3 tangentDirLightDir;
in vec3 tangentSpotLightPositions[NUM_SPOT_LIGHTS];
in vec3 tangentSpotLightDirections[NUM_SPOT_LIGHTS];
in vec3 tangentPointLightPositions[NUM_POINT_LIGHTS];

//
// Uniforms
//
struct Material {
    sampler2D diffuse;
    sampler2D specular;
    sampler2D normal;
    sampler2D emission;
    float shininess;
};
uniform Material material;

struct DirLight {
    vec3 dir;
    vec3 diffuseColor;
    vec3 specularColor;
    sampler2D shadowMap;
};
uniform DirLight dirLight;

struct PointLight {
    vec3 pos;
    vec3 diffuseColor;
    vec3 specularColor;
    float falloff;
    float radius;
    float farPlane;
};
uniform PointLight pointLights[NUM_POINT_LIGHTS];
uniform samplerCubeArray pointLightCubeShadowMaps;

struct SpotLight {
    vec3 pos;
    vec3 dir;
    vec3 diffuseColor;
    vec3 specularColor;
    float innerCutoff;
    float outerCutoff;
};
uniform SpotLight spotLights[NUM_SPOT_LIGHTS];
uniform sampler2DArray spotLightShadowMaps;

uniform vec3 ambientColor = vec3(0.2, 0.2, 0.2);

//
// Outputs
//
out vec4 fragColor;

//
// Global variables used as cache for lighting calculations
//
vec3 tangentViewDir;
vec4 diffuseTexColor;
vec4 specularTexColor;
vec4 emissionTexColor;
vec3 normalizedVertNorm;

float CalcDirShadow(sampler2D shadowMap, vec3 tangentLightDir)
{
    // Move from [-1,1] to [0, 1]
    vec3 projCoords = fragPosDirLight * 0.5 + 0.5;

    // If sampling outside the depth texture then force 'no shadow'
    if(projCoords.z > 1)
        return 0;

    // currentDepth is the fragment depth from the light's perspective
    float currentDepth = projCoords.z;

    // Bias in the range [0.005, 0.05] depending on the angle, where a higher
    // angle gives a higher bias, as shadow acne gets worse with angle
    float bias = max(0.05 * (1 - dot(normalizedVertNorm, tangentLightDir)), 0.005);

    // 'Percentage Close Filtering'.
    // Basically get soft shadows by averaging this texel and surrounding ones
    float shadow = 0;
    vec2 texelSize = 1 / textureSize(shadowMap, 0);
    for(int x = -1; x <= 1; x++)
    {
        for(int y = -1; y <= 1; y++)
        {
            float pcfDepth = texture(shadowMap, projCoords.xy + vec2(x, y) * texelSize).r; 

            // If our depth is larger than the lights closest depth at the texel we checked (projCoords),
            // then there is something closer to the light than us, and so we are in shadow
            shadow += currentDepth - bias > pcfDepth ? 1 : 0;
        }
    }

    shadow /= 9;

    return shadow;
}

vec3 CalcDirLight()
{
    vec3 lightDir = normalize(-tangentDirLightDir);

    // Diffuse
    float diffuseAmount = max(0.0, dot(normalizedVertNorm, lightDir));
    vec3 finalDiffuse = diffuseAmount * dirLight.diffuseColor * diffuseTexColor.rgb;

    // Specular
    vec3 halfwayDir = normalize(lightDir + tangentViewDir);
    float specularAmount = pow(max(dot(normalizedVertNorm, halfwayDir), 0.0), material.shininess);
    vec3 finalSpecular = specularAmount * dirLight.specularColor * specularTexColor.rgb;

    // Shadow
    float shadow = CalcDirShadow(dirLight.shadowMap, lightDir);

    return (finalDiffuse + finalSpecular) * (1 - shadow);
}

float CalcPointShadow(int lightIndex, vec3 worldLightPos, vec3 tangentLightDir, float farPlane) {

    vec3 lightToFrag = fragPos - worldLightPos;

    float closestDepth = texture(pointLightCubeShadowMaps, vec4(lightToFrag, lightIndex)).r;

    // We stored depth in the cubemap in the range [0, 1], so now we move back to [0, farPlane]
    closestDepth *= farPlane;

    // Get depth of current fragment
    float currentDepth = length(lightToFrag);

    float bias = max(0.05 * (1 - dot(normalizedVertNorm, tangentLightDir)), 0.005);

    float shadow = currentDepth -  bias > closestDepth ? 1.0 : 0.0;

    return shadow;
}

//
// The following point light attenuation formulas
// are from https://lisyarus.github.io/blog/posts/point-light-attenuation.html
//
// I found them more intuitive than the standard implementation and it also ensures
// we have zero light at the selected distance.
//
float sqr(float x)
{
    return x * x;
}

// This version doesn't have a harsh cutoff at radius
float AttenuateNoCusp(float dist, float radius, float falloff)
{
    // Since we only use this as attenuation and max intensity defines
    // the max output value, anything more than 1 would increase
    // the output of the light, which I don't think makes sense for
    // our attenuation purposes.
    //
    // Seems to me this can be done simply by increasing color values above 255. 
    //
    // Forcing to 1 for now.
    #define MAX_INTENSITY 1

    float s = dist / radius;

    if (s >= 1.0)
        return 0.0;

    float s2 = sqr(s);

    return MAX_INTENSITY * sqr(1 - s2) / (1 + falloff * s2);
}

// This version has a harsh/immediate cutoff at radius
float AttenuateCusp(float dist, float radius, float falloff)
{
    #define MAX_INTENSITY 1

    float s = dist / radius;

    if (s >= 1.0)
        return 0.0;

    float s2 = sqr(s);

    return MAX_INTENSITY * sqr(1 - s2) / (1 + falloff * s);
}

vec3 CalcPointLight(PointLight pointLight, int lightIndex)
{
    // Ignore inactive lights
    if (pointLight.radius == 0){
        return vec3(0);
    }

    vec3 tangentLightPos = tangentPointLightPositions[lightIndex];
    vec3 tangentLightDir = normalize(tangentLightPos - tangentFragPos);

    // Diffuse
    float diffuseAmount = max(0.0, dot(normalizedVertNorm, tangentLightDir));
    vec3 finalDiffuse = diffuseAmount * pointLight.diffuseColor * diffuseTexColor.rgb;

    // Specular
    vec3 halfwayDir = normalize(tangentLightDir + tangentViewDir);
    float specularAmount = pow(max(dot(normalizedVertNorm, halfwayDir), 0.0), material.shininess);
    vec3 finalSpecular = specularAmount * pointLight.specularColor * specularTexColor.rgb;

    // Attenuation
    float distToLight = length(tangentLightPos - tangentFragPos);
    float attenuation = AttenuateNoCusp(distToLight, pointLight.radius, pointLight.falloff);

    // Shadow
    float shadow = CalcPointShadow(lightIndex, pointLight.pos, tangentLightDir, pointLight.farPlane);

    return (finalDiffuse + finalSpecular) * attenuation * (1 - shadow);
}

float CalcSpotShadow(vec3 tangentLightDir, int lightIndex)
{
    // Move from clip space to NDC
    vec3 projCoords = fragPosSpotLight[lightIndex].xyz / fragPosSpotLight[lightIndex].w;

    // Move from [-1,1] to [0, 1]
    projCoords = projCoords * 0.5 + 0.5;

    // If sampling outside the depth texture then force 'no shadow'
    if(projCoords.z > 1)
        return 0;

    // currentDepth is the fragment depth from the light's perspective
    float currentDepth = projCoords.z;

    // Bias in the range [0.005, 0.05] depending on the angle, where a higher
    // angle gives a higher bias, as shadow acne gets worse with angle
    float bias = max(0.05 * (1 - dot(normalizedVertNorm, tangentLightDir)), 0.005);

    // 'Percentage Close Filtering'.
    // Basically get soft shadows by averaging this texel and surrounding ones
    float shadow = 0;
    vec2 texelSize = 1 / textureSize(spotLightShadowMaps, 0).xy;
    for(int x = -1; x <= 1; x++)
    {
        for(int y = -1; y <= 1; y++)
        {
            float pcfDepth = texture(spotLightShadowMaps, vec3(projCoords.xy + vec2(x, y) * texelSize, lightIndex)).r; 

            // If our depth is larger than the lights closest depth at the texel we checked (projCoords),
            // then there is something closer to the light than us, and so we are in shadow
            shadow += currentDepth - bias > pcfDepth ? 1 : 0;
        }
    }

    shadow /= 9;

    return shadow;
}

vec3 CalcSpotLight(SpotLight light, int lightIndex)
{
    if (light.innerCutoff == 0)
        return vec3(0);

    vec3 tangentLightDir = tangentSpotLightDirections[lightIndex];
    vec3 fragToLightDir = normalize(tangentSpotLightPositions[lightIndex] - tangentFragPos);

    // Spot light cone with full intensity within inner cutoff,
    // and falloff between inner-outer cutoffs, and zero
    // light after outer cutoff
    float theta = dot(fragToLightDir, normalize(-tangentLightDir));
    float epsilon = (light.innerCutoff - light.outerCutoff);
    float intensity = clamp((theta - light.outerCutoff) / epsilon, float(0), float(1));

    if (intensity == 0)
        return vec3(0);

    // Diffuse
    float diffuseAmount = max(0.0, dot(normalizedVertNorm, fragToLightDir));
    vec3 finalDiffuse = diffuseAmount * light.diffuseColor * diffuseTexColor.rgb;

    // Specular
    vec3 halfwayDir = normalize(fragToLightDir + tangentViewDir);
    float specularAmount = pow(max(dot(normalizedVertNorm, halfwayDir), 0.0), material.shininess);
    vec3 finalSpecular = specularAmount * light.specularColor * specularTexColor.rgb;

    // Shadow
    float shadow = CalcSpotShadow(fragToLightDir, lightIndex);

    return (finalDiffuse + finalSpecular) * intensity * (1 - shadow);
}

#define DRAW_NORMALS false

void main()
{
    // Shared values
    tangentViewDir = normalize(tangentCamPos - tangentFragPos);
    diffuseTexColor = texture(material.diffuse, vertUV0);
    specularTexColor = texture(material.specular, vertUV0);
    emissionTexColor = texture(material.emission, vertUV0);

    // Read normal data encoded [0,1]
    normalizedVertNorm = texture(material.normal, vertUV0).rgb;

    // Remap normal to [-1,1]
    normalizedVertNorm = normalize(normalizedVertNorm * 2.0 - 1.0);

    // Light contributions
    vec3 finalColor = CalcDirLight();

    for (int i = 0; i < NUM_POINT_LIGHTS; i++)
    {
        finalColor += CalcPointLight(pointLights[i], i);
    }

    for (int i = 0; i < NUM_SPOT_LIGHTS; i++)
    {
        finalColor += CalcSpotLight(spotLights[i], i);
    }

    vec3 finalEmission = emissionTexColor.rgb;
    vec3 finalAmbient = ambientColor * diffuseTexColor.rgb;

    fragColor = vec4(finalColor + finalAmbient + finalEmission, 1);

    if (DRAW_NORMALS)
    {
        fragColor = vec4(texture(material.normal, vertUV0).rgb, 1);
    }
}
