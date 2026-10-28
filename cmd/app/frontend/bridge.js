// Browser access only: Wails injects window.go and window.runtime inside the
// desktop webview, which the plain HTTP server does not provide. Forward the
// same binding calls to the App's HTTP RPC endpoint instead.
// shortcut: no event stream over HTTP, so chat updates need a reload; add SSE
// when chat has to work in the browser.
if (!window.go) {
    window.__nexflowBrowser = true;

    const rpc = (name, args) =>
        fetch('/api/rpc/' + name, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(args),
        }).then(async (res) => {
            const text = await res.text();
            if (!res.ok) throw new Error(text || res.statusText);
            return text ? JSON.parse(text) : null;
        });

    window.go = {
        application: {
            App: new Proxy({}, { get: (_, name) => (...args) => rpc(name, args) }),
        },
    };

    window.runtime = { EventsOn() {}, EventsOff() {} };
}
