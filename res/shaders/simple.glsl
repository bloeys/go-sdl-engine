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
uniform mat4 viewMat;
uniform mat4 projMat;

void main()
{
    // @TODO: Calculate this on the CPU and send it as a uniform

    // This produces the normal matrix that multiplies with the model normal to produce the
    // world space normal. Based on 'One last thing' section from: https://learnopengl.com/Lighting/Basic-Lighting
    vertNormal = mat3(transpose(inverse(modelMat))) * vertNormalIn;
    
    vertUV0 = vertUV0In;
    vertColor = vertColorIn;
    fragPos = vec3(modelMat * vec4(vertPosIn, 1.0));

    gl_Position = projMat * viewMat * modelMat * vec4(vertPosIn, 1.0);
}

//shader:fragment
#version 410

uniform vec3 ambientColor = vec3(1, 1, 1);

uniform float specularShininess = 32;
uniform float specularStrength = 0.5;

uniform vec3 camPos;

uniform vec3 lightPos1;
uniform vec3 lightColor1;

uniform sampler2D diffTex;

in vec3 vertColor;
in vec3 vertNormal;
in vec2 vertUV0;
in vec3 fragPos;

out vec4 fragColor;

void main()
{
    vec3 lightDir = normalize(lightPos1 - fragPos);
    vec4 diffuseTexColor = texture(diffTex, vertUV0);

    // Ambient
    vec3 finalAmbient = ambientColor * diffuseTexColor.rgb;

    // Diffuse
    float diffuseAmount = max(0.0, dot(normalize(vertNormal), lightDir));
    vec3 finalDiffuse = diffuseAmount * lightColor1 * diffuseTexColor.rgb;

    // Specular
    vec3 viewDir = normalize(camPos - fragPos);
    vec3 reflectDir = reflect(-lightDir, vertNormal);
    float specularAmount = pow(max(dot(viewDir, reflectDir), 0.0), specularShininess);
    vec3 finalSpecular = specularAmount * specularStrength * lightColor1 * diffuseTexColor.rgb;

    fragColor = vec4(finalAmbient + finalDiffuse + finalSpecular, 1);
} 
