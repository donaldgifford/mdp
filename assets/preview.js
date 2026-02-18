// mdp preview client — handles WebSocket/SSE connection and DOM updates.
"use strict";

(function () {
  var content = document.getElementById("content");
  var statusEl = document.getElementById("connection-status");
  var reconnectDelay = 250;
  var maxReconnectDelay = 5000;
  var useSSE = false;

  // Initialize Mermaid with theme detection.
  if (typeof mermaid !== "undefined") {
    var prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    var theme = document.body.getAttribute("data-theme");
    var mermaidTheme = "default";
    if (theme === "dark" || (theme === "auto" && prefersDark)) {
      mermaidTheme = "dark";
    }
    mermaid.initialize({ startOnLoad: false, theme: mermaidTheme });
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

  // Run on initial page load.
  renderClientSide();

  function connectWebSocket() {
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    var ws = new WebSocket(proto + "//" + location.host + "/ws");

    ws.onopen = function () {
      reconnectDelay = 250;
      useSSE = false;
      setConnectionStatus(true);
    };

    ws.onmessage = function (event) {
      updateContent(event.data);
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
      updateContent(event.data);
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
