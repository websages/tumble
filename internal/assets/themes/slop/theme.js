// Theme: slop - everything a seed-stage landing page needs: live
// metrics, a vibe-level dial, testimonials, a trending beam card, an AI
// summarize button, a Cmd-K "Ask AI" bar (it searches), social-proof
// toasts, sparkles, and a features/pricing section under the feed.
(function () {
  var html = document.documentElement;
  var reduceMotion = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text != null) e.textContent = text;
    return e;
  }

  function pick(list) {
    return list[Math.floor(Math.random() * list.length)];
  }

  // ---------- Vibe level (per-viewer, remembered) ----------

  function setLevel(level) {
    html.setAttribute("data-slop-level", level);
    document.querySelectorAll(".slop-vibe button").forEach(function (b) {
      b.setAttribute("aria-pressed", b.getAttribute("data-slop-level") === level ? "true" : "false");
    });
    try { localStorage.setItem("slopLevel", level); } catch (e) {}
  }
  var level = "2";
  try { level = localStorage.getItem("slopLevel") || "2"; } catch (e) {}
  setLevel(level);
  document.querySelectorAll(".slop-vibe button").forEach(function (b) {
    b.addEventListener("click", function () {
      setLevel(b.getAttribute("data-slop-level"));
      sparkle(b);
    });
  });

  // ---------- Live metrics from the page ----------

  var links = document.querySelectorAll('#content .item[data-item-type="link"]');
  var quotes = document.querySelectorAll('#content .item[data-item-type="quote"]');
  var posters = [];
  links.forEach(function (item) {
    var u = item.getAttribute("data-user");
    if (u && posters.indexOf(u) === -1) posters.push(u);
  });
  document.querySelectorAll("#content .quote-author").forEach(function (a) {
    var u = a.textContent.trim();
    if (u && posters.indexOf(u) === -1) posters.push(u);
  });

  function countUp(node, to) {
    if (!node || !to) return;
    if (reduceMotion) { node.textContent = to.toLocaleString(); return; }
    var start = performance.now();
    var dur = 1200;
    (function frame(now) {
      var t = Math.min(1, (now - start) / dur);
      var eased = 1 - Math.pow(1 - t, 3);
      node.textContent = Math.round(to * eased).toLocaleString();
      if (t < 1) requestAnimationFrame(frame);
    })(start);
  }
  document.querySelectorAll('[data-slop-stat="links"]').forEach(function (n) { countUp(n, links.length); });
  document.querySelectorAll('[data-slop-stat="quotes"]').forEach(function (n) { countUp(n, quotes.length); });
  document.querySelectorAll('[data-slop-stat="posters"]').forEach(function (n) {
    if (posters.length) n.textContent = posters.length.toLocaleString();
  });

  // Avatar stack of real posters (or the usual suspects)
  var avatars = document.querySelector(".slop-avatars");
  if (avatars) {
    var faces = posters.length ? posters.slice(0, 5) : ["vc", "ai", "ceo", "dev", "pm"];
    faces.forEach(function (name, i) {
      var a = el("span", "slop-avatar slop-avatar-" + (i % 5), name.charAt(0).toUpperCase());
      a.title = name;
      avatars.appendChild(a);
    });
  }

  // ---------- Feed: trending, AI summaries, testimonials ----------

  var SUMMARIES = [
    "It's a link. You should click it. (98% confidence)",
    "TL;DR: vibes.",
    "This changes everything. Probably.",
    "Key insight: the URL starts with https.",
    "Our model read this so you don't have to. It was fine.",
    "3 takeaways: 1) link 2) posted 3) synergy.",
    "Sentiment: bullish. Moat: none. Kittens: unclear."
  ];

  document.querySelectorAll("#content .date-section").forEach(function (section) {
    var first = section.querySelector('.item[data-item-type="link"]');
    if (!first) return;
    first.classList.add("slop-trending");
    var header = first.querySelector(".link-header");
    if (header) header.insertBefore(el("span", "slop-badge slop-badge-hot slop-trending-badge", "🔥 Trending"), header.firstChild);
  });

  links.forEach(function (item) {
    var header = item.querySelector(".link-header");
    if (!header) return;
    var btn = el("button", "slop-ai-chip", "✨ Summarize");
    btn.type = "button";
    btn.addEventListener("click", function (e) {
      e.preventDefault();
      sparkle(btn);
      toast("🤖 AI summary", pick(SUMMARIES));
    });
    header.appendChild(btn);
  });

  quotes.forEach(function (item) {
    var card = item.querySelector(".quote-card");
    if (!card) return;
    card.insertBefore(el("div", "slop-quote-stars", "★★★★★"), card.firstChild);
    var author = card.querySelector(".quote-author");
    if (author) {
      var name = author.textContent.trim();
      var row = el("div", "slop-quote-person");
      row.appendChild(el("span", "slop-avatar slop-avatar-" + (name.length % 5), name.charAt(0).toUpperCase()));
      var who = el("span", "slop-quote-who");
      who.appendChild(el("span", "slop-quote-name", name));
      who.appendChild(el("span", "slop-quote-title", pick([
        "Founder & CEO, Stealth", "10x Engineer", "Chief Vibes Officer",
        "Thought Leader", "Angel Investor (2 checks)", "Head of Growth Hacking"
      ])));
      row.appendChild(who);
      row.appendChild(el("span", "slop-badge slop-badge-verified", "✓ Verified"));
      author.replaceWith(row);
    }
  });

  document.querySelectorAll("#content .kitten-card").forEach(function (card) {
    card.appendChild(el("span", "slop-badge slop-badge-cyan slop-kitten-badge", "Kitten-as-a-Service · GA"));
  });

  // ---------- Cursor spotlight on cards ----------

  document.addEventListener("pointermove", function (e) {
    var card = e.target.closest && e.target.closest(".link-card-wrapper, .quote-card, .slop-card, .slop-stat, .slop-price");
    if (!card) return;
    var r = card.getBoundingClientRect();
    card.style.setProperty("--mouse-x", (e.clientX - r.left) + "px");
    card.style.setProperty("--mouse-y", (e.clientY - r.top) + "px");
  });

  // ---------- Sparkles ----------

  function sparkle(target) {
    if (reduceMotion || !target) return;
    var r = target.getBoundingClientRect();
    for (var i = 0; i < 10; i++) {
      var s = el("span", "slop-particle", pick(["✨", "⭐", "🚀", "💸", "🦄"]));
      s.style.left = (r.left + r.width / 2) + "px";
      s.style.top = (r.top + r.height / 2) + "px";
      var angle = Math.random() * Math.PI * 2;
      var dist = 40 + Math.random() * 70;
      s.style.setProperty("--dx", Math.cos(angle) * dist + "px");
      s.style.setProperty("--dy", Math.sin(angle) * dist + "px");
      document.body.appendChild(s);
      setTimeout(s.remove.bind(s), 900);
    }
  }
  document.addEventListener("click", function (e) {
    var t = e.target.closest && e.target.closest(".slop-btn, .slop-sparkle-burst, #navigation .nav-link, #navigation-top .nav-link");
    if (t) sparkle(t);
  });

  // ---------- Toasts ----------

  var toasts = el("div", "slop-toasts");
  toasts.setAttribute("aria-live", "polite");
  document.body.appendChild(toasts);

  function toast(title, body) {
    var t = el("div", "slop-toast");
    t.appendChild(el("div", "slop-toast-title", title));
    t.appendChild(el("div", "slop-toast-body", body));
    var x = el("button", "slop-toast-close", "×");
    x.type = "button";
    x.setAttribute("aria-label", "Dismiss");
    x.addEventListener("click", function () { t.remove(); });
    t.appendChild(x);
    toasts.appendChild(t);
    setTimeout(function () { t.classList.add("slop-toast-out"); }, 5200);
    setTimeout(function () { t.remove(); }, 5800);
  }

  // Social proof, a few times per visit
  var PROOF = [
    function (n) { return [n + " just shipped a link", "📍 San Francisco · 2m ago"]; },
    function (n) { return [n + " upgraded to Enterprise", "💼 Contact Sales · just now"]; },
    function () { return ["Someone in Palo Alto raised a pre-seed", "💸 $2.5M on a napkin · 5m ago"]; },
    function (n) { return [n + " achieved 10x engineer status", "🏆 Verified by AI · 1m ago"]; },
    function () { return ["A VC is viewing this page", "👀 Partner at Hockey Stick Ventures"]; }
  ];
  var proofShown = 0;
  function proof() {
    if (proofShown++ >= 3 || document.hidden) return;
    var msg = pick(PROOF)(posters.length ? pick(posters) : "A founder");
    toast(msg[0], msg[1]);
  }
  setTimeout(proof, 6000);
  setInterval(proof, 22000);

  // ---------- Ask AI (Cmd-K): it's a search box ----------

  var PROMPTS = [
    "Ask AI: what did the team link last week?",
    "Ask AI: find me that kitten",
    "Ask AI: summarize the group chat's vibe",
    "Ask AI: generate a moat"
  ];
  var fab = el("button", "slop-ai-fab");
  fab.type = "button";
  fab.innerHTML = '<span class="slop-ai-fab-icon">✨</span> Ask tumble AI <kbd>⌘K</kbd>';
  document.body.appendChild(fab);

  var modal = el("div", "slop-ai-modal");
  modal.hidden = true;
  modal.innerHTML =
    '<form class="slop-ai-bar" action="/search" method="get" role="search">' +
    '<span class="slop-ai-bar-icon">✨</span>' +
    '<input type="text" name="search" autocomplete="off" aria-label="Ask tumble AI" />' +
    '<kbd>esc</kbd>' +
    "</form>" +
    '<div class="slop-ai-hint">Powered by a GPT wrapper · Results may be links</div>';
  document.body.appendChild(modal);
  var input = modal.querySelector("input");

  function openAI() {
    input.placeholder = pick(PROMPTS);
    modal.hidden = false;
    input.focus();
  }
  function closeAI() {
    modal.hidden = true;
  }
  fab.addEventListener("click", openAI);
  modal.addEventListener("click", function (e) {
    if (e.target === modal) closeAI();
  });
  document.addEventListener("keydown", function (e) {
    if ((e.metaKey || e.ctrlKey) && (e.key === "k" || e.key === "K")) {
      e.preventDefault();
      modal.hidden ? openAI() : closeAI();
    } else if (e.key === "Escape" && !modal.hidden) {
      closeAI();
    }
  });

  // ---------- Below the feed: the rest of the landing page ----------

  var nav = document.getElementById("navigation");
  if (!nav || !document.querySelector("#content .date-section")) return;

  var sections = el("div", "slop-sections");
  sections.innerHTML =
    '<section class="slop-section">' +
    '  <span class="slop-pill">✨ Features</span>' +
    '  <h2 class="slop-h2">Everything you need to <em class="slop-serif">10x</em> your links</h2>' +
    '  <p class="slop-section-lead">One platform. Infinite synergy. Zero product-market fit required.</p>' +
    '  <div class="slop-bento">' +
    '    <div class="slop-card slop-card-beam slop-bento-wide"><div class="slop-icon-box">🤖</div><h3>Agentic curation</h3><p>Autonomous agents post links to the group chat so your humans can focus on posting links to the group chat.</p></div>' +
    '    <div class="slop-card"><div class="slop-icon-box slop-icon-cyan">⚡</div><h3>Edge-native kittens</h3><p>A fresh kitten, served from 300+ PoPs, every day at 10 AM Central.</p></div>' +
    '    <div class="slop-card"><div class="slop-icon-box slop-icon-pink">🧠</div><h3>RAG</h3><p>Retrieval-Augmented Gossip. Every quote, indexed forever.</p></div>' +
    '    <a class="slop-card slop-bento-wide" href="/stats"><div class="slop-icon-box slop-icon-amber">📈</div><h3>Hockey-stick analytics <span class="slop-arrow">→</span></h3><p>Real-time dashboards that go up and to the right. See who is carrying the team.</p></a>' +
    '    <div class="slop-card"><div class="slop-icon-box slop-icon-emerald">🛡️</div><h3>SOC 2 Type Vibes</h3><p>Audited by a guy named Steve. He said it looked fine.</p></div>' +
    '    <div class="slop-card"><div class="slop-icon-box slop-icon-cyan">⛓️</div><h3>Blockchain-ready</h3><p>Not on a blockchain. But ready.</p></div>' +
    '    <div class="slop-card"><div class="slop-icon-box slop-icon-pink">🌱</div><h3>Carbon neutral</h3><p>We planted a tree in Minecraft for every kitten.</p></div>' +
    "  </div>" +
    "</section>" +

    '<section class="slop-section">' +
    '  <span class="slop-pill">🚀 Get started</span>' +
    '  <h2 class="slop-h2">From zero to unicorn in <em class="slop-serif">one</em> command</h2>' +
    '  <div class="slop-terminal">' +
    '    <div class="slop-terminal-bar"><i></i><i></i><i></i><span>zsh — ~/my-startup</span></div>' +
    '    <pre class="slop-terminal-body"></pre>' +
    "  </div>" +
    "</section>" +

    '<section class="slop-section">' +
    '  <span class="slop-pill">💸 Pricing</span>' +
    '  <h2 class="slop-h2">Simple, <em class="slop-serif">transparent</em> pricing</h2>' +
    '  <p class="slop-section-lead">Start free. Stay free. We make money on vibes.</p>' +
    '  <div class="slop-pricing">' +
    '    <div class="slop-price"><h3>Hobby</h3><div class="slop-price-amt">$0<span>/mo</span></div><ul><li>Unlimited links</li><li>1 daily kitten</li><li>Community support (the group chat)</li></ul><a class="slop-btn slop-btn-ghost" href="/">Start free</a></div>' +
    '    <div class="slop-price slop-price-popular"><span class="slop-badge slop-badge-popular">Most popular</span><h3>Pro</h3><div class="slop-price-amt">$0<span>/mo</span></div><ul><li>Everything in Hobby</li><li>✨ AI summaries (vibes-based)</li><li>Dark mode</li><li>A sense of belonging</li></ul><a class="slop-btn slop-btn-primary" href="/">Go Pro — free</a></div>' +
    '    <div class="slop-price"><h3>Enterprise</h3><div class="slop-price-amt">Let’s talk</div><ul><li>Everything in Pro</li><li>SSO (Slack)</li><li>Dedicated kitten</li><li>A Gartner quadrant</li></ul><a class="slop-btn slop-btn-ghost" href="/api/docs">Contact sales</a></div>' +
    "  </div>" +
    "</section>" +

    '<section class="slop-section slop-cta-band">' +
    '  <h2 class="slop-h2">Ready to <span class="slop-underline">10x</span> your links?</h2>' +
    '  <p class="slop-section-lead">No credit card required. No product either.</p>' +
    '  <div class="slop-ctas"><a class="slop-btn slop-btn-primary" href="#masthead">Get started — it’s free</a><button type="button" class="slop-btn slop-btn-ghost slop-open-ai">✨ Ask tumble AI</button></div>' +
    "</section>";
  nav.parentNode.insertBefore(sections, nav.nextSibling);
  sections.querySelector(".slop-open-ai").addEventListener("click", openAI);

  // Terminal types itself out when scrolled into view
  var LINES = [
    ["$ ", "npx create-tumble-app@latest my-startup"],
    ["", "✔ Installed 1,337 dependencies (4 of them useful)"],
    ["", "✔ Generated pitch deck (42 slides, 0 numbers)"],
    ["", "✔ Raised seed round from your uncle"],
    ["", "⚠ Generating moat... skipped"],
    ["", "✔ Fetched today's kitten"],
    ["", "🚀 Ready in 0.3s. Valuation: $4.20B"]
  ];
  var term = sections.querySelector(".slop-terminal-body");
  function typeTerminal() {
    if (reduceMotion) {
      term.textContent = LINES.map(function (l) { return l[0] + l[1]; }).join("\n");
      return;
    }
    var li = 0, ci = 0;
    (function step() {
      if (li >= LINES.length) {
        term.appendChild(el("span", "slop-caret", "▌"));
        return;
      }
      var line = LINES[li];
      if (ci === 0) {
        if (li) term.appendChild(document.createTextNode("\n"));
        if (line[0]) term.appendChild(el("span", "slop-prompt", line[0]));
      }
      if (li === 0 && ci < line[1].length) {
        term.appendChild(document.createTextNode(line[1].charAt(ci++)));
        setTimeout(step, 35);
      } else {
        if (li !== 0) term.appendChild(el("span", line[1].charAt(0) === "⚠" ? "slop-warn" : "slop-ok", line[1]));
        li++; ci = 0;
        setTimeout(step, li === 1 ? 400 : 260);
      }
    })();
  }
  if ("IntersectionObserver" in window) {
    var io = new IntersectionObserver(function (entries) {
      if (entries[0].isIntersecting) { io.disconnect(); typeTerminal(); }
    });
    io.observe(term);
  } else {
    typeTerminal();
  }
})();
