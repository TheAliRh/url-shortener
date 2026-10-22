package main

import "github.com/gin-gonic/gin"

const (
	// HSTS: browsers remember "HTTPS only" for this long. Add "; includeSubDomains"
	// once every subdomain serves HTTPS. Add "; preload" only if you are ready to
	// submit the domain to hstspreload.org (hard to undo).
	hstsValue = "max-age=31536000"

	// The API returns JSON and redirects, never pages, so nothing may load or embed.
	apiCSP = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"

	// For the one small HTML page the server renders (the "link disabled" page):
	// no scripts, inline styles only.
	htmlPageCSP = "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"
)

// securityHeaders sets defensive headers on every response, including 404s and
// errors. Register it first: router.Use(securityHeaders()).
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()

		h.Set("X-Content-Type-Options", "nosniff") // never guess a different content type
		h.Set("X-Frame-Options", "DENY")           // legacy twin of CSP frame-ancestors
		h.Set("Content-Security-Policy", apiCSP)
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")

		// Only meaningful (and only honored by browsers) over HTTPS. X-Forwarded-Proto
		// can be spoofed, but the worst case is an ignored header on a plain-HTTP reply.
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			h.Set("Strict-Transport-Security", hstsValue)
		}

		c.Next()
	}
}

// noStore keeps responses that carry tokens or per-user data out of browser
// and proxy caches. Apply it to the /api group, not to redirects.
func noStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}
