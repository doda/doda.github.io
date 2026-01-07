(function(){const v=window.matchMedia("(prefers-reduced-motion: reduce)").matches;if(v)return;const c=66,h=100;let a=!1,o=!0,s=!0,n=new Set;const b=`
    attribute vec2 a_position;
    varying vec2 v_position;
    void main() {
      gl_Position = vec4(a_position, 0, 1);
      v_position = a_position;
    }
  `,u=`
    precision mediump float;
    uniform vec2 u_resolution;
    uniform float u_time;
    uniform float u_isDark;
    uniform float u_fadeTop;
    uniform float u_dpr;
    varying vec2 v_position;

    float hash(vec2 p) {
      return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
    }

    vec2 hash2(vec2 p) {
      p = vec2(dot(p, vec2(127.1, 311.7)), dot(p, vec2(269.5, 183.3)));
      return fract(sin(p) * 43758.5453);
    }

    float vnoise(vec2 p) {
      vec2 i = floor(p);
      vec2 f = fract(p);
      f = f * f * (3.0 - 2.0 * f);
      return mix(
        mix(hash(i), hash(i + vec2(1.0, 0.0)), f.x),
        mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), f.x),
        f.y
      ) * 2.0 - 1.0;
    }

    // Returns vec2(field, colorVariation) where colorVariation is 0-1 based on dominant blob
    vec2 metaball(vec2 p, float time) {
      vec2 i = floor(p);
      vec2 f = fract(p);
      float sum = 0.0;
      float colorSum = 0.0;

      for (int y = -1; y <= 1; y++) {
        for (int x = -1; x <= 1; x++) {
          vec2 neighbor = vec2(float(x), float(y));
          vec2 cellId = i + neighbor;
          vec2 point = hash2(cellId);
          point = 0.5 + 0.4 * sin(time * 0.3 + 6.28 * point);
          vec2 diff = neighbor + point - f;
          float r2 = dot(diff, diff);
          float influence = max(0.0, 1.0 - r2);
          float contrib = influence * influence * influence;
          sum += contrib;
          // Each blob gets a unique color variation based on its cell
          float blobColor = hash(cellId + 0.5);
          colorSum += contrib * blobColor;
        }
      }
      float avgColor = sum > 0.0 ? colorSum / sum : 0.0;
      return vec2(sum, avgColor);
    }

    vec2 warpCoords(vec2 p, float time) {
      float warp1 = vnoise(p * 0.5 + time * 0.05);
      float warp2 = vnoise(p * 0.3 - time * 0.03 + 100.0);
      return p + vec2(warp1, warp2) * 0.4;
    }

    void main() {
      float scale = 0.0133;
      vec2 p = gl_FragCoord.xy * scale;

      // Monochrome palette - narrow range
      vec3 lightPageBg = vec3(1.0, 1.0, 1.0);
      vec3 lightLayer1 = vec3(0.93, 0.93, 0.93);  // Lightest (back)
      vec3 lightLayer2 = vec3(0.90, 0.90, 0.90);  // Mid
      vec3 lightLayer3 = vec3(0.86, 0.86, 0.86);  // Darkest (front)

      vec3 darkPageBg = vec3(0.10, 0.10, 0.10);
      vec3 darkLayer1 = vec3(0.16, 0.16, 0.16);   // Lightest
      vec3 darkLayer2 = vec3(0.20, 0.20, 0.20);   // Mid
      vec3 darkLayer3 = vec3(0.25, 0.25, 0.25);   // Darkest

      vec3 pageBg = mix(lightPageBg, darkPageBg, u_isDark);
      vec3 layer1 = mix(lightLayer1, darkLayer1, u_isDark);
      vec3 layer2 = mix(lightLayer2, darkLayer2, u_isDark);
      vec3 layer3 = mix(lightLayer3, darkLayer3, u_isDark);

      float baseHeight = 75.0 * u_dpr;
      float xCoord = gl_FragCoord.x / u_dpr;
      float boundary = baseHeight;
      // Undulating waves
      boundary += sin(xCoord * 0.008 + u_time * 0.15) * 18.0 * u_dpr;         // Primary wave
      boundary += sin(xCoord * 0.015 - u_time * 0.1) * 12.0 * u_dpr;          // Secondary wave (opposite dir)
      boundary += sin(xCoord * 0.025 + u_time * 0.2) * 6.0 * u_dpr;           // Tertiary ripple
      // Random variation on top
      boundary += vnoise(vec2(xCoord * 0.01, u_time * 0.03)) * 8.0 * u_dpr;
      // Clamp to canvas bounds
      boundary = clamp(boundary, 20.0 * u_dpr, u_resolution.y - 10.0);

      float pixelY = mix(gl_FragCoord.y, u_resolution.y - gl_FragCoord.y, u_fadeTop);
      float distToBoundary = pixelY - boundary;

      float wallRange = 40.0 * u_dpr;
      float wallInfluence = smoothstep(wallRange, 0.0, distToBoundary) * 0.25;

      float taperRange = 30.0 * u_dpr;
      float taper = smoothstep(-taperRange, 0.0, distToBoundary);
      float aaWidth = 0.005 / max(u_dpr, 1.0);
      float varStrength = 0.06;

      vec3 color = pageBg;

      // 5 layers from back to front
      for (int i = 0; i < 5; i++) {
        float fi = float(i);
        float t = fi / 4.0;  // 0 to 1

        // Scale: large (0.15) to small (1.2)
        float scale = 0.15 + t * 1.05;
        // Speed: slow to fast
        float speed = 0.02 + t * 0.08;
        // Threshold: loose (0.75) to tight (0.96)
        float threshold = 0.75 + t * 0.21;
        // Offset for variety
        vec2 offset = vec2(fi * 17.3, fi * 23.7);

        vec2 pLayer = p * scale + vec2(u_time * speed * 0.05, u_time * speed * 0.03) + offset;
        vec2 meta = metaball(warpCoords(pLayer, u_time * speed), u_time * speed);
        float field = (meta.x + wallInfluence) * taper;
        float blend = smoothstep(threshold - aaWidth, threshold + aaWidth, field);

        // Interpolate color through the pink palette
        vec3 layerColor = mix(layer1, layer3, t);
        float variation = 1.0 + (meta.y - 0.5) * varStrength;

        color = mix(color, layerColor * variation, blend);
      }

      gl_FragColor = vec4(color, 1.0);
    }
  `;function l(e,t,n){const s=e.createShader(t);return e.shaderSource(s,n),e.compileShader(s),s}function p(){return document.body.classList.contains("dark")?1:0}function f(e,t){const o=document.getElementById(e);if(!o)return null;const n=o.getContext("webgl2",{antialias:!0,powerPreference:"low-power"})||o.getContext("webgl",{antialias:!0,powerPreference:"low-power"});if(!n)return null;const a=l(n,n.VERTEX_SHADER,b),r=l(n,n.FRAGMENT_SHADER,u),s=n.createProgram();n.attachShader(s,a),n.attachShader(s,r),n.linkProgram(s),n.useProgram(s);const c=n.createBuffer();n.bindBuffer(n.ARRAY_BUFFER,c),n.bufferData(n.ARRAY_BUFFER,new Float32Array([-1,-1,1,-1,-1,1,1,1]),n.STATIC_DRAW);const i=n.getAttribLocation(s,"a_position");return n.enableVertexAttribArray(i),n.vertexAttribPointer(i,2,n.FLOAT,!1,0,0),{canvas:o,gl:n,resolutionLoc:n.getUniformLocation(s,"u_resolution"),timeLoc:n.getUniformLocation(s,"u_time"),isDarkLoc:n.getUniformLocation(s,"u_isDark"),fadeTopLoc:n.getUniformLocation(s,"u_fadeTop"),dprLoc:n.getUniformLocation(s,"u_dpr"),hoverLoc:n.getUniformLocation(s,"u_hover"),hoverTarget:0,hoverValue:0,needsResize:!0,fadeTop:t?1:0}}let e=[],m=performance.now(),i=0,d=window.devicePixelRatio;function g(){const e=window.devicePixelRatio;return e<=1?1.5:e}function t(a){if(!o||n.size===0){requestAnimationFrame(t);return}const f=s?c:h;if(a-i<f){requestAnimationFrame(t);return}i=a;const v=(performance.now()-m)/1e3,b=p(),l=window.devicePixelRatio;l!==d&&(d=l,e.forEach(e=>e.needsResize=!0));const r=g(),u=c/1e3;for(const t of e){if(!n.has(t.canvas))continue;const{canvas:o,gl:s,resolutionLoc:a,timeLoc:c,isDarkLoc:l,fadeTopLoc:d,dprLoc:h,hoverLoc:m}=t;t.needsResize&&(o.width=o.offsetWidth*r,o.height=o.offsetHeight*r,s.viewport(0,0,o.width,o.height),s.uniform2f(a,o.width,o.height),s.uniform1f(d,t.fadeTop),t.needsResize=!1);const i=1/2.65;t.hoverTarget>t.hoverValue?t.hoverValue=Math.min(t.hoverTarget,t.hoverValue+u*i):t.hoverTarget<t.hoverValue&&(t.hoverValue=Math.max(t.hoverTarget,t.hoverValue-u*i)),s.uniform1f(c,v),s.uniform1f(l,b),s.uniform1f(h,r),s.uniform1f(m,t.hoverValue),s.drawArrays(s.TRIANGLE_STRIP,0,4)}requestAnimationFrame(t)}function r(){if(a)return;if(a=!0,e=[f("footer-canvas",!0)].filter(Boolean),e.length===0)return;e.forEach(e=>{const t=e.canvas.parentElement;t&&(t.addEventListener("mouseenter",()=>e.hoverTarget=1),t.addEventListener("mouseleave",()=>e.hoverTarget=0))}),document.addEventListener("visibilitychange",()=>{o=!document.hidden}),window.addEventListener("focus",()=>{s=!0}),window.addEventListener("blur",()=>{s=!1});const i=new IntersectionObserver(e=>{e.forEach(e=>{e.isIntersecting?n.add(e.target):n.delete(e.target)})},{threshold:0});e.forEach(e=>i.observe(e.canvas)),window.addEventListener("resize",()=>e.forEach(e=>e.needsResize=!0)),t(performance.now())}document.readyState==="loading"?document.addEventListener("DOMContentLoaded",r):r()})()