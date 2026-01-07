(function(){const c=33;let l=!1,a=!0,t=new Set;const u=`
    attribute vec2 a_position;
    varying vec2 v_position;
    void main() {
      gl_Position = vec4(a_position, 0, 1);
      v_position = a_position;
    }
  `,h=`
    precision mediump float;
    uniform vec2 u_resolution;
    uniform float u_time;
    uniform float u_isDark;
    uniform float u_fadeTop;
    uniform float u_dpr;
    uniform float u_hover;
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

    vec2 metaball(vec2 p, float time) {
      vec2 i = floor(p);
      vec2 f = fract(p);
      float sum = 0.0;
      float accentSum = 0.0;

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
          float isAccent = step(0.92, hash(cellId + 0.5));
          float blobDelay = hash(cellId + 0.7) * 0.75;
          float blobVisible = smoothstep(blobDelay, blobDelay + 0.25, u_hover);
          accentSum += contrib * isAccent * blobVisible;
        }
      }
      float accentWeight = sum > 0.0 ? accentSum / sum : 0.0;
      return vec2(sum, accentWeight);
    }

    vec2 warpCoords(vec2 p, float time) {
      float warp1 = vnoise(p * 0.5 + time * 0.05);
      float warp2 = vnoise(p * 0.3 - time * 0.03 + 100.0);
      return p + vec2(warp1, warp2) * 0.4;
    }

    void main() {
      float scale = 0.0133;
      vec2 p = gl_FragCoord.xy * scale;

      vec3 lightBright = vec3(0.10, 0.42, 0.70);
      vec3 lightDark = vec3(0.04, 0.22, 0.44);
      vec3 lightPageBg = vec3(1.0, 1.0, 1.0);

      vec3 darkBright = vec3(0.32, 0.55, 0.82);
      vec3 darkDark = vec3(0.12, 0.24, 0.40);
      vec3 darkPageBg = vec3(0.106, 0.192, 0.337);

      vec3 lightAccent = vec3(0.804, 0.055, 0.055);
      vec3 darkAccent = vec3(1.0, 0.4, 0.4);

      vec3 bright = mix(lightBright, darkBright, u_isDark);
      vec3 dark = mix(lightDark, darkDark, u_isDark);
      vec3 pageBg = mix(lightPageBg, darkPageBg, u_isDark);
      vec3 accent = mix(lightAccent, darkAccent, u_isDark);

      float baseHeight = 50.0 * u_dpr;
      float waveAmplitude = 25.0 * u_dpr;
      float xCoord = gl_FragCoord.x / u_dpr;
      float boundary = baseHeight;
      boundary += vnoise(vec2(xCoord * 0.008, u_time * 0.08)) * waveAmplitude;
      boundary += vnoise(vec2(xCoord * 0.02, u_time * 0.04 + 50.0)) * waveAmplitude * 0.4;

      float pixelY = mix(gl_FragCoord.y, u_resolution.y - gl_FragCoord.y, u_fadeTop);
      float distToBoundary = pixelY - boundary;

      float wallRange = 40.0 * u_dpr;
      float wallInfluence = smoothstep(wallRange, 0.0, distToBoundary) * 0.25;

      vec2 p1 = p * 0.7 + vec2(u_time * 0.02, u_time * 0.015);
      vec2 meta1 = metaball(warpCoords(p1, u_time * 0.6), u_time * 0.6);
      float field1 = meta1.x + wallInfluence;
      float accent1 = meta1.y;

      vec2 p2 = p + vec2(u_time * 0.06, -u_time * 0.02);
      vec2 meta2 = metaball(warpCoords(p2 + 100.0, u_time), u_time);
      float field2 = meta2.x + wallInfluence;
      float accent2 = meta2.y;

      float taperRange = 30.0 * u_dpr;
      float taper = smoothstep(-taperRange, 0.0, distToBoundary);
      field1 *= taper;
      field2 *= taper;

      float aaWidth = 0.005 / max(u_dpr, 1.0);
      float blend1 = smoothstep(0.92 - aaWidth, 0.92 + aaWidth, field1);
      float blend2 = smoothstep(0.95 - aaWidth, 0.95 + aaWidth, field2);

      float isAccent1 = step(0.5, accent1);
      float isAccent2 = step(0.5, accent2);

      vec3 dark1 = mix(dark, accent, isAccent1 * u_hover);
      vec3 bright2 = mix(bright, accent, isAccent2 * u_hover);

      vec3 color = mix(pageBg, dark1, blend1);
      color = mix(color, bright2, blend2);

      gl_FragColor = vec4(color, 1.0);
    }
  `;function o(e,t,n){const s=e.createShader(t);return e.shaderSource(s,n),e.compileShader(s),s}function m(){return document.body.classList.contains("dark")?1:0}function i(e,t){const i=document.getElementById(e);if(!i)return null;const n=i.getContext("webgl",{antialias:!0});if(!n)return null;const r=o(n,n.VERTEX_SHADER,u),c=o(n,n.FRAGMENT_SHADER,h),s=n.createProgram();n.attachShader(s,r),n.attachShader(s,c),n.linkProgram(s),n.useProgram(s);const l=n.createBuffer();n.bindBuffer(n.ARRAY_BUFFER,l),n.bufferData(n.ARRAY_BUFFER,new Float32Array([-1,-1,1,-1,-1,1,1,1]),n.STATIC_DRAW);const a=n.getAttribLocation(s,"a_position");return n.enableVertexAttribArray(a),n.vertexAttribPointer(a,2,n.FLOAT,!1,0,0),{canvas:i,gl:n,resolutionLoc:n.getUniformLocation(s,"u_resolution"),timeLoc:n.getUniformLocation(s,"u_time"),isDarkLoc:n.getUniformLocation(s,"u_isDark"),fadeTopLoc:n.getUniformLocation(s,"u_fadeTop"),dprLoc:n.getUniformLocation(s,"u_dpr"),hoverLoc:n.getUniformLocation(s,"u_hover"),hoverTarget:0,hoverValue:0,needsResize:!0,fadeTop:t?1:0}}let e=[],f=performance.now(),r=0,s=window.devicePixelRatio;function p(){const e=window.devicePixelRatio;return e<=1?1.5:e}function n(o){if(!a||t.size===0){requestAnimationFrame(n);return}if(o-r<c){requestAnimationFrame(n);return}r=o;const u=(performance.now()-f)/1e3,h=m(),l=window.devicePixelRatio;l!==s&&(s=l,e.forEach(e=>e.needsResize=!0));const i=p(),d=c/1e3;for(const n of e){if(!t.has(n.canvas))continue;const{canvas:o,gl:s,resolutionLoc:r,timeLoc:c,isDarkLoc:l,fadeTopLoc:m,dprLoc:f,hoverLoc:p}=n;n.needsResize&&(o.width=o.offsetWidth*i,o.height=o.offsetHeight*i,s.viewport(0,0,o.width,o.height),s.uniform2f(r,o.width,o.height),s.uniform1f(m,n.fadeTop),n.needsResize=!1);const a=1/2.65;n.hoverTarget>n.hoverValue?n.hoverValue=Math.min(n.hoverTarget,n.hoverValue+d*a):n.hoverTarget<n.hoverValue&&(n.hoverValue=Math.max(n.hoverTarget,n.hoverValue-d*a)),s.uniform1f(c,u),s.uniform1f(l,h),s.uniform1f(f,i),s.uniform1f(p,n.hoverValue),s.drawArrays(s.TRIANGLE_STRIP,0,4)}requestAnimationFrame(n)}function d(){if(l)return;if(l=!0,e=[i("header-canvas",!1),i("footer-canvas",!0)].filter(Boolean),e.length===0)return;e.forEach(e=>{const t=e.canvas.parentElement;t&&(t.addEventListener("mouseenter",()=>e.hoverTarget=1),t.addEventListener("mouseleave",()=>e.hoverTarget=0))}),document.addEventListener("visibilitychange",()=>{a=!document.hidden});const s=new IntersectionObserver(e=>{e.forEach(e=>{e.isIntersecting?t.add(e.target):t.delete(e.target)})},{threshold:0});e.forEach(e=>s.observe(e.canvas)),window.addEventListener("resize",()=>e.forEach(e=>e.needsResize=!0)),n(performance.now())}document.readyState==="loading"?document.addEventListener("DOMContentLoaded",d):d()})()