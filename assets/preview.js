// mdp preview client — handles WebSocket/SSE connection, DOM updates, and scroll sync.
"use strict";

(function () {
  var content = document.getElementById("content");
  var statusEl = document.getElementById("connection-status");
  var reconnectDelay = 250;
  var maxReconnectDelay = 5000;
  var useSSE = false;

  // ---------------------------------------------------------------------
  // Mermaid diagram skin (DESIGN-0004)
  //
  // Everything that is not a colour is constant here, identical for every
  // theme. Colours come from the theme's seven --mermaid-* slots, or are
  // derived from its --color-* prose properties when the slots are absent.
  // The functions below are pure (no DOM access) so a JS test harness
  // (#77) can cover them directly.
  // ---------------------------------------------------------------------

  // Diagram font stacks. Inter and JetBrains Mono are vendored and declared
  // with @font-face in preview.css; they are used inside diagrams only.
  var SKIN = {
    font: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    mono: '"JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace',
  };

  // mixHex mixes fg into bg at pct percent per sRGB channel and returns a
  // six-digit lowercase hex. Same arithmetic as CSS
  // color-mix(in srgb, fg pct%, bg), which beautiful-mermaid uses for its
  // derived slots and which produced the DESIGN-0004 seed table.
  function mixHex(fg, bg, pct) {
    var out = "#";
    for (var i = 1; i < 7; i += 2) {
      var f = parseInt(fg.slice(i, i + 2), 16);
      var b = parseInt(bg.slice(i, i + 2), 16);
      var c = Math.round((f * pct + b * (100 - pct)) / 100);
      out += (c < 16 ? "0" : "") + c.toString(16);
    }
    return out;
  }

  // readPalette returns the seven diagram colour slots from a computed
  // style. A theme may define any subset of --mermaid-bg/-fg/-line/-accent/
  // -muted/-surface/-border; each missing slot is derived from the required
  // --color-* prose properties with the same mixes as the DESIGN-0004 seed
  // table. The auto theme and custom theme files without slots therefore
  // get a coherent palette with no extra CSS.
  function readPalette(style) {
    function prop(name) {
      return style.getPropertyValue(name).trim();
    }
    var p = {
      bg: prop("--mermaid-bg") || prop("--color-canvas-default"),
      fg: prop("--mermaid-fg") || prop("--color-fg-default"),
      muted: prop("--mermaid-muted") || prop("--color-fg-muted"),
      accent: prop("--mermaid-accent") || prop("--color-accent-fg"),
    };
    p.line = prop("--mermaid-line") || mixHex(p.fg, p.bg, 50);
    p.surface = prop("--mermaid-surface") || mixHex(p.fg, p.bg, 3);
    p.border = prop("--mermaid-border") || mixHex(p.fg, p.bg, 20);
    return p;
  }

  // expandPalette maps the seven slots onto Mermaid's base-theme variables
  // (the DESIGN-0004 "Palette expansion" table) and adds the fixed
  // geometry. useGradient and dropShadow are what the neo look reads to
  // decide on gradient strokes and the drop-shadow filter.
  function expandPalette(p) {
    return {
      // bg
      background: p.bg,
      edgeLabelBackground: p.bg,
      labelBackgroundColor: p.bg,
      // fg
      primaryTextColor: p.fg,
      textColor: p.fg,
      nodeTextColor: p.fg,
      titleColor: p.fg,
      actorTextColor: p.fg,
      signalTextColor: p.fg,
      labelTextColor: p.fg,
      loopTextColor: p.fg,
      noteTextColor: p.fg,
      classText: p.fg,
      stateLabelColor: p.fg,
      transitionLabelColor: p.fg,
      // line
      lineColor: p.line,
      defaultLinkColor: p.line,
      signalColor: p.line,
      actorLineColor: p.line,
      transitionColor: p.line,
      relationColor: p.line,
      archEdgeColor: p.line,
      // accent
      arrowheadColor: p.accent,
      archEdgeArrowColor: p.accent,
      activationBorderColor: p.accent,
      specialStateColor: p.accent,
      // muted
      secondaryTextColor: p.muted,
      tertiaryTextColor: p.muted,
      sequenceNumberColor: p.muted,
      // surface
      primaryColor: p.surface,
      secondaryColor: p.surface,
      tertiaryColor: p.surface,
      nodeBkg: p.surface,
      mainBkg: p.surface,
      actorBkg: p.surface,
      noteBkgColor: p.surface,
      labelBoxBkgColor: p.surface,
      activationBkgColor: p.surface,
      clusterBkg: p.surface,
      stateBkg: p.surface,
      compositeBackground: p.surface,
      altBackground: p.surface,
      attributeBackgroundColorOdd: p.surface,
      attributeBackgroundColorEven: p.surface,
      requirementBackground: p.surface,
      // border
      primaryBorderColor: p.border,
      secondaryBorderColor: p.border,
      tertiaryBorderColor: p.border,
      nodeBorder: p.border,
      clusterBorder: p.border,
      actorBorder: p.border,
      noteBorderColor: p.border,
      labelBoxBorderColor: p.border,
      compositeBorder: p.border,
      requirementBorderColor: p.border,
      archGroupBorderColor: p.border,
      // fixed geometry and typography
      useGradient: false,
      dropShadow: "none",
      strokeWidth: 1,
      radius: 6,
      fontFamily: SKIN.font,
      fontSize: "13px",
    };
  }

  // Initialize Mermaid with theme detection.
  if (typeof mermaid !== "undefined") {
    // Register Iconify icon packs for `pack:icon` references in architecture
    // diagrams. Packs download lazily on first use only; offline diagrams
    // render with a fallback glyph.
    if (typeof mermaid.registerIconPacks === "function") {
      try {
        mermaid.registerIconPacks([
          {
            name: "logos",
            loader: () =>
              fetch("https://cdn.jsdelivr.net/npm/@iconify-json/logos@1/icons.json").then((res) => res.json()),
          },
          {
            name: "devicon",
            loader: () =>
              fetch("https://cdn.jsdelivr.net/npm/@iconify-json/devicon@1/icons.json").then((res) => res.json()),
          },
          {
            name: "k8s",
            loader: () =>
              fetch("https://cdn.jsdelivr.net/npm/@iconify-json/k8s@1/icons.json").then((res) => res.json()),
          },
        ]);
      } catch (e) {
        console.warn("mermaid icon pack registration failed:", e);
      }
    }
    var prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    var mermaidTheme = document.body.dataset.mermaidTheme;
    // "dagre" when the dagre escape hatch is set, else "" for the Mermaid v12 ELK default.
    var mermaidLayout = document.body.dataset.mermaidLayout;
    var mermaidInit = { startOnLoad: false };
    if (mermaidLayout) {
      mermaidInit.layout = mermaidLayout;
    }
    if (mermaidTheme === "base") {
      // Named built-in theme: read --mermaid-* CSS custom properties that the
      // theme stylesheet defines on [data-theme] / body.
      var bodyStyle = getComputedStyle(document.body);
      var themeVariables = {
        primaryColor:        bodyStyle.getPropertyValue("--mermaid-primaryColor").trim(),
        primaryTextColor:    bodyStyle.getPropertyValue("--mermaid-primaryTextColor").trim(),
        primaryBorderColor:  bodyStyle.getPropertyValue("--mermaid-primaryBorderColor").trim(),
        lineColor:           bodyStyle.getPropertyValue("--mermaid-lineColor").trim(),
        secondaryColor:      bodyStyle.getPropertyValue("--mermaid-secondaryColor").trim(),
        tertiaryColor:       bodyStyle.getPropertyValue("--mermaid-tertiaryColor").trim(),
        background:          bodyStyle.getPropertyValue("--mermaid-background").trim(),
        noteBkgColor:        bodyStyle.getPropertyValue("--mermaid-noteBkgColor").trim(),
        noteTextColor:       bodyStyle.getPropertyValue("--mermaid-noteTextColor").trim(),
        edgeLabelBackground: bodyStyle.getPropertyValue("--mermaid-edgeLabelBackground").trim(),
        actorBkg:            bodyStyle.getPropertyValue("--mermaid-actorBkg").trim(),
        actorTextColor:      bodyStyle.getPropertyValue("--mermaid-actorTextColor").trim(),
      };
      mermaidInit.theme = "base";
      mermaidInit.themeVariables = themeVariables;
      mermaid.initialize(mermaidInit);
    } else {
      // auto: fall back to prefers-color-scheme for Mermaid theme selection.
      mermaidInit.theme = prefersDark ? "dark" : "default";
      mermaid.initialize(mermaidInit);
    }
  }

  // Run all client-side rendering after content update.
  function renderClientSide() {
    // Mermaid: re-render diagram blocks.
    if (typeof mermaid !== "undefined") {
      // Remove previous Mermaid SVG output so re-init works cleanly.
      var rendered = content.querySelectorAll(".mermaid[data-processed]");
      for (var i = 0; i < rendered.length; i++) {
        rendered[i].removeAttribute("data-processed");
      }
      try {
        mermaid.run({ nodes: content.querySelectorAll(".mermaid") });
      } catch (e) {
        console.warn("mermaid render error:", e);
      }
    }

    // KaTeX: render math expressions.
    if (typeof renderMathInElement !== "undefined") {
      try {
        renderMathInElement(content, {
          delimiters: [
            { left: "$$", right: "$$", display: true },
            { left: "$", right: "$", display: false }
          ],
          throwOnError: false
        });
      } catch (e) {
        console.warn("katex render error:", e);
      }
    }

    // highlight.js: highlight un-highlighted code blocks.
    if (typeof hljs !== "undefined") {
      var blocks = content.querySelectorAll("pre code:not(.hljs)");
      for (var j = 0; j < blocks.length; j++) {
        hljs.highlightElement(blocks[j]);
      }
    }
  }

  // Rewrite relative image paths to use the /local/ prefix.
  function rewriteImagePaths() {
    if (!content) return;
    var imgs = content.querySelectorAll("img");
    for (var i = 0; i < imgs.length; i++) {
      var src = imgs[i].getAttribute("src");
      if (src && !src.match(/^(https?:|\/|data:)/)) {
        imgs[i].setAttribute("src", "/local/" + src);
      }
    }
  }

  // Run on initial page load.
  renderClientSide();
  rewriteImagePaths();

  // --- Scroll Sync ---

  // Find the nearest element with data-source-line <= targetLine.
  function findScrollTarget(targetLine) {
    if (!content) return null;

    // Footnote definitions render into a trailing .footnotes list but
    // keep the source line where they were defined, so their
    // data-source-line values are out of document order. Two shapes
    // produce this: a definition placed mid-document, and
    // first-reference numbering when reference order differs from
    // definition order. Excluding the subtree keeps the remaining
    // values non-decreasing, which the `break` below relies on --
    // without it, a cursor line past a definition selects the footnote
    // instead of the intended block.
    var elements = content.querySelectorAll(
      "[data-source-line]:not(.footnotes [data-source-line])"
    );
    var best = null;

    for (var i = 0; i < elements.length; i++) {
      var line = parseInt(elements[i].getAttribute("data-source-line"), 10);
      if (isNaN(line)) continue;
      if (line <= targetLine) {
        best = elements[i];
      } else {
        break;
      }
    }

    return best;
  }

  var highlightTimer = null;

  function scrollToLine(line) {
    if (line <= 0) return;

    var target = findScrollTarget(line);

    if (!target) {
      // Cursor past end of document — scroll to bottom.
      window.scrollTo({ top: document.body.scrollHeight, behavior: "smooth" });
      return;
    }

    target.scrollIntoView({ behavior: "smooth", block: "center" });

    // Brief highlight for visual feedback.
    target.classList.add("scroll-target");
    if (highlightTimer) clearTimeout(highlightTimer);
    // Remove previous highlights.
    var prev = content.querySelectorAll(".scroll-target");
    for (var i = 0; i < prev.length; i++) {
      if (prev[i] !== target) prev[i].classList.remove("scroll-target");
    }
    highlightTimer = setTimeout(function () {
      target.classList.remove("scroll-target");
    }, 1000);
  }

  // --- Message Handling ---

  function handleMessage(raw) {
    var msg;
    try {
      msg = JSON.parse(raw);
    } catch (e) {
      // Fallback: treat as raw HTML for backward compatibility.
      updateContent(raw);
      return;
    }

    if (msg.type === "content") {
      updateContent(msg.html);
    } else if (msg.type === "cursor") {
      scrollToLine(msg.line);
    }
  }

  function connectWebSocket() {
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    var ws = new WebSocket(proto + "//" + location.host + "/ws");

    ws.onopen = function () {
      reconnectDelay = 250;
      useSSE = false;
      setConnectionStatus(true);
    };

    ws.onmessage = function (event) {
      handleMessage(event.data);
    };

    ws.onclose = function () {
      setConnectionStatus(false);
      scheduleReconnect();
    };

    ws.onerror = function () {
      useSSE = true;
      ws.close();
    };
  }

  function connectSSE() {
    var source = new EventSource("/events");

    source.onopen = function () {
      reconnectDelay = 250;
      setConnectionStatus(true);
    };

    source.onmessage = function (event) {
      handleMessage(event.data);
    };

    source.onerror = function () {
      source.close();
      setConnectionStatus(false);
      scheduleReconnect();
    };
  }

  function connect() {
    if (useSSE) {
      connectSSE();
    } else {
      connectWebSocket();
    }
  }

  function scheduleReconnect() {
    setTimeout(function () {
      reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
      connect();
    }, reconnectDelay);
  }

  function updateContent(html) {
    if (content) {
      content.innerHTML = html;
      renderClientSide();
      rewriteImagePaths();
    }
  }

  function setConnectionStatus(connected) {
    if (!statusEl) return;
    if (connected) {
      statusEl.className = "connected";
    } else {
      statusEl.className = "disconnected";
      statusEl.textContent = "Disconnected \u2014 reconnecting\u2026";
    }
  }

  // Connect on page load.
  connect();
})();
