# Publish and index (what makes everything else exist)

## Deploy (Cloudflare Workers Assets, static export)

`wrangler.jsonc` with `assets.directory: "./out"` and `not_found_handling: "404-page"`.
`npx wrangler login` is interactive: if the agent has no session, ask the owner (or
`CLOUDFLARE_API_TOKEN`). The public build variable (host of the form API) must be
defined at build time, otherwise the form refuses submission on purpose. Confirm live:

```bash
curl -s https://<site>/ | grep -o '<title>[^<]*'
curl -s https://<site>/ | grep -o 'application/ld+json' | wc -l
curl -s https://<site>/robots.txt | head -3
curl -s https://<site>/sitemap.xml | grep -o '<loc>[^<]*'
curl -s -o /dev/null -w '%{http_code}\n' https://<site>/llms.txt
curl -sI https://<site>/nao-existe | head -1          # 404
curl -s -o /dev/null -w '%{http_code} %{redirect_url}\n' https://www.<site>   # 301 to root
```

## Google Search Console

**Domain** property (covers www, http, subfolders) → TXT in Cloudflare DNS → Sitemaps →
`https://<site>/sitemap.xml`. Meta tag verification (`metadata.verification.google`) is the
alternative when DNS is not at hand.

## Bing Webmaster Tools (feeds ChatGPT search, Copilot, DuckDuckGo)

1. [bing.com/webmasters](https://www.bing.com/webmasters), Microsoft or Google login.
2. **Import from Google Search Console**: authorize the same Google account and tick the sites. It brings
   the verification; **it did not bring the sitemaps** in practice. If the button does not show, add a site
   manually first.
3. **Sitemaps → Submit sitemap**: XML only. An HTML page submitted there stays "Processing" and turns into an
   error (it happened). A single page to index goes through **IndexNow** (main menu); in new accounts
   the old "URL Submission" no longer exists and "Configuration" only has Crawl Control and Block URLs.
4. The site dropdown at the top is a selector, not "add site".
5. Menus worth knowing: **AI Performance (beta)** = citations on Copilot/ChatGPT (free GEO
   measurement); **Site Scan** = Bing's technical audit (run after a few days of crawling);
   **Keyword Research**; **Backlinks**.
6. Check the exact domain of each property (`.com` vs `.com.br`): a sitemap from the wrong domain
   does not match.

## IndexNow via Cloudflare (the shortcut that was not top of mind)

Cloudflare → domain → **Caching → Configuration → Crawler Hints → On**. Free on any
plan; needs the proxy (orange cloud). On every content change Cloudflare notifies
Bing/Yandex/DuckDuckGo via IndexNow on its own: it is the automatic "submit URL" on every deploy. Google
does not take part in IndexNow (there the sitemap counts). Turn it on for every domain in the account.

## Content Signals / AI bots on Cloudflare

If the served `robots.txt` comes with a "Content Signals" block you did not write, it is Cloudflare's
managed robots (Security → Bots). With `ai-train=no` it signals AI crawlers not to
use the content, and it may be on account-wide. For a new brand, everything allowed.

Two more switches on the same domain, both defaulting to "block" on zones created since mid-2025:
**Security → Bots → "Block AI Scrapers and Crawlers"** (set Off) and **AI Crawl Control** (per-crawler
allow/block, formerly "AI Audit"). A blocked crawler gets a 403 that `robots.txt` never shows, so
confirm with `curl -A "GPTBot" -o /dev/null -w '%{http_code}' https://<site>/` (repeat for
`ClaudeBot`, `PerplexityBot`, `OAI-SearchBot`).

## Learned from a baseline run (things a generic answer got right)

- The `*.pages.dev` / `*.workers.dev` preview host is a public duplicate of the site: send
  `X-Robots-Tag: noindex` for it (`public/_headers` on Pages) or the canonical will compete with it.
- Cloudflare **Speed → Optimization → Rocket Loader: Off** for a Next.js site; it rewrites script
  loading and has broken hydration in the field.

## Afterwards

- IndexNow/Crawler Hints covers Bing on every deploy; Google takes days. Requesting indexing of the root in
  Search Console speeds it up.
- 301 redirect `www` → root and `.com.br` → `.com` (or the reverse) consistent with the canonical.
- Field CWV (PageSpeed Insights mobile) after a few days.
