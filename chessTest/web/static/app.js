// path: chessTest/web/static/app.js
(function () {
  "use strict";

  // ===== Bootstrap & shared state =====
  const initScript = document.getElementById("__init");
  const init = initScript ? JSON.parse(initScript.textContent || "{}") : {};
  const defaultState = {
    pieces: [],        // [{id, type, color, square, element?}]
    blockFacing: {},   // { [pieceId]: directionIndex }
    abilities: {},
    elements: {},
    turn: 0,
    turnName: "white",
    lastNote: "",
    locked: false
  };
  let state = Object.assign({}, defaultState, init.state || {});
  state.locked = !!state.locked;
  let selectedSquare = null;   // 0..63
  let possibleMoves = [];      // UI hint only
  let isAnimating = false;

  // ===== DOM refs =====
  const boardEl = document.getElementById("board");
  const turnLabel = document.getElementById("turnLabel");
  const noteLabel = document.getElementById("noteLabel");
  const selectedLabel = document.getElementById("selectedSquare");
  const hoverLabel = document.getElementById("hoverSquare");
  const moveForm = document.getElementById("moveForm");
  const moveError = document.getElementById("moveError");
  const blockSummary = document.getElementById("blockSummary");
  const resetBtn = document.getElementById("resetBtn");
  const configForms = document.querySelectorAll(".config-form");
  const configMessage = document.getElementById("configMessage");
  const blockDirOverlay = document.getElementById("blockDirOverlay");

  // New UI hooks (index.html update)
  const abilityAnnounce = document.getElementById("abilityAnnounce");
  const abilityToastContainer = document.getElementById("abilityToastContainer");
  const eventFeed = document.getElementById("eventFeed");
  const moveList = document.getElementById("moveList");
  const logItemTpl = document.getElementById("logItemTpl");
  const toastTpl = document.getElementById("toastTpl");

  // Directions (engine uses 0..7)
  const DIRS = ["N","NE","E","SE","S","SW","W","NW"];

  let pendingMove = null;
  let pendingBlockDir = "";

  // ===== Tiny SFX =====
  const sounds = {
    select: () => playTone(800, 80),
    move: () => playTone(600, 140),
    capture: () => playTone(420, 160),
    error: () => playTone(220, 200),
  };
  function playTone(freq, duration) {
    const Ctx = window.AudioContext || window.webkitAudioContext;
    if (!Ctx) return;
    try {
      const ctx = new Ctx();
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.connect(gain);
      gain.connect(ctx.destination);
      osc.frequency.value = freq;
      gain.gain.setValueAtTime(0.12, ctx.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + duration/1000);
      osc.start(ctx.currentTime);
      osc.stop(ctx.currentTime + duration/1000);
    } catch (_) {}
  }

  // ===== Rendering =====
  function createPieceElement(piece) {
    // Unicode set + element badge via CSS class
    const colorName = String(piece.colorName || piece.color || "").toLowerCase();
    const isWhite = colorName === "white" || piece.color === 0;
    const t = String(piece.typeName || piece.type || "").toUpperCase();
    const glyph = (function () {
      switch (t) {
        case "K": return isWhite ? "♔" : "♚";
        case "Q": return isWhite ? "♕" : "♛";
        case "R": return isWhite ? "♖" : "♜";
        case "B": return isWhite ? "♗" : "♝";
        case "N": return isWhite ? "♘" : "♞";
        case "P": return isWhite ? "♙" : "♟";
        default:  return "●";
      }
    })();
    const el = document.createElement("div");
    el.className = "piece";
    el.textContent = glyph;
    el.dataset.id = piece.id;
    el.dataset.color = isWhite ? "white" : "black";
    const elementName = String(piece.elementName || piece.element || "").toLowerCase();
    if (elementName && elementName !== "none") {
      el.classList.add("element-" + elementName);
      el.dataset.element = elementName;
    }
    const colorLabel = piece.colorName ? capitalize(piece.colorName) : (isWhite ? "White" : "Black");
    const typeLabel = getPieceTypeName(piece.typeName || piece.type);
    el.setAttribute("title", `${colorLabel} ${typeLabel}`);
    // Show BlockPath indicator
    const facing = state.blockFacing && state.blockFacing[piece.id];
    if (facing !== undefined) {
      el.setAttribute("title", `${colorLabel} ${typeLabel} • BlockPath:${DIRS[facing] ?? "?"}`);
    }
    return el.outerHTML;
  }

  function renderBlockSummary() {
    if (!blockSummary) return;
    blockSummary.innerHTML = "";
    const entries = Object.entries(state.blockFacing || {});
    for (const [pid, dir] of entries) {
      const li = document.createElement("li");
      li.textContent = `Piece ${pid}: ${DIRS[dir] ?? "?"}`;
      blockSummary.appendChild(li);
    }
  }

  function renderBoard() {
    const overlayEl = blockDirOverlay;
    boardEl.innerHTML = "";
    for (let rank = 7; rank >= 0; rank--) {
      for (let file = 0; file < 8; file++) {
        const sqIndex = rank * 8 + file;
        const sq = document.createElement("div");
        sq.dataset.sq = String(sqIndex);
        sq.setAttribute("role","gridcell");
        sq.setAttribute("aria-label", sqToAlg(sqIndex));

        let className = "square " + ((rank + file) % 2 === 0 ? "light" : "dark");
        if (selectedSquare === sqIndex) className += " selected";
        if (possibleMoves.includes(sqIndex)) className += " highlight";
        sq.className = className;

        // Piece on this square?
        const piece = state.pieces.find(p => p.square === sqIndex);
        if (piece) {
          sq.innerHTML = createPieceElement(piece);
        }

        // Hover tooltip
        const colorLabel = piece && (piece.colorName ? capitalize(piece.colorName) : (piece.color === 0 ? "White" : "Black"));
        const typeLabel = piece && getPieceTypeName(piece.typeName || piece.type);
        sq.title = piece
          ? `${colorLabel} ${typeLabel}`
          : sqToAlg(sqIndex);

        // Events
        sq.addEventListener("click", () => onSquareClick(sqIndex));
        sq.addEventListener("mouseenter", () => {
          if (hoverLabel) hoverLabel.textContent = sqToAlg(sqIndex);
        });
        boardEl.appendChild(sq);
      }
    }
    if (overlayEl) {
      boardEl.appendChild(overlayEl);
    }
    // Labels
    if (turnLabel) turnLabel.textContent = state.turnName ? capitalize(state.turnName) : getTurnName(state.turn);
    if (noteLabel) noteLabel.textContent = state.note || state.lastNote || "Ready";
    if (selectedLabel) selectedLabel.textContent = selectedSquare !== null ? sqToAlg(selectedSquare) : "—";
    renderBlockSummary();
  }

  // ===== Event/UI logic =====
  function onSquareClick(sqIndex) {
    if (isAnimating) return;

    const piece = state.pieces.find(p => p.square === sqIndex);
    if (selectedSquare === null) {
      // Select if piece belongs to side to move
      if (piece && isPieceTurn(piece)) {
        selectedSquare = sqIndex;
        sounds.select();
        updateMoveHints();
      }
    } else if (selectedSquare === sqIndex) {
      // Deselect
      selectedSquare = null;
      possibleMoves = [];
    } else {
      // Treat as destination
      const fromAlg = sqToAlg(selectedSquare);
      const toAlg = sqToAlg(sqIndex);
      // If BlockPath required, ensure a direction chosen
      const movingPiece = state.pieces.find(p => p.square === selectedSquare);
      if (movingPiece && needsBlockPathDirection(movingPiece)) {
        prepareBlockDirSelection(movingPiece, selectedSquare, sqIndex);
        return;
      }
      submitMove(fromAlg, toAlg, "");
    }
    renderBoard();
  }

  function prepareBlockDirSelection(piece, fromSqIndex, toSqIndex) {
    if (!blockDirOverlay) {
      showMoveError("🛡️ Direction selector unavailable.");
      sounds.error();
      return;
    }
    pendingMove = { from: sqToAlg(fromSqIndex), to: sqToAlg(toSqIndex) };
    pendingBlockDir = "";
    possibleMoves = [];
    renderBoard();
    showBlockDirOverlay(piece, fromSqIndex, toSqIndex);
  }

  function showBlockDirOverlay(piece, fromSqIndex, toSqIndex) {
    if (!blockDirOverlay) return;
    blockDirOverlay.innerHTML = "";
    blockDirOverlay.hidden = false;
    blockDirOverlay.setAttribute("aria-hidden", "false");
    blockDirOverlay.classList.add("active");
    blockDirOverlay.tabIndex = -1;

    const headingId = "blockDirOverlayHeading";
    const infoId = "blockDirOverlayInfo";
    blockDirOverlay.setAttribute("aria-labelledby", headingId);
    blockDirOverlay.setAttribute("aria-describedby", infoId);

    const panel = document.createElement("div");
    panel.className = "block-dir-panel";

    const heading = document.createElement("h3");
    heading.id = headingId;
    heading.textContent = "Choose Block Path facing";
    panel.appendChild(heading);

    const pieceColor = piece ? (piece.colorName ? capitalize(piece.colorName) : (piece.color === 0 ? "White" : "Black")) : "";
    const pieceType = piece ? getPieceTypeName(piece.typeName || piece.type) : "";
    const info = document.createElement("p");
    info.id = infoId;
    info.className = "block-dir-instructions";
    info.textContent = `${pieceColor} ${pieceType} ${sqToAlg(fromSqIndex)} → ${sqToAlg(toSqIndex)}`.trim();
    panel.appendChild(info);

    const grid = document.createElement("div");
    grid.className = "block-dir-grid";
    DIRS.forEach((dir) => {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "block-dir-cone";
      button.dataset.dir = dir;
      const full = dirToFullName(dir);
      button.setAttribute("aria-label", `Face ${full}`);
      button.title = `Face ${full}`;
      button.textContent = dir;
      grid.appendChild(button);
    });
    panel.appendChild(grid);

    const cancel = document.createElement("button");
    cancel.type = "button";
    cancel.className = "block-dir-cancel";
    cancel.dataset.action = "cancel";
    cancel.textContent = "Cancel";
    panel.appendChild(cancel);

    blockDirOverlay.appendChild(panel);
    const focusOverlay = () => {
      if (blockDirOverlay && !blockDirOverlay.hidden) {
        blockDirOverlay.focus({ preventScroll: true });
      }
    };
    if (typeof queueMicrotask === "function") {
      queueMicrotask(focusOverlay);
    } else {
      Promise.resolve().then(focusOverlay);
    }
  }

  function clearBlockDirOverlay() {
    if (!blockDirOverlay) return;
    blockDirOverlay.innerHTML = "";
    blockDirOverlay.hidden = true;
    blockDirOverlay.setAttribute("aria-hidden", "true");
    blockDirOverlay.classList.remove("active");
    blockDirOverlay.removeAttribute("aria-labelledby");
    blockDirOverlay.removeAttribute("aria-describedby");
  }

  function submitPendingBlockDir(dir) {
    if (!pendingMove) return;
    pendingBlockDir = dir;
    const { from, to } = pendingMove;
    pendingMove = null;
    possibleMoves = [];
    submitMove(from, to, dir);
  }

  function cancelBlockDirSelection() {
    clearBlockDirOverlay();
    pendingMove = null;
    pendingBlockDir = "";
    selectedSquare = null;
    possibleMoves = [];
    renderBoard();
  }

  function onBlockDirOverlayClick(ev) {
    const target = ev.target instanceof Element ? ev.target : null;
    if (!target) return;
    const cancel = target.closest('[data-action="cancel"]');
    if (cancel) {
      ev.preventDefault();
      cancelBlockDirSelection();
      return;
    }
    const dirBtn = target.closest(".block-dir-cone");
    if (dirBtn && dirBtn.dataset.dir) {
      ev.preventDefault();
      const dir = dirBtn.dataset.dir;
      const buttons = blockDirOverlay ? blockDirOverlay.querySelectorAll(".block-dir-cone") : [];
      for (const btn of buttons) {
        btn.classList.remove("is-selected");
      }
      dirBtn.classList.add("is-selected");
      submitPendingBlockDir(dir);
    }
  }

  function updateMoveHints() {
    // UI-only hints: adjacent squares that are on-board and not occupied by same color
    possibleMoves = [];
    if (selectedSquare == null) return;
    const mover = state.pieces.find(p => p.square === selectedSquare);
    if (!mover) return;
    const adj = getAdjacentSquares(selectedSquare);
    possibleMoves = adj.filter(sq => {
      const occ = state.pieces.find(p => p.square === sq);
      return !occ || occ.color !== mover.color;
    });
  }

  function isPieceTurn(piece) {
    const turn = state.turn;
    const isWhiteTurn = turn === 0 || String(turn).toLowerCase() === "white";
    const isWhite = piece.color === 0 || String(piece.colorName || piece.color).toLowerCase() === "white";
    return isWhiteTurn === isWhite;
  }

  moveForm.addEventListener("submit", async (ev) => {
    ev.preventDefault();
    if (isAnimating) return;

    moveError.textContent = "";
    moveError.className = "";

    const from = (ev.target.from.value || "").trim().toLowerCase();
    const to = (ev.target.to.value || "").trim().toLowerCase();
    const fromSq = algToSq(from);
    const toSq = algToSq(to);
    const movingPiece = state.pieces.find(p => p.square === fromSq);
    if (movingPiece && needsBlockPathDirection(movingPiece) && toSq >= 0) {
      prepareBlockDirSelection(movingPiece, fromSq, toSq);
      return;
    }

    await submitMove(from, to, "");
  });

  async function submitMove(from, to, dir) {
    clearBlockDirOverlay();
    pendingMove = null;
    pendingBlockDir = "";
    isAnimating = true;
    moveForm.classList.add("loading");
    try {
      const payloadDir = typeof dir === "string" ? dir.toUpperCase() : String(dir || "");
      const result = await fetchJSON("/api/move", { from, to, dir: payloadDir });
      // Optional client animation
      await animateMove(algToSq(from), algToSq(to));
      updateState(result);
      const softMessage = result && typeof result.message === "string" ? result.message.trim() : "";
      if (softMessage) {
        showToast("Notice", softMessage);
        logEvent("Notice", softMessage);
        moveError.textContent = softMessage;
        moveError.className = "warning";
      } else {
        moveError.textContent = "";
        moveError.className = "";
      }
      sounds.move();
      // Clear selection if success
      selectedSquare = null;
      possibleMoves = [];
      moveForm.reset();
      // Move list
      addMoveToList(from, to, result);
    } catch (err) {
      showMoveError(err.message || String(err));
      sounds.error();
    } finally {
      isAnimating = false;
      moveForm.classList.remove("loading");
      renderBoard();
    }
  }

  resetBtn.addEventListener("click", async () => {
    if (isAnimating) return;
    if (!confirm("🏰 Reset the entire battle? This will clear all progress!")) return;
    clearBlockDirOverlay();
    pendingMove = null;
    pendingBlockDir = "";
    isAnimating = true;
    resetBtn.classList.add("loading");
    try {
      const result = await fetchJSON("/api/reset", {});
      updateState(result);
      selectedSquare = null;
      possibleMoves = [];
      // Clear side panels
      if (eventFeed) eventFeed.innerHTML = "";
      if (moveList) moveList.innerHTML = "";
      if (abilityToastContainer) abilityToastContainer.innerHTML = "";
      if (abilityAnnounce) abilityAnnounce.textContent = "";
      // UX message
      configMessage.textContent = "🎮 Battle arena reset! Configure both sides to begin.";
      configMessage.className = "success";
      setTimeout(() => { configMessage.textContent = ""; configMessage.className = ""; }, 2000);
    } catch (err) {
      showMoveError(err.message || String(err));
      sounds.error();
    } finally {
      isAnimating = false;
      resetBtn.classList.remove("loading");
      renderBoard();
    }
  });

  // Config submit
  configForms.forEach((form) => {
    form.addEventListener("submit", async (ev) => {
      ev.preventDefault();
      if (isAnimating) return;

      configMessage.textContent = "";
      configMessage.className = "";

      const ability = form.querySelector(".ability-select").value;
      const element = form.querySelector(".element-select").value;
      const color = form.dataset.color; // "white" | "black"

      if (!ability || !element) {
        configMessage.textContent = "⚠️ Please select both ability and element";
        configMessage.className = "error";
        return;
      }

      isAnimating = true;
      form.classList.add("loading");
      try {
        const result = await fetchJSON("/api/config", {
          color,
          abilities: [ability],
          element
        });
        updateState(result);
        // Announce
        announce(`${capitalize(color)} chose ${ability} • ${element}`);
        showToast("Loadout Set", `${capitalize(color)}: ${ability} + ${element}`);
        logEvent("Config", `${capitalize(color)} set ${ability} • ${element}`);
        if (state.locked) {
          configMessage.textContent = "⚔️ Configuration locked - battle ready!";
          configMessage.className = "success";
        }
      } catch (err) {
        configMessage.textContent = err.message || String(err);
        configMessage.className = "error";
        sounds.error();
      } finally {
        isAnimating = false;
        form.classList.remove("loading");
        setTimeout(() => { configMessage.textContent = ""; configMessage.className = ""; }, 2000);
        renderBoard();
      }
    });
  });

  // ===== State/UI =====
  function updateState(res) {
    const st = res && (res.state || res) || {};
    state = Object.assign({}, state || defaultState, st);
    state.locked = !!state.locked;
    renderBoard();
    updateConfigUI();
    updateMoveUI();
    // Handle events/notes from backend
    applyEvents(res);
  }

  function updateConfigUI() {
    const isLocked = !!state.locked;
    configForms.forEach((form) => {
      const button = form.querySelector('button[type="submit"]');
      const selects = form.querySelectorAll('select');
      if (isLocked) {
        button.disabled = true;
        button.textContent = "Game Started";
        selects.forEach((sel) => sel.disabled = true);
      } else {
        button.disabled = false;
        button.textContent = form.classList.contains("team-white") ? "🛡️ Consecrate White Forces" : "⚔️ Anoint Black Forces";
        selects.forEach((sel) => sel.disabled = false);
      }
    });
  }

  function updateMoveUI() {
    const fromInput = document.getElementById("fromInput");
    const toInput = document.getElementById("toInput");
    const submitBtn = moveForm.querySelector('button[type="submit"]');
    const abilityMap = state.abilities || {};
    const configMap = state.config || {};

    const whiteReady = Object.prototype.hasOwnProperty.call(abilityMap, "white")
      || Object.prototype.hasOwnProperty.call(configMap, "white");
    const blackReady = Object.prototype.hasOwnProperty.call(abilityMap, "black")
      || Object.prototype.hasOwnProperty.call(configMap, "black");
    const configured = whiteReady && blackReady;

    moveForm.style.display = "block";
    if (configured) {
      fromInput.disabled = false;
      toInput.disabled = false;
      submitBtn.disabled = false;
      submitBtn.textContent = state.locked ? "⚔️ Execute Move" : "⚔️ Execute Move (locks armies)";
    } else {
      fromInput.disabled = true;
      toInput.disabled = true;
      submitBtn.disabled = true;
      submitBtn.textContent = "Configure both armies to start";
    }
  }

  function showMoveError(message) {
    moveError.textContent = message;
    moveError.className = "error";
    // Small visual shake
    moveError.style.transform = "translateX(0)";
    let t = 0;
    const id = setInterval(() => {
      moveError.style.transform = `translateX(${(t%2? -1:1)*4}px)`;
      if (++t > 10) { clearInterval(id); moveError.style.transform = "translateX(0)"; }
    }, 30);
  }

  // ===== Helpers =====
  function getTurnName(turn) {
    return (turn === 0 || String(turn).toLowerCase() === "white") ? "White" : "Black";
  }
  function getPieceTypeName(t) {
    switch (String(t).toUpperCase()) {
      case "K": return "King";
      case "Q": return "Queen";
      case "R": return "Rook";
      case "B": return "Bishop";
      case "N": return "Knight";
      case "P": return "Pawn";
      default: return String(t);
    }
  }
  function dirToFullName(dir) {
    switch (String(dir || "").toUpperCase()) {
      case "N": return "North";
      case "NE": return "North East";
      case "E": return "East";
      case "SE": return "South East";
      case "S": return "South";
      case "SW": return "South West";
      case "W": return "West";
      case "NW": return "North West";
      default: return "Unknown";
    }
  }
  function sqToAlg(sq) {
    const file = sq % 8;
    const rank = Math.floor(sq / 8);
    return "abcdefgh"[file] + String(rank + 1);
  }
  function algToSq(alg) {
    if (!alg || alg.length !== 2) return -1;
    const file = "abcdefgh".indexOf(alg[0]);
    const rank = parseInt(alg[1], 10) - 1;
    if (file < 0 || rank < 0 || rank > 7) return -1;
    return rank * 8 + file;
  }
  function getAdjacentSquares(sq) {
    const file = sq % 8, rank = Math.floor(sq / 8);
    const outs = [];
    const add = (f,r) => { if (f>=0 && f<8 && r>=0 && r<8) outs.push(r*8+f); };
    add(file, rank+1); add(file+1, rank+1); add(file+1, rank);
    add(file+1, rank-1); add(file, rank-1); add(file-1, rank-1);
    add(file-1, rank); add(file-1, rank+1);
    return outs;
  }

  function hasBlockPath(piece) {
    if (!piece) return false;
    if (abilityListHasBlockPath(piece.abilityNames)) return true;
    if (abilityListHasBlockPath(piece.abilities)) return true;

    const colorKeyBase = piece.colorName
      ? String(piece.colorName).toLowerCase()
      : (piece.color === 0 ? "white" : piece.color === 1 ? "black" : String(piece.color));
    const colorKeys = [
      colorKeyBase,
      colorKeyBase && colorKeyBase.toUpperCase(),
      colorKeyBase && capitalize(colorKeyBase),
      String(piece.color)
    ].filter(Boolean);

    const abilityMap = state.abilities || {};
    for (const key of colorKeys) {
      if (abilityListHasBlockPath(abilityMap[key])) return true;
    }

    const cfgMap = state.config || {};
    for (const key of colorKeys) {
      const cfg = cfgMap[key];
      if (cfg && abilityListHasBlockPath(cfg.abilities)) return true;
    }
    return false;
  }

  function needsBlockPathDirection(piece) {
    if (!piece) return false;
    return hasBlockPath(piece);
  }

  function abilityListHasBlockPath(list) {
    if (!Array.isArray(list)) return false;
    return list.some((ability) => {
      const normalized = String(ability || "").toLowerCase().replace(/\s+/g, "");
      return normalized === "blockpath";
    });
  }

  async function animateMove(fromSq, toSq) {
    // Simple highlight flicker; real animation optional
    try {
      const fromEl = document.querySelector(`[data-sq="${fromSq}"]`);
      const toEl = document.querySelector(`[data-sq="${toSq}"]`);
      if (fromEl) { fromEl.classList.add("moving"); await wait(180); fromEl.classList.remove("moving"); }
      if (toEl) { toEl.classList.add("moving"); await wait(180); toEl.classList.remove("moving"); }
    } catch {}
  }
  function wait(ms){ return new Promise(r=>setTimeout(r,ms)); }

  async function fetchJSON(url, body) {
    const opts = body ? {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body)
    } : { method: "GET" };
    const res = await fetch(url, opts);
    let payload;
    try { payload = await res.json(); } catch { payload = {}; }
    if (!res.ok) {
      const msg = (payload && (payload.error || payload.message)) || `${res.status} ${res.statusText}`;
      throw new Error(msg);
    }
    return payload;
  }

  // ===== Event plumbing (announce, toast, log, move list) =====
  function nowHHMMSS() {
    const d = new Date();
    const s = (n) => String(n).padStart(2,"0");
    return `${s(d.getHours())}:${s(d.getMinutes())}:${s(d.getSeconds())}`;
  }
  function announce(text) {
    if (abilityAnnounce) abilityAnnounce.textContent = text || "";
  }
  function showToast(title, body) {
    if (!abilityToastContainer) return;
    let el;
    if (toastTpl && "content" in toastTpl) {
      el = toastTpl.content.firstElementChild.cloneNode(true);
      el.querySelector(".toast-title").textContent = title || "Event";
      el.querySelector(".toast-body").textContent = body || "";
    } else {
      el = document.createElement("div");
      el.className = "card";
      el.style.cssText = "padding:12px 16px; margin-bottom:10px; min-width:260px;";
      el.innerHTML = `<strong>${title || "Event"}</strong><div class="hint">${body || ""}</div>`;
    }
    abilityToastContainer.prepend(el);
    setTimeout(() => { el.remove(); }, 5000);
  }
  function logEvent(type, msg) {
    if (!eventFeed) return;
    let li;
    if (logItemTpl && "content" in logItemTpl) {
      li = logItemTpl.content.firstElementChild.cloneNode(true);
      li.querySelector(".event-time").textContent = `[${nowHHMMSS()}]`;
      li.querySelector(".event-type").textContent = type ? `${type}:` : "";
      li.querySelector(".event-msg").textContent = msg || "";
    } else {
      li = document.createElement("li");
      li.className = "event-item";
      li.textContent = `[${nowHHMMSS()}] ${type ? type + ": " : ""}${msg || ""}`;
    }
    eventFeed.prepend(li);
  }
  function addMoveToList(fromAlg, toAlg, result) {
    if (!moveList) return;
    const li = document.createElement("li");
    const caps = (result && (result.captures || result.extraCaptures)) || [];
    const san = result && (result.san || (result.move && result.move.san));
    li.textContent = san ? san : `${fromAlg}→${toAlg}${caps.length>0?" x":""}`;
    moveList.appendChild(li);
  }

  function applyEvents(res) {
    if (!res) return;
    const events = res.events || res.logs || res.abilityEvents || [];
    if (Array.isArray(events)) {
      for (const ev of events) {
        const type = (ev && (ev.type || ev.kind || ev.code)) || "Event";
        const msg  = (ev && (ev.message || ev.msg || ev.detail)) || JSON.stringify(ev);
        logEvent(type, msg);
        if (String(type).toLowerCase().includes("ability")) {
          showToast("Ability Triggered", msg);
          announce(msg);
        }
      }
    }
    const announceMsg = res.announce || res.announcement || res.note || res.lastNote;
    if (announceMsg) {
      announce(announceMsg);
      if (/ability|kill|do\s*over|block\s*path|double\s*kill|quantum/i.test(String(announceMsg))) {
        showToast("Battle Update", String(announceMsg));
      }
    }
  }

  // ===== Boot =====
  function populateConfigSelects() {
    const abilities = init.abilities || ["DoOver","BlockPath","DoubleKill","Obstinant"];
    const elements  = init.elements  || ["Light","Shadow","Fire","Water","Earth","Air","Lightning"];
    configForms.forEach((form) => {
      const abilitySelect = form.querySelector(".ability-select");
      const elementSelect = form.querySelector(".element-select");
      abilitySelect.innerHTML = "";
      elementSelect.innerHTML = "";
      const aPH = document.createElement("option");
      aPH.value = ""; aPH.textContent = "— Select ability —"; aPH.disabled = true; aPH.selected = true;
      abilitySelect.appendChild(aPH);
      const ePH = document.createElement("option");
      ePH.value = ""; ePH.textContent = "— Select element —"; ePH.disabled = true; ePH.selected = true;
      elementSelect.appendChild(ePH);
      abilities.forEach((a) => {
        const opt = document.createElement("option");
        opt.value = a; opt.textContent = a;
        abilitySelect.appendChild(opt);
      });
      elements.forEach((e) => {
        const opt = document.createElement("option");
        opt.value = e; opt.textContent = e;
        elementSelect.appendChild(opt);
      });
      abilitySelect.required = true;
      elementSelect.required = true;
    });
  }

  function capitalize(s){ return s ? s[0].toUpperCase()+s.slice(1) : s; }

  if (blockDirOverlay) {
    blockDirOverlay.addEventListener("click", onBlockDirOverlayClick);
  }
  document.addEventListener("keydown", (ev) => {
    if (ev.key === "Escape" && blockDirOverlay && !blockDirOverlay.hidden) {
      ev.preventDefault();
      cancelBlockDirSelection();
    }
  });

  populateConfigSelects();
  renderBoard();
  updateConfigUI();
  updateMoveUI();
})();
