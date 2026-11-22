// Unfurl relay for tumble's link previews.
//
// Some sites behind Cloudflare serve a challenge page to requests from
// hosting providers, so tumble's server cannot read them. Requests made
// from a Worker usually get through. tumble calls this Worker only after
// it sees a challenge (see scrapeViaWorker in internal/handler/preview.go).
//
// GET /?url=<page> with header X-Unfurl-Key: <UNFURL_KEY>
// Returns the page's status, Content-Type and body. Also cf-mitigated, so
// tumble can tell when the Worker was challenged too.

const USER_AGENT =
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36";

function keyMatches(given, expected) {
  const enc = new TextEncoder();
  const a = enc.encode(given);
  const b = enc.encode(expected);
  return a.byteLength === b.byteLength && crypto.subtle.timingSafeEqual(a, b);
}

export default {
  async fetch(request, env) {
    if (request.method !== "GET") {
      return new Response("method not allowed", { status: 405 });
    }
    if (!env.UNFURL_KEY || !keyMatches(request.headers.get("X-Unfurl-Key") ?? "", env.UNFURL_KEY)) {
      return new Response("forbidden", { status: 401 });
    }

    let target;
    try {
      target = new URL(new URL(request.url).searchParams.get("url") ?? "");
    } catch {
      return new Response("url parameter must be an absolute URL", { status: 400 });
    }
    if (target.protocol !== "http:" && target.protocol !== "https:") {
      return new Response("only http(s) URLs are allowed", { status: 400 });
    }

    const upstream = await fetch(target, {
      headers: { "User-Agent": USER_AGENT, Accept: "text/html,*/*;q=0.8" },
      redirect: "follow",
    });

    const headers = new Headers();
    for (const name of ["Content-Type", "cf-mitigated"]) {
      const value = upstream.headers.get(name);
      if (value) headers.set(name, value);
    }
    return new Response(upstream.body, { status: upstream.status, headers });
  },
};
