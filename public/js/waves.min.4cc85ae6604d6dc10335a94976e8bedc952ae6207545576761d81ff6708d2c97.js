(function(){const a=33;let o=!1,i=!0,t=new Set;const d=`
    attribute vec2 a_position;
    varying vec2 v_position;
    void main() {
      gl_Position = vec4(a_position, 0, 1);
      v_position = a_position;
    }
  `,p=`
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

      // Monochrome palette - light range, 15 distinct shades
      vec3 lightPageBg = vec3(1.0, 1.0, 1.0);
      vec3 lightLayer1 = vec3(0.95, 0.95, 0.95);  // Lightest (back)
      vec3 lightLayer2 = vec3(0.90, 0.90, 0.90);  // Mid
      vec3 lightLayer3 = vec3(0.80, 0.80, 0.80);  // Darkest (front)

      vec3 darkPageBg = vec3(0.10, 0.10, 0.10);
      vec3 darkLayer1 = vec3(0.18, 0.18, 0.18);   // Lightest
      vec3 darkLayer2 = vec3(0.25, 0.25, 0.25);   // Mid
      vec3 darkLayer3 = vec3(0.35, 0.35, 0.35);   // Darkest

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

      // 15 layers from back to front
      for (int i = 0; i < 15; i++) {
        float fi = float(i);
        float t = fi / 14.0;  // 0 to 1

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
  `;function c(e,t,n){const s=e.createShader(t);return e.shaderSource(s,n),e.compileShader(s),s}function f(){return document.body.classList.contains("dark")?1:0}function h(e,t){const o=document.getElementById(e);if(!o)return null;const n=o.getContext("webgl",{antialias:!0});if(!n)return null;const a=c(n,n.VERTEX_SHADER,d),r=c(n,n.FRAGMENT_SHADER,p),s=n.createProgram();n.attachShader(s,a),n.attachShader(s,r),n.linkProgram(s),n.useProgram(s);const l=n.createBuffer();n.bindBuffer(n.ARRAY_BUFFER,l),n.bufferData(n.ARRAY_BUFFER,new Float32Array([-1,-1,1,-1,-1,1,1,1]),n.STATIC_DRAW);const i=n.getAttribLocation(s,"a_position");return n.enableVertexAttribArray(i),n.vertexAttribPointer(i,2,n.FLOAT,!1,0,0),{canvas:o,gl:n,resolutionLoc:n.getUniformLocation(s,"u_resolution"),timeLoc:n.getUniformLocation(s,"u_time"),isDarkLoc:n.getUniformLocation(s,"u_isDark"),fadeTopLoc:n.getUniformLocation(s,"u_fadeTop"),dprLoc:n.getUniformLocation(s,"u_dpr"),hoverLoc:n.getUniformLocation(s,"u_hover"),hoverTarget:0,hoverValue:0,needsResize:!0,fadeTop:t?1:0}}let e=[],u=performance.now(),s=0,l=window.devicePixelRatio;function m(){const e=window.devicePixelRatio;return e<=1?1.5:e}function n(o){if(!i||t.size===0){requestAnimationFrame(n);return}if(o-s<a){requestAnimationFrame(n);return}s=o;const h=(performance.now()-u)/1e3,p=f(),c=window.devicePixelRatio;c!==l&&(l=c,e.forEach(e=>e.needsResize=!0));const r=m(),d=a/1e3;for(const n of e){if(!t.has(n.canvas))continue;const{canvas:o,gl:s,resolutionLoc:a,timeLoc:c,isDarkLoc:l,fadeTopLoc:u,dprLoc:m,hoverLoc:f}=n;n.needsResize&&(o.width=o.offsetWidth*r,o.height=o.offsetHeight*r,s.viewport(0,0,o.width,o.height),s.uniform2f(a,o.width,o.height),s.uniform1f(u,n.fadeTop),n.needsResize=!1);const i=1/2.65;n.hoverTarget>n.hoverValue?n.hoverValue=Math.min(n.hoverTarget,n.hoverValue+d*i):n.hoverTarget<n.hoverValue&&(n.hoverValue=Math.max(n.hoverTarget,n.hoverValue-d*i)),s.uniform1f(c,h),s.uniform1f(l,p),s.uniform1f(m,r),s.uniform1f(f,n.hoverValue),s.drawArrays(s.TRIANGLE_STRIP,0,4)}requestAnimationFrame(n)}function r(){if(o)return;if(o=!0,e=[h("footer-canvas",!0)].filter(Boolean),e.length===0)return;e.forEach(e=>{const t=e.canvas.parentElement;t&&(t.addEventListener("mouseenter",()=>e.hoverTarget=1),t.addEventListener("mouseleave",()=>e.hoverTarget=0))}),document.addEventListener("visibilitychange",()=>{i=!document.hidden});const s=new IntersectionObserver(e=>{e.forEach(e=>{e.isIntersecting?t.add(e.target):t.delete(e.target)})},{threshold:0});e.forEach(e=>s.observe(e.canvas)),window.addEventListener("resize",()=>e.forEach(e=>e.needsResize=!0)),n(performance.now())}document.readyState==="loading"?document.addEventListener("DOMContentLoaded",r):r()})()