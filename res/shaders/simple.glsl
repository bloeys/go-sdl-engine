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
out vec4 fragPosDirLight;

uniform mat4 modelMat;
uniform mat4 projViewMat;
uniform mat4 dirLightProjViewMat;

void main()
{
    // @TODO: Calculate this on the CPU and send it as a uniform
    //
    // This produces the normal matrix that multiplies with the model normal to produce the
    // world space normal. Based on 'One last thing' section from: https://learnopengl.com/Lighting/Basic-Lighting
    vertNormal = mat3(transpose(inverse(modelMat))) * vertNormalIn;
    
    vertUV0 = vertUV0In;
    vertColor = vertColorIn;

    vec4 modelVert = modelMat * vec4(vertPosIn, 1);
    fragPos = modelVert.xyz;
    fragPosDirLight = dirLightProjViewMat * vec4(fragPos, 1);

    gl_Position = projViewMat * modelVert;
}

//shader:fragment
#version 410

struct Material {
    sampler2D diffuse;
    sampler2D specular;
    // sampler2D normal;
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
    float constant;
    float linear;
    float quadratic;
};

#define NUM_POINT_LIGHTS 16
uniform PointLight pointLights[NUM_POINT_LIGHTS];

struct SpotLight {
    vec3 pos;
    vec3 dir;
    vec3 diffuseColor;
    vec3 specularColor;
    float innerCutoff;
    float outerCutoff;
};

#define NUM_SPOT_LIGHTS 4
uniform SpotLight spotLights[NUM_SPOT_LIGHTS];

uniform vec3 camPos;
uniform vec3 ambientColor = vec3(0.2, 0.2, 0.2);

in vec3 vertColor;
in vec3 vertNormal;
in vec2 vertUV0;
in vec3 fragPos;
in vec4 fragPosDirLight;

out vec4 fragColor;

// Global variables used as cache for lighting calculations
vec4 diffuseTexColor;
vec4 specularTexColor;
vec4 emissionTexColor;
vec3 normalizedVertNorm;
vec3 viewDir;

float CalcShadow(sampler2D shadowMap, vec3 lightDir)
{
    // Move from clip space to NDC
    vec3 projCoords = fragPosDirLight.xyz / fragPosDirLight.w;

    // Move from [-1,1] to [0, 1]
    projCoords = projCoords * 0.5 + 0.5;

    // If sampling outside the depth texture then force 'no shadow'
    if(projCoords.z > 1)
        return 0;

    // currentDepth is the fragment depth from the light's perspective
    float currentDepth = projCoords.z;

    // Bias in the range [0.005, 0.05] depending on the angle, where a higher
    // angle gives a higher bias, as shadow acne gets worse with angle
    float bias = max(0.05 * (1 - dot(normalizedVertNorm, lightDir)), 0.005);

    // 'Percentage Close Filtering'. B
    // Basically get soft shadows by averaging this texel and surrounding ones
    float shadow = 0;
    vec2 texelSize = 1 / textureSize(shadowMap, 0);
    for(int x = -1; x <= 1; ++x)
    {
        for(int y = -1; y <= 1; ++y)
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
    vec3 lightDir = normalize(-dirLight.dir);

    // Diffuse
    float diffuseAmount = max(0.0, dot(normalizedVertNorm, lightDir));
    vec3 finalDiffuse = diffuseAmount * dirLight.diffuseColor * diffuseTexColor.rgb;

    // Specular
    vec3 halfwayDir = normalize(lightDir + viewDir);
    float specularAmount = pow(max(dot(normalizedVertNorm, halfwayDir), 0.0), material.shininess);
    vec3 finalSpecular = specularAmount * dirLight.specularColor * specularTexColor.rgb;

    // Shadow
    float shadow = CalcShadow(dirLight.shadowMap, lightDir);

    return (finalDiffuse + finalSpecular) * (1 - shadow);
}

vec3 CalcPointLight(PointLight pointLight)
{
    // Ignore unset lights
    if (pointLight.constant == 0){
        return vec3(0);
    }

    vec3 lightDir = normalize(pointLight.pos - fragPos);

    // Diffuse
    float diffuseAmount = max(0.0, dot(normalizedVertNorm, lightDir));
    vec3 finalDiffuse = diffuseAmount * pointLight.diffuseColor * diffuseTexColor.rgb;

    // Specular
    vec3 halfwayDir = normalize(lightDir + viewDir);
    float specularAmount = pow(max(dot(normalizedVertNorm, halfwayDir), 0.0), material.shininess);
    vec3 finalSpecular = specularAmount * pointLight.specularColor * specularTexColor.rgb;

    // attenuation
    float distToLight = length(pointLight.pos - fragPos);
    float attenuation = 1 / (pointLight.constant + pointLight.linear * distToLight + pointLight.quadratic * (distToLight * distToLight));  

    return (finalDiffuse + finalSpecular) * attenuation;
}

vec3 CalcSpotLight(SpotLight light)
{
    if (light.innerCutoff == 0)
        return vec3(0);

    vec3 fragToLightDir = normalize(light.pos - fragPos);

    // Spot light cone with full intensity within inner cutoff,
    // and falloff between inner-outer cutoffs, and zero
    // light after outer cutoff
    float theta = dot(fragToLightDir, normalize(-light.dir));
    float epsilon = (light.innerCutoff - light.outerCutoff);
    float intensity = clamp((theta - light.outerCutoff) / epsilon, float(0), float(1));

    if (intensity == 0)
        return vec3(0);

    // Diffuse
    float diffuseAmount = max(0.0, dot(normalizedVertNorm, fragToLightDir));
    vec3 finalDiffuse = diffuseAmount * light.diffuseColor * diffuseTexColor.rgb;

    // Specular
    vec3 halfwayDir = normalize(fragToLightDir + viewDir);
    float specularAmount = pow(max(dot(normalizedVertNorm, halfwayDir), 0.0), material.shininess);
    vec3 finalSpecular = specularAmount * light.specularColor * specularTexColor.rgb;

    return (finalDiffuse + finalSpecular) * intensity;
}

void main()
{
    // Shared values
    diffuseTexColor = texture(material.diffuse, vertUV0);
    specularTexColor = texture(material.specular, vertUV0);
    emissionTexColor = texture(material.emission, vertUV0);

    normalizedVertNorm = normalize(vertNormal);
    viewDir = normalize(camPos - fragPos);

    // Light contributions
    vec3 finalColor = CalcDirLight();

    for (int i = 0; i < NUM_POINT_LIGHTS; i++)
    {
        finalColor += CalcPointLight(pointLights[i]);
    }

    for (int i = 0; i < NUM_SPOT_LIGHTS; i++)
    {
        finalColor += CalcSpotLight(spotLights[i]);
    }

    vec3 finalEmission = emissionTexColor.rgb;
    vec3 finalAmbient = ambientColor * diffuseTexColor.rgb;

    fragColor = vec4(finalColor + finalAmbient + finalEmission, 1);
}
