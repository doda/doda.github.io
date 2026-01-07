// Wave animation effect (WebGL metaballs)
// Adapted from lucumr.pocoo.org
(function() {
  // Respect reduced motion preference
  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  if (prefersReducedMotion) return;

  const FRAME_INTERVAL = 42;  // 24fps
  const FRAME_INTERVAL_BACKGROUND = 100;  // ~10fps when not focused
  let initialized = false;
  let isPageVisible = true;
  let isWindowFocused = true;
  let visibleCanvases = new Set();

  const vsSource = `
    attribute vec2 a_position;
    varying vec2 v_position;
    void main() {
      gl_Position = vec4(a_position, 0, 1);
      v_position = a_position;
    }
  `;

  const fsSource = `
    precision mediump float;
    uniform vec2 u_resolution;
    uniform float u_time;
    uniform float u_isDark;
    uniform float u_fadeTop;
    uniform float u_dpr;
    uniform vec3 u_bgColor;
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

      // Derive layer colors from actual background color
      vec3 pageBg = u_bgColor;

      // In dark mode, layers go lighter; in light mode, layers go darker
      float direction = u_isDark > 0.5 ? 1.0 : -1.0;
      vec3 layer1 = clamp(pageBg + direction * vec3(0.02), 0.0, 1.0);
      vec3 layer2 = clamp(pageBg + direction * vec3(0.05), 0.0, 1.0);
      vec3 layer3 = clamp(pageBg + direction * vec3(0.09), 0.0, 1.0);

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
        // Threshold: tight (0.95) to loose (0.4) - front layers fill more
        float threshold = 0.95 - t * 0.55;
        // Offset for variety
        vec2 offset = vec2(fi * 17.3, fi * 23.7);

        // Each layer has slightly different boundary offset
        float layerBoundaryOffset = (fi - 2.0) * 8.0 * u_dpr;
        float layerBoundary = boundary + layerBoundaryOffset;
        float layerDistToBoundary = pixelY - layerBoundary;
        float layerWallInfluence = smoothstep(wallRange, 0.0, layerDistToBoundary) * 0.25;
        float layerTaper = smoothstep(-taperRange, 0.0, layerDistToBoundary);

        vec2 pLayer = p * scale + vec2(u_time * speed * 0.05, u_time * speed * 0.03) + offset;
        vec2 meta = metaball(warpCoords(pLayer, u_time * speed), u_time * speed);
        float field = (meta.x + layerWallInfluence) * layerTaper;
        float blend = smoothstep(threshold - aaWidth, threshold + aaWidth, field);

        // Interpolate color through the pink palette
        vec3 layerColor = mix(layer1, layer3, t);
        float variation = 1.0 + (meta.y - 0.5) * varStrength;

        color = mix(color, layerColor * variation, blend);
      }

      gl_FragColor = vec4(color, 1.0);
    }
  `;

  function createShader(gl, type, source) {
    const shader = gl.createShader(type);
    gl.shaderSource(shader, source);
    gl.compileShader(shader);
    return shader;
  }

  function getIsDark() {
    return document.documentElement.getAttribute('data-theme') === 'dark' ? 1.0 : 0.0;
  }

  function getBgColor() {
    const style = getComputedStyle(document.body);
    const bg = style.backgroundColor;
    const match = bg.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
    if (match) {
      return [parseInt(match[1]) / 255, parseInt(match[2]) / 255, parseInt(match[3]) / 255];
    }
    return [1.0, 1.0, 1.0]; // fallback to white
  }

  function initWaterEffect(canvasId, fadeTop) {
    const canvas = document.getElementById(canvasId);
    if (!canvas) return null;
    // Try WebGL2 first, fall back to WebGL1
    const gl = canvas.getContext('webgl2', { antialias: true, powerPreference: 'low-power' })
            || canvas.getContext('webgl', { antialias: true, powerPreference: 'low-power' });
    if (!gl) return null;

    const vs = createShader(gl, gl.VERTEX_SHADER, vsSource);
    const fs = createShader(gl, gl.FRAGMENT_SHADER, fsSource);
    const program = gl.createProgram();
    gl.attachShader(program, vs);
    gl.attachShader(program, fs);
    gl.linkProgram(program);
    gl.useProgram(program);

    const posBuffer = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, posBuffer);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1,-1, 1,-1, -1,1, 1,1]), gl.STATIC_DRAW);
    const posLoc = gl.getAttribLocation(program, 'a_position');
    gl.enableVertexAttribArray(posLoc);
    gl.vertexAttribPointer(posLoc, 2, gl.FLOAT, false, 0, 0);

    return {
      canvas, gl,
      resolutionLoc: gl.getUniformLocation(program, 'u_resolution'),
      timeLoc: gl.getUniformLocation(program, 'u_time'),
      isDarkLoc: gl.getUniformLocation(program, 'u_isDark'),
      fadeTopLoc: gl.getUniformLocation(program, 'u_fadeTop'),
      dprLoc: gl.getUniformLocation(program, 'u_dpr'),
      hoverLoc: gl.getUniformLocation(program, 'u_hover'),
      bgColorLoc: gl.getUniformLocation(program, 'u_bgColor'),
      hoverTarget: 0, hoverValue: 0, needsResize: true,
      fadeTop: fadeTop ? 1.0 : 0.0
    };
  }

  let effects = [];
  let startTime = performance.now();
  let lastFrameTime = 0;
  let currentDpr = window.devicePixelRatio;

  function getEffectiveDpr() {
    const dpr = window.devicePixelRatio;
    return dpr <= 1 ? 1.5 : dpr;
  }

  function render(timestamp) {
    if (!isPageVisible || visibleCanvases.size === 0) {
      requestAnimationFrame(render);
      return;
    }
    const frameInterval = isWindowFocused ? FRAME_INTERVAL : FRAME_INTERVAL_BACKGROUND;
    if (timestamp - lastFrameTime < frameInterval) {
      requestAnimationFrame(render);
      return;
    }
    lastFrameTime = timestamp;

    const elapsed = (performance.now() - startTime) / 1000.0;
    const isDark = getIsDark();
    const bgColor = getBgColor();
    const newDpr = window.devicePixelRatio;
    if (newDpr !== currentDpr) {
      currentDpr = newDpr;
      effects.forEach(e => e.needsResize = true);
    }
    const effectiveDpr = getEffectiveDpr();
    const deltaTime = FRAME_INTERVAL / 1000.0;

    for (const effect of effects) {
      if (!visibleCanvases.has(effect.canvas)) continue;

      const { canvas, gl, resolutionLoc, timeLoc, isDarkLoc, fadeTopLoc, dprLoc, hoverLoc, bgColorLoc } = effect;

      if (effect.needsResize) {
        canvas.width = canvas.offsetWidth * effectiveDpr;
        canvas.height = canvas.offsetHeight * effectiveDpr;
        gl.viewport(0, 0, canvas.width, canvas.height);
        gl.uniform2f(resolutionLoc, canvas.width, canvas.height);
        gl.uniform1f(fadeTopLoc, effect.fadeTop);
        effect.needsResize = false;
      }

      const hoverSpeed = 1.0 / 2.65;
      if (effect.hoverTarget > effect.hoverValue) {
        effect.hoverValue = Math.min(effect.hoverTarget, effect.hoverValue + deltaTime * hoverSpeed);
      } else if (effect.hoverTarget < effect.hoverValue) {
        effect.hoverValue = Math.max(effect.hoverTarget, effect.hoverValue - deltaTime * hoverSpeed);
      }

      gl.uniform1f(timeLoc, elapsed);
      gl.uniform1f(isDarkLoc, isDark);
      gl.uniform3f(bgColorLoc, bgColor[0], bgColor[1], bgColor[2]);
      gl.uniform1f(dprLoc, effectiveDpr);
      gl.uniform1f(hoverLoc, effect.hoverValue);
      gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
    }
    requestAnimationFrame(render);
  }

  function init() {
    if (initialized) return;
    initialized = true;

    effects = [
      initWaterEffect('footer-canvas', true)
    ].filter(Boolean);

    if (effects.length === 0) return;

    effects.forEach(effect => {
      const container = effect.canvas.parentElement;
      if (container) {
        container.addEventListener('mouseenter', () => effect.hoverTarget = 1);
        container.addEventListener('mouseleave', () => effect.hoverTarget = 0);
      }
    });

    document.addEventListener('visibilitychange', () => {
      isPageVisible = !document.hidden;
    });

    window.addEventListener('focus', () => { isWindowFocused = true; });
    window.addEventListener('blur', () => { isWindowFocused = false; });

    const observer = new IntersectionObserver(entries => {
      entries.forEach(entry => {
        if (entry.isIntersecting) visibleCanvases.add(entry.target);
        else visibleCanvases.delete(entry.target);
      });
    }, { threshold: 0 });

    effects.forEach(e => observer.observe(e.canvas));
    window.addEventListener('resize', () => effects.forEach(e => e.needsResize = true));
    render(performance.now());
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
