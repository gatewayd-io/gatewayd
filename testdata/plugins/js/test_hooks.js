// Test hooks for gatewayd-plugin-js CI tests.
// Intercepts client traffic, logs queries, and passes through.
function onTrafficFromClient(ctx, req) {
  const msg = req.Fields["request"].GetBytesValue()
  if (String.fromCharCode(msg[0]) === "Q") {
    const query = String.fromCharCode(...msg.slice(5, -1))
    console.log("js-plugin-test: query intercepted:", query)
    const parsed = parseSQL(query)
    console.log("js-plugin-test: parsed:", parsed)
  }
  return req
}
