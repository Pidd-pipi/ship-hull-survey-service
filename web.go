package main

import (
	"net/http"
)

// staticHandler serves the single-page survey console assets. The index page
// is served only at the exact root path so that misspelled or stale URLs are
// reported as 404 instead of silently rendering the app shell, which would
// hide routing mistakes and confuse clients that probe for real resources.
//
// Cache headers are intentionally short and revalidation-friendly: the index
// must always be re-fetched (it points at the current app bundle), while the
// script is allowed a brief browser cache so deploys take effect quickly
// without forcing a full re-download on every navigation.
func staticHandler() http.Handler {
	const index = `<!doctype html><html><head><meta charset="utf-8"><title>Hull survey</title></head><body><h1>Ship hull survey</h1><p id="health">Loading...</p><button id="refresh">Refresh findings</button><ul id="items"></ul><script src="/app.js"></script></body></html>`
	const app = `const list=document.querySelector('#items');async function load(){const xs=await (await fetch('/api/findings')).json();list.innerHTML=xs.map(x=>'<li>'+x.vessel+' / '+x.zone+' - '+x.finding+' ('+x.status+') <button data-id="'+x.id+'">Close</button></li>').join('');document.querySelectorAll('[data-id]').forEach(b=>b.onclick=async()=>{await fetch('/api/findings/'+b.dataset.id+'/status',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({status:'closed'})});load()})}fetch('/healthz').then(r=>r.json()).then(x=>document.querySelector('#health').textContent=x.status+' / '+x.service);document.querySelector('#refresh').onclick=load;load();`
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// nosniff hardens the response against content-type confusion.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		switch r.URL.Path {
		case "/", "":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			_, _ = w.Write([]byte(index))
		case "/app.js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=300, must-revalidate")
			_, _ = w.Write([]byte(app))
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	})
}
