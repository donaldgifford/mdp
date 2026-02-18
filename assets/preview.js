// mdp preview client — handles WebSocket connection and DOM updates.
"use strict";

(function () {
  const content = document.getElementById("content");
  const status = document.getElementById("connection-status");

  // Phase 2 will add WebSocket live-reload here.
  // For now this is a static page; the JS scaffold is ready for extension.

  /**
   * Update the preview content.
   * @param {string} html - Rendered HTML to display.
   */
  function updateContent(html) {
    if (content) {
      content.innerHTML = html;
    }
  }

  /**
   * Set connection status indicator.
   * @param {boolean} connected
   */
  function setConnectionStatus(connected) {
    if (!status) return;
    if (connected) {
      status.className = "connected";
    } else {
      status.className = "disconnected";
      status.textContent = "Disconnected — reconnecting\u2026";
    }
  }

  // Export for use by future WebSocket handler.
  window.mdp = { updateContent, setConnectionStatus };
})();
