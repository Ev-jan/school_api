package middlewares

import "net/http"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Disables DNS prefetching (browsers won't pre-resolve domain names in links)
		// Helps reduce privacy leaks and unnecessary DNS queries
		w.Header().Set("X-DNS-Prefetch-Control", "off")

		// Prevents this page from being embedded in an <iframe> on another site
		// Mitigates clickjacking attacks
		w.Header().Set("X-Frame-Options", "DENY")

		// Enables the browser's built-in XSS filter and blocks rendering if XSS is detected
		// Helps protect against some types of cross-site scripting attacks
		w.Header().Set("X-XSS-Protection", "1;mode=block")

		// Prevents browsers from MIME-sniffing the content type
		// Ensures files are interpreted only as their declared Content-Type
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Forces all communication to use HTTPS for 2 years (63072000 seconds)
		// 'includeSubDomains' applies it to all subdomains
		// 'preload' allows inclusion in browser preload lists for HSTS
		w.Header().Set("Strict-Transport-Security", "max-age=63072000;includeSubDomains;preload")

		// Restricts where resources (scripts, images, etc.) can be loaded from
		// Here: only from the same origin ('self')
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		// Prevents the browser from sending the Referer header when navigating away
		// Protects sensitive URL information from being shared with third parties
		w.Header().Set("Referrer-Policy", "no-referrer")

		// this header tells clients what backend tech is used. U can set a wrong one to confuse individuals with malicious intents
		w.Header().Set("X-Powered-By", "Django")

		w.Header().Set("Server", "")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")

		w.Header().Set("Permissions-Policy", "geolocation=(self), microphone=()")

		// Continue processing the request with the next handler
		next.ServeHTTP(w, r)
	})
}

// BASIC MIDDLEWARE SKELETON
//func securityHeaders(next http.Handler) http.Handler {
// return http.HandleFunc(func(w http.ResponseWriter, r *http.Request){
// next.ServeHTTP(w, r)
// })
// }
